package bitget

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/tigusigalpa/bitget-go/models"
)

// Public/private WebSocket endpoints.
//
// Docs: https://www.bitget.com/api-doc/uta/guide
const (
	DefaultPublicWSURL  = "wss://ws.bitget.com/v3/ws/public"
	DefaultPrivateWSURL = "wss://ws.bitget.com/v3/ws/private"
	DemoPublicWSURL     = "wss://wspap.bitget.com/v3/ws/public"
	DemoPrivateWSURL    = "wss://wspap.bitget.com/v3/ws/private"
)

const (
	wsPingInterval   = 25 * time.Second
	wsPongTimeout    = 30 * time.Second
	wsWriteTimeout   = 10 * time.Second
	wsLoginTimeout   = 10 * time.Second
	wsReconnectMin   = 1 * time.Second
	wsReconnectMax   = 60 * time.Second
	wsSubBufferSize  = 100
	wsMaxMessageSize = 10 << 20
)

// WSOption configures a WSClient at construction time.
type WSOption func(*WSClient)

// RawFrame is an exact WebSocket message received from Bitget before the SDK
// decodes it. Payload is a private copy and must be treated as read-only.
// Generation increases after every successful connection; ReceivedAt is taken
// immediately after the network read completes.
type RawFrame struct {
	Payload    []byte
	ReceivedAt time.Time
	Generation uint64
}

// RawFrameHandler observes every received WebSocket message synchronously.
// It is the lossless canonical path for consumers that need exact wire data.
// Keep it short or apply backpressure deliberately. Returning an error emits
// a terminal lifecycle event and disconnects the current connection.
type RawFrameHandler func(RawFrame) error

// WSLifecycleType identifies a WebSocket connection or subscription event.
type WSLifecycleType string

// WSLifecycle event types are delivered to a WSEventHandler.
const (
	WSConnected               WSLifecycleType = "connected"
	WSDisconnected            WSLifecycleType = "disconnected"
	WSClosed                  WSLifecycleType = "closed"
	WSAuthenticated           WSLifecycleType = "authenticated"
	WSSubscriptionRequested   WSLifecycleType = "subscription_requested"
	WSSubscribed              WSLifecycleType = "subscribed"
	WSUnsubscriptionRequested WSLifecycleType = "unsubscription_requested"
	WSUnsubscribed            WSLifecycleType = "unsubscribed"
	WSProviderError           WSLifecycleType = "provider_error"
	WSTerminal                WSLifecycleType = "terminal"
)

// WSLifecycleEvent reports the observable control-plane lifecycle. Arg is
// copied from Bitget's control event, allowing callers to correlate an ACK or
// provider error to a subscription. Err is non-nil for local terminal errors.
type WSLifecycleEvent struct {
	Type       WSLifecycleType
	Arg        *models.WSArg
	Code       string
	Message    string
	Err        error
	ReceivedAt time.Time
	Generation uint64
}

// WSEventHandler handles lifecycle events synchronously. It must return
// promptly and must not panic; the SDK does not buffer or drop these events.
type WSEventHandler func(WSLifecycleEvent)

// WithWSURL overrides the WebSocket endpoint, e.g. for Bitget's demo/paper
// trading WS or the Lo-La VIP endpoint.
func WithWSURL(url string) WSOption {
	return func(c *WSClient) { c.url = url }
}

// WithWSLogger sets a structured logger for connection lifecycle events.
func WithWSLogger(l Logger) WSOption {
	return func(c *WSClient) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithRawFrameHandler installs the canonical, synchronous raw-frame observer.
// It receives a full immutable copy before JSON decoding and before the
// bounded WSPush subscriber buffers are considered.
func WithRawFrameHandler(handler RawFrameHandler) WSOption {
	return func(c *WSClient) { c.rawFrameHandler = handler }
}

// WithWSEventHandler installs a synchronous observer for connection and
// subscription lifecycle events, including subscribe ACKs and provider errors.
func WithWSEventHandler(handler WSEventHandler) WSOption {
	return func(c *WSClient) { c.eventHandler = handler }
}

// WithWSAutoReconnect toggles automatic reconnection with exponential
// backoff on unexpected disconnects. Enabled by default.
func WithWSAutoReconnect(enabled bool) WSOption {
	return func(c *WSClient) { c.autoReconnect = enabled }
}

// wsSubscription tracks one active channel subscription: the args used to
// (re)subscribe, and the channel data pushes are delivered on.
type wsSubscription struct {
	mu     sync.RWMutex
	arg    models.WSArg
	ch     chan models.WSPush
	closed bool
}

// WSClient is a reconnecting WebSocket client for Bitget's public or
// private UTA v3 streams. Create one with NewPublicWSClient or
// NewPrivateWSClient.
//
// Docs: https://www.bitget.com/api-doc/uta/websocket/private/Fast-Fill-Channel
type WSClient struct {
	url        string
	apiKey     string
	secretKey  string
	passphrase string
	private    bool

	logger          Logger
	autoReconnect   bool
	rawFrameHandler RawFrameHandler
	eventHandler    WSEventHandler

	mu            sync.RWMutex
	writeMu       sync.Mutex
	conn          *websocket.Conn
	subscriptions map[string]*wsSubscription
	closed        bool
	active        bool
	reconnecting  bool
	done          chan struct{}
	loginWait     chan error
	lastPong      time.Time
	generation    uint64
}

// NewPublicWSClient creates a client for Bitget's public market-data
// WebSocket streams (no authentication).
func NewPublicWSClient(opts ...WSOption) *WSClient {
	c := &WSClient{
		url:           DefaultPublicWSURL,
		logger:        noopLogger{},
		autoReconnect: true,
		subscriptions: make(map[string]*wsSubscription),
		done:          make(chan struct{}),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewPrivateWSClient creates a client for Bitget's private WebSocket
// streams (order fills, positions, etc.), authenticating automatically on
// Connect.
func NewPrivateWSClient(apiKey, secretKey, passphrase string, opts ...WSOption) *WSClient {
	c := &WSClient{
		url:           DefaultPrivateWSURL,
		apiKey:        apiKey,
		secretKey:     secretKey,
		passphrase:    passphrase,
		private:       true,
		logger:        noopLogger{},
		autoReconnect: true,
		subscriptions: make(map[string]*wsSubscription),
		done:          make(chan struct{}),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func wsSubKey(arg models.WSArg) string {
	return fmt.Sprintf("%s:%s:%s:%s", arg.InstType, arg.Topic, arg.Symbol, arg.Coin)
}

// wsLoginSign signs the WebSocket login pre-hash: timestamp + "GET" +
// "/user/verify".
//
// Docs: https://www.bitget.com/api-doc/uta/guide
func wsLoginSign(secretKey, timestamp string) string {
	prehash := timestamp + "GET" + "/user/verify"
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(prehash))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// Connect dials the WebSocket endpoint, logs in (for private clients), and
// starts the background read/ping/reconnect loops. Connect blocks until the
// initial connection (and, for private clients, login) succeeds or ctx is
// done.
func (c *WSClient) Connect(ctx context.Context) error {
	c.mu.RLock()
	closed := c.closed
	connected := c.conn != nil
	c.mu.RUnlock()
	if closed {
		return fmt.Errorf("bitget: websocket client is closed")
	}
	if connected {
		return fmt.Errorf("bitget: websocket already connected")
	}
	return c.establish(ctx, true)
}

func (c *WSClient) establish(ctx context.Context, startPing bool) error {
	if err := c.dial(ctx); err != nil {
		return err
	}

	if c.private {
		go c.readPump()
		if err := c.login(ctx); err != nil {
			c.closeCurrentConnection()
			return err
		}
	}

	c.mu.Lock()
	if c.closed || c.conn == nil {
		c.mu.Unlock()
		return fmt.Errorf("bitget: websocket client is closed")
	}
	c.active = true
	c.mu.Unlock()
	if !c.private {
		go c.readPump()
	}
	if startPing {
		go c.pingPump()
	}
	return nil
}

func (c *WSClient) dial(ctx context.Context) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.url, nil)
	if err != nil {
		return fmt.Errorf("bitget: websocket dial: %w", err)
	}
	conn.SetReadLimit(wsMaxMessageSize)

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		_ = conn.Close()
		return fmt.Errorf("bitget: websocket client is closed")
	}
	if c.conn != nil {
		c.mu.Unlock()
		_ = conn.Close()
		return fmt.Errorf("bitget: websocket already connected")
	}
	c.conn = conn
	c.lastPong = time.Now()
	c.generation++
	generation := c.generation
	c.mu.Unlock()
	c.logger.Info("bitget: websocket connected", "url", safeWebSocketURL(c.url))
	c.notifyEvent(WSLifecycleEvent{Type: WSConnected, ReceivedAt: time.Now(), Generation: generation})
	return nil
}

// safeWebSocketURL removes a custom endpoint's user info, query string, and
// fragment before it reaches a logger. A query string is not needed to
// identify a Bitget endpoint and can contain a credential when callers use a
// proxy or a custom gateway.
func safeWebSocketURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "[invalid WebSocket URL]"
	}
	u.User = nil
	u.RawQuery = ""
	u.ForceQuery = false
	u.Fragment = ""
	return u.String()
}

func (c *WSClient) login(ctx context.Context) error {
	loginWait := make(chan error, 1)
	c.mu.Lock()
	c.loginWait = loginWait
	c.mu.Unlock()
	defer c.clearLoginWait(loginWait)

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	req := models.WSLoginRequest{
		Op: "login",
		Args: []models.WSLoginArg{{
			APIKey:     c.apiKey,
			Passphrase: c.passphrase,
			Timestamp:  timestamp,
			Sign:       wsLoginSign(c.secretKey, timestamp),
		}},
	}
	if err := c.writeJSON(req); err != nil {
		return err
	}

	timer := time.NewTimer(wsLoginTimeout)
	defer timer.Stop()
	select {
	case err := <-loginWait:
		if err != nil {
			return err
		}
		c.logger.Info("bitget: websocket login succeeded")
		return nil
	case <-timer.C:
		return fmt.Errorf("bitget: websocket login timed out")
	case <-ctx.Done():
		return ctx.Err()
	case <-c.done:
		return fmt.Errorf("bitget: websocket client is closed")
	}
}

func (c *WSClient) clearLoginWait(loginWait chan error) {
	c.mu.Lock()
	if c.loginWait == loginWait {
		c.loginWait = nil
	}
	c.mu.Unlock()
}

func (c *WSClient) completeLogin(err error) {
	c.mu.RLock()
	loginWait := c.loginWait
	c.mu.RUnlock()
	if loginWait == nil {
		return
	}
	select {
	case loginWait <- err:
	default:
	}
}

func (c *WSClient) writeJSON(v interface{}) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.mu.RLock()
	conn := c.conn
	closed := c.closed
	c.mu.RUnlock()
	if closed || conn == nil {
		return fmt.Errorf("bitget: websocket not connected")
	}
	_ = conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
	return conn.WriteJSON(v)
}

func (c *WSClient) writeMessage(messageType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.mu.RLock()
	conn := c.conn
	closed := c.closed
	c.mu.RUnlock()
	if closed || conn == nil {
		return fmt.Errorf("bitget: websocket not connected")
	}
	_ = conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
	return conn.WriteMessage(messageType, data)
}

// Subscribe subscribes to a channel and returns a buffered channel of data
// pushes. The subscription is automatically restored after a reconnect.
func (c *WSClient) Subscribe(ctx context.Context, arg models.WSArg) (<-chan models.WSPush, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key := wsSubKey(arg)
	c.mu.Lock()
	sub, exists := c.subscriptions[key]
	if exists {
		c.mu.Unlock()
		return sub.ch, nil
	}
	sub = &wsSubscription{arg: arg, ch: make(chan models.WSPush, wsSubBufferSize)}
	c.subscriptions[key] = sub
	c.mu.Unlock()

	req := models.WSSubscribeRequest{Op: "subscribe", Args: []models.WSArg{arg}}
	if err := c.writeJSON(req); err != nil {
		c.mu.Lock()
		if c.subscriptions[key] == sub {
			delete(c.subscriptions, key)
		}
		c.mu.Unlock()
		sub.close()
		return nil, err
	}
	c.notifyEvent(WSLifecycleEvent{
		Type:       WSSubscriptionRequested,
		Arg:        copyWSArg(&arg),
		ReceivedAt: time.Now(),
		Generation: c.currentGeneration(),
	})
	return sub.ch, nil
}

// Unsubscribe removes a channel subscription and closes its data channel.
func (c *WSClient) Unsubscribe(arg models.WSArg) error {
	key := wsSubKey(arg)

	c.mu.Lock()
	sub, exists := c.subscriptions[key]
	if exists {
		delete(c.subscriptions, key)
	}
	c.mu.Unlock()

	if !exists {
		return nil
	}
	sub.close()

	req := models.WSSubscribeRequest{Op: "unsubscribe", Args: []models.WSArg{arg}}
	if err := c.writeJSON(req); err != nil {
		return err
	}
	c.notifyEvent(WSLifecycleEvent{
		Type:       WSUnsubscriptionRequested,
		Arg:        copyWSArg(&arg),
		ReceivedAt: time.Now(),
		Generation: c.currentGeneration(),
	})
	return nil
}

func (s *wsSubscription) send(push models.WSPush) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return false
	}
	select {
	case s.ch <- push:
		return true
	default:
		return false
	}
}

func (s *wsSubscription) close() {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		close(s.ch)
	}
	s.mu.Unlock()
}

// Close terminates the connection and stops all background loops. Safe to
// call multiple times.
func (c *WSClient) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.active = false
	conn := c.conn
	c.conn = nil
	done := c.done
	subscriptions := make([]*wsSubscription, 0, len(c.subscriptions))
	for _, sub := range c.subscriptions {
		subscriptions = append(subscriptions, sub)
	}
	c.mu.Unlock()

	close(done)
	for _, sub := range subscriptions {
		sub.close()
	}
	if conn != nil {
		err := conn.Close()
		c.notifyEvent(WSLifecycleEvent{Type: WSClosed, ReceivedAt: time.Now(), Generation: c.currentGeneration()})
		return err
	}
	c.notifyEvent(WSLifecycleEvent{Type: WSClosed, ReceivedAt: time.Now(), Generation: c.currentGeneration()})
	return nil
}

func (c *WSClient) pingPump() {
	c.mu.RLock()
	done := c.done
	c.mu.RUnlock()
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			c.mu.RLock()
			conn := c.conn
			lastPong := c.lastPong
			closed := c.closed
			c.mu.RUnlock()
			if closed || conn == nil {
				continue
			}
			if time.Since(lastPong) > wsPongTimeout {
				c.logger.Warn("bitget: websocket pong timed out")
				if current, active := c.disconnect(conn); current && active {
					c.startReconnect()
				}
				continue
			}
			if err := c.writeMessage(websocket.TextMessage, []byte("ping")); err != nil {
				c.logger.Warn("bitget: websocket ping failed", "error", err)
				if current, active := c.disconnect(conn); current && active {
					c.startReconnect()
				}
			}
		}
	}
}

func (c *WSClient) readPump() {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			c.logger.Warn("bitget: websocket read error", "error", err)
			c.completeLogin(fmt.Errorf("bitget: websocket read: %w", err))
			if current, active := c.disconnect(conn); current && active {
				c.startReconnect()
			}
			return
		}

		receivedAt := time.Now()
		if err := c.observeRawFrame(message, receivedAt); err != nil {
			terminalErr := fmt.Errorf("bitget: raw frame handler: %w", err)
			c.logger.Error("bitget: websocket raw frame handler failed", "error", terminalErr)
			c.completeLogin(terminalErr)
			c.notifyEvent(WSLifecycleEvent{
				Type:       WSTerminal,
				Err:        terminalErr,
				ReceivedAt: receivedAt,
				Generation: c.currentGeneration(),
			})
			if current, active := c.disconnect(conn); current && active {
				c.startReconnect()
			}
			return
		}

		if string(message) == "pong" {
			c.mu.Lock()
			if c.conn == conn {
				c.lastPong = time.Now()
			}
			c.mu.Unlock()
			continue
		}

		c.handleMessage(message)
	}
}

func (c *WSClient) disconnect(conn *websocket.Conn) (bool, bool) {
	c.mu.Lock()
	if c.conn != conn {
		c.mu.Unlock()
		return false, false
	}
	active := c.active
	generation := c.generation
	c.conn = nil
	c.active = false
	c.mu.Unlock()
	_ = conn.Close()
	c.notifyEvent(WSLifecycleEvent{Type: WSDisconnected, ReceivedAt: time.Now(), Generation: generation})
	return true, active
}

func (c *WSClient) closeCurrentConnection() {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn != nil {
		c.disconnect(conn)
	}
}

func (c *WSClient) startReconnect() {
	c.mu.Lock()
	if c.closed || !c.autoReconnect || c.reconnecting {
		c.mu.Unlock()
		return
	}
	c.reconnecting = true
	c.mu.Unlock()
	go c.reconnectLoop()
}

func (c *WSClient) handleMessage(message []byte) {
	var event models.WSEvent
	if err := json.Unmarshal(message, &event); err == nil && event.Event != "" {
		switch event.Event {
		case "login":
			if event.Code != "" && event.Code != "0" {
				loginErr := fmt.Errorf("bitget: websocket login failed: code=%s, message=%s", event.Code, event.Msg)
				c.completeLogin(loginErr)
				c.notifyEvent(c.lifecycleEvent(WSProviderError, event, loginErr))
				return
			}
			c.completeLogin(nil)
			c.notifyEvent(c.lifecycleEvent(WSAuthenticated, event, nil))
		case "subscribe":
			if event.Code != "" && event.Code != "0" {
				providerErr := fmt.Errorf("bitget: websocket subscribe failed: code=%s, message=%s", event.Code, event.Msg)
				c.removeSubscription(event.Arg)
				c.notifyEvent(c.lifecycleEvent(WSProviderError, event, providerErr))
				return
			}
			c.notifyEvent(c.lifecycleEvent(WSSubscribed, event, nil))
		case "unsubscribe":
			if event.Code != "" && event.Code != "0" {
				providerErr := fmt.Errorf("bitget: websocket unsubscribe failed: code=%s, message=%s", event.Code, event.Msg)
				c.notifyEvent(c.lifecycleEvent(WSProviderError, event, providerErr))
				return
			}
			c.notifyEvent(c.lifecycleEvent(WSUnsubscribed, event, nil))
		case "error":
			loginErr := fmt.Errorf("bitget: websocket error event: code=%s, message=%s", event.Code, event.Msg)
			c.completeLogin(loginErr)
			c.removeSubscription(event.Arg)
			c.notifyEvent(c.lifecycleEvent(WSProviderError, event, loginErr))
			c.logger.Error("bitget: websocket error event", "code", event.Code, "msg", event.Msg)
		}
		return
	}

	var push models.WSPush
	if err := json.Unmarshal(message, &push); err != nil {
		c.logger.Warn("bitget: websocket decode push failed", "error", err)
		return
	}

	key := wsSubKey(push.Arg)
	c.mu.RLock()
	sub, ok := c.subscriptions[key]
	c.mu.RUnlock()
	if !ok {
		return
	}
	if !sub.send(push) {
		c.logger.Warn("bitget: websocket subscriber channel full, dropping message", "channel", key)
	}
}

func (c *WSClient) reconnectLoop() {
	c.mu.RLock()
	done := c.done
	c.mu.RUnlock()
	defer func() {
		c.mu.Lock()
		c.reconnecting = false
		c.mu.Unlock()
	}()

	backoff := wsReconnectMin
	for {
		timer := time.NewTimer(backoff)
		select {
		case <-done:
			timer.Stop()
			return
		case <-timer.C:
		}

		ctx, cancel := context.WithTimeout(context.Background(), wsLoginTimeout)
		err := c.establish(ctx, false)
		cancel()
		if err != nil {
			c.logger.Warn("bitget: websocket reconnect failed", "error", err, "backoff", backoff)
			backoff *= 2
			if backoff > wsReconnectMax {
				backoff = wsReconnectMax
			}
			continue
		}

		c.resubscribeAll()
		c.logger.Info("bitget: websocket reconnected")
		return
	}
}

func (c *WSClient) resubscribeAll() {
	c.mu.RLock()
	args := make([]models.WSArg, 0, len(c.subscriptions))
	for _, sub := range c.subscriptions {
		args = append(args, sub.arg)
	}
	c.mu.RUnlock()

	if len(args) == 0 {
		return
	}
	req := models.WSSubscribeRequest{Op: "subscribe", Args: args}
	if err := c.writeJSON(req); err != nil {
		c.logger.Warn("bitget: websocket resubscribe failed", "error", err)
		return
	}
	for _, arg := range args {
		c.notifyEvent(WSLifecycleEvent{
			Type:       WSSubscriptionRequested,
			Arg:        copyWSArg(&arg),
			ReceivedAt: time.Now(),
			Generation: c.currentGeneration(),
		})
	}
}

func (c *WSClient) observeRawFrame(payload []byte, receivedAt time.Time) error {
	if c.rawFrameHandler == nil {
		return nil
	}
	frame := RawFrame{
		Payload:    append([]byte(nil), payload...),
		ReceivedAt: receivedAt,
		Generation: c.currentGeneration(),
	}
	return c.rawFrameHandler(frame)
}

func (c *WSClient) lifecycleEvent(eventType WSLifecycleType, event models.WSEvent, err error) WSLifecycleEvent {
	return WSLifecycleEvent{
		Type:       eventType,
		Arg:        copyWSArg(event.Arg),
		Code:       event.Code,
		Message:    event.Msg,
		Err:        err,
		ReceivedAt: time.Now(),
		Generation: c.currentGeneration(),
	}
}

func (c *WSClient) notifyEvent(event WSLifecycleEvent) {
	if c.eventHandler != nil {
		c.eventHandler(event)
	}
}

func (c *WSClient) currentGeneration() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.generation
}

func copyWSArg(arg *models.WSArg) *models.WSArg {
	if arg == nil {
		return nil
	}
	copy := *arg
	return &copy
}

func (c *WSClient) removeSubscription(arg *models.WSArg) {
	if arg == nil {
		return
	}
	key := wsSubKey(*arg)
	c.mu.Lock()
	sub, ok := c.subscriptions[key]
	if ok {
		delete(c.subscriptions, key)
	}
	c.mu.Unlock()
	if ok {
		sub.close()
	}
}
