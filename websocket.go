package bitget

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

	logger        Logger
	autoReconnect bool

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
	c.mu.Unlock()
	c.logger.Info("bitget: websocket connected", "url", c.url)
	return nil
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
	return c.writeJSON(req)
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
		return conn.Close()
	}
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
	c.conn = nil
	c.active = false
	c.mu.Unlock()
	_ = conn.Close()
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
				c.completeLogin(fmt.Errorf("bitget: websocket login failed: code=%s, message=%s", event.Code, event.Msg))
				return
			}
			c.completeLogin(nil)
		case "error":
			loginErr := fmt.Errorf("bitget: websocket error event: code=%s, message=%s", event.Code, event.Msg)
			c.completeLogin(loginErr)
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
	}
}
