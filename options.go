package bitget

import (
	"log/slog"
	"net/http"
	"time"
)

// DefaultBaseURL is Bitget's production UTA v3 REST endpoint.
//
// Docs: https://www.bitget.com/api-doc/uta/guide
const DefaultBaseURL = "https://api.bitget.com"

// DefaultTimeout is applied to the internal *http.Client when none is
// supplied via WithHTTPClient.
const DefaultTimeout = 15 * time.Second

// Logger is a minimal structured-logging interface. Pass an adapter (see
// NewSlogLogger) or your own implementation via WithLogger; the default is
// a no-op logger.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Debug(string, ...any) {}
func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

// slogLogger adapts the standard library's *slog.Logger to the Logger
// interface.
type slogLogger struct{ l *slog.Logger }

// NewSlogLogger wraps l so it can be passed to WithLogger.
func NewSlogLogger(l *slog.Logger) Logger { return &slogLogger{l: l} }

func (s *slogLogger) Debug(msg string, args ...any) { s.l.Debug(msg, args...) }
func (s *slogLogger) Info(msg string, args ...any)  { s.l.Info(msg, args...) }
func (s *slogLogger) Warn(msg string, args ...any)  { s.l.Warn(msg, args...) }
func (s *slogLogger) Error(msg string, args ...any) { s.l.Error(msg, args...) }

// Option configures a Client at construction time.
type Option func(*Client)

// WithHTTPClient injects a custom *http.Client (for proxies, custom
// transports, or test doubles). The client is used as-is; DefaultTimeout is
// not applied when this option is used.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithBaseURL overrides the REST base URL, e.g. for Bitget's Lo-La
// (VIP/institutional) endpoint or a test server.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = baseURL }
}

// WithDemoTrading enables Bitget's simulated-trading mode by sending the
// "paptrading: 1" header on every request. Use together with a Demo API
// key; see https://www.bitget.com/api-doc/uta/guide.
func WithDemoTrading() Option {
	return func(c *Client) { c.demoTrading = true }
}

// WithTimeout sets the timeout of the internally constructed *http.Client.
// Ignored if WithHTTPClient is also used.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// WithLogger sets a structured logger for request/response diagnostics.
// Never logs ACCESS-KEY, ACCESS-SIGN, or ACCESS-PASSPHRASE values.
func WithLogger(l Logger) Option {
	return func(c *Client) { c.logger = l }
}

// WithLocale sets the "locale" header sent on every request (e.g. "en-US",
// "zh-CN"). Defaults to "en-US".
func WithLocale(locale string) Option {
	return func(c *Client) { c.locale = locale }
}
