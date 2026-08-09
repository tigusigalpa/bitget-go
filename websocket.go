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
	wsPingInterval  = 25 * time.Second
	wsPongTimeout   = 30 * time.Second
	wsWriteTimeout  = 10 * time.Second
	wsLoginTimeout  = 10 * time.Second
	wsReconnectMin  = 1 * time.Second
	wsReconnectMax  = 60 * time.Second
	wsSubBufferSize = 100
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
	return func(c *WSClient) { c.logger = l }
}

// WithWSAutoReconnect toggles automatic reconnection with exponential
// backoff on unexpected disconnects. Enabled by default.
func WithWSAutoReconnect(enabled bool) WSOption {
	return func(c *WSClient) { c.autoReconnect = enabled }
}

// wsSubscription tracks one active channel subscription: the args used to
// (re)subscribe, and the channel data pushes are delivered on.
type wsSubscription struct {
	arg models.WSArg
	ch  chan models.WSPush
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
	conn          *websocket.Conn
	subscriptions map[string]*wsSubscription
	closed        bool
	done          chan struct{}
	loggedIn      chan struct{}
}

// NewPublicWSClient creates a client for Bitget's public market-data
// WebSocket streams (no authentication).
func NewPublicWSClient(opts ...WSOption) *WSClient {
	c := &WSClient{
		url:           DefaultPublicWSURL,
		logger:        noopLogger{},
		autoReconnect: true,
		subscriptions: make(map[string]*wsSubscription),
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
	c.mu.Lock()
	c.closed = false
	c.done = make(chan struct{})
	c.mu.Unlock()

	if err := c.dial(ctx); err != nil {
		return err
	}

	go c.readPump()
	go c.pingPump()

	if c.private {
		if err := c.login(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *WSClient) dial(ctx context.Context) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.url, nil)
	if err != nil {
		return fmt.Errorf("bitget: websocket dial: %w", err)
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	c.logger.Info("bitget: websocket connected", "url", c.url)
	return nil
}

func (c *WSClient) login(ctx context.Context) error {
	c.mu.Lock()
	c.loggedIn = make(chan struct{})
	c.mu.Unlock()

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	req := models.WSLoginRequest{
		Op: "login",
		Args: []models.WSLoginArg{{
			ApiKey:     c.apiKey,
			Passphrase: c.passphrase,
			Timestamp:  timestamp,
			Sign:       wsLoginSign(c.secretKey, timestamp),
		}},
	}
	if err := c.writeJSON(req); err != nil {
		return err
	}

	select {
	case <-c.loggedIn:
		c.logger.Info("bitget: websocket login succeeded")
		return nil
	case <-time.After(wsLoginTimeout):
		return fmt.Errorf("bitget: websocket login timed out")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *WSClient) writeJSON(v interface{}) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return fmt.Errorf("bitget: websocket not connected")
	}
	_ = conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
	return conn.WriteJSON(v)
}

// Subscribe subscribes to a channel and returns a buffered channel of data
// pushes. The subscription is automatically restored after a reconnect.
func (c *WSClient) Subscribe(ctx context.Context, arg models.WSArg) (<-chan models.WSPush, error) {
	key := wsSubKey(arg)

	c.mu.Lock()
	sub, exists := c.subscriptions[key]
	if !exists {
		sub = &wsSubscription{arg: arg, ch: make(chan models.WSPush, wsSubBufferSize)}
		c.subscriptions[key] = sub
	}
	c.mu.Unlock()

	req := models.WSSubscribeRequest{Op: "subscribe", Args: []models.WSArg{arg}}
	if err := c.writeJSON(req); err != nil {
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
	close(sub.ch)

	req := models.WSSubscribeRequest{Op: "unsubscribe", Args: []models.WSArg{arg}}
	return c.writeJSON(req)
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
	conn := c.conn
	done := c.done
	c.mu.Unlock()

	if done != nil {
		close(done)
	}
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (c *WSClient) pingPump() {
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()
			if conn == nil {
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
				c.logger.Warn("bitget: websocket ping failed", "error", err)
			}
		}
	}
}

func (c *WSClient) readPump() {
	for {
		c.mu.RLock()
		conn := c.conn
		closed := c.closed
		c.mu.RUnlock()
		if closed || conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			c.logger.Warn("bitget: websocket read error", "error", err)
			if c.autoReconnect && !c.isClosed() {
				go c.reconnectLoop()
			}
			return
		}

		if string(message) == "pong" {
			continue
		}

		c.handleMessage(message)
	}
}

func (c *WSClient) isClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

func (c *WSClient) handleMessage(message []byte) {
	var event models.WSEvent
	if err := json.Unmarshal(message, &event); err == nil && event.Event != "" {
		switch event.Event {
		case "login":
			c.mu.RLock()
			loggedIn := c.loggedIn
			c.mu.RUnlock()
			if loggedIn != nil {
				select {
				case <-loggedIn:
				default:
					close(loggedIn)
				}
			}
		case "error":
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
	select {
	case sub.ch <- push:
	default:
		c.logger.Warn("bitget: websocket subscriber channel full, dropping message", "channel", key)
	}
}

func (c *WSClient) reconnectLoop() {
	backoff := wsReconnectMin
	for {
		if c.isClosed() {
			return
		}
		time.Sleep(backoff)

		ctx, cancel := context.WithTimeout(context.Background(), wsLoginTimeout)
		err := c.Connect(ctx)
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
