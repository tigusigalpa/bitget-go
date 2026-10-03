# Bitget Golang Client/SDK/Library

![Bitget Golang SDK](https://i.postimg.cc/j2ZkYg04/bitget-golang-github.jpg)

[![CI](https://github.com/tigusigalpa/bitget-go/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/bitget-go/actions/workflows/ci.yml)
[![Tests](https://github.com/tigusigalpa/bitget-go/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/bitget-go/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)](LICENSE)
[![CodeQL](https://github.com/tigusigalpa/bitget-go/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/bitget-go/actions/workflows/codeql.yml)
[![Codecov](https://codecov.io/gh/tigusigalpa/bitget-go/graph/badge.svg)](https://codecov.io/gh/tigusigalpa/bitget-go)
[![GitHub Release](https://img.shields.io/github/v/release/tigusigalpa/bitget-go?style=flat-square)](https://github.com/tigusigalpa/bitget-go/releases)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue?style=flat-square&logo=go)](https://pkg.go.dev/github.com/tigusigalpa/bitget-go)

An intentionally small, typed Go client for the [Bitget Unified Trading Account (UTA) API v3](https://www.bitget.com/docs/uta/quick-start). It takes care of request signing, response envelopes, WebSocket lifecycle, and numeric precision, while leaving trading decisions entirely in your application.

It is not an official Bitget SDK and it does not execute a request during import or client creation.

## What this library is for

Think of the SDK as a small transport layer between your Go application and
Bitget. It signs private REST requests, turns documented responses into Go
types, and keeps a WebSocket connection alive. It deliberately does **not**
make trading decisions, convert prices to `float64`, or hide reconnects and
subscription errors from your application.

That gives you a simple rule of thumb:

- Use **REST** when you need a snapshot: balances, an order book, a recent
  page of orders, candles, or funding history.
- Use the regular **WebSocket subscription channel** when a bounded in-process
  feed is enough.
- Use the **raw-frame and lifecycle handlers** when every received payload,
  its receipt time, and the current connection generation matter to your
  downstream system.

## Start here

Start with a public REST call. It needs no credentials and cannot touch an
account. In a new directory, create a tiny Go module and run the program:

```bash
mkdir bitget-first-request
cd bitget-first-request
go mod init example.com/bitget-first-request
go get github.com/tigusigalpa/bitget-go
# Save the program below as main.go, then:
go run .
```

The module requires Go 1.21 or newer. Prices, quantities, balances, fees, and timestamps are represented as strings where the API returns strings. That avoids accidental `float64` rounding in financial code.

```go
package main

import (
	"context"
	"fmt"
	"log"

	bitget "github.com/tigusigalpa/bitget-go"
	"github.com/tigusigalpa/bitget-go/models"
)

func main() {
	client := bitget.NewRestClient("", "", "") // public endpoints need no key
	tickers, err := client.Market.GetTickers(context.Background(), models.CategorySpot, "BTCUSDT")
	if err != nil {
		log.Fatal(err)
	}
	if len(tickers) > 0 {
		fmt.Printf("BTC/USDT last price: %s\n", tickers[0].LastPrice)
	}
}
```

If you prefer to explore the repository examples, clone the project and run
`go run ./examples/rest`. The public REST and public WebSocket examples are
safe to run without environment variables.

## Choose the smallest client for the job

| Need | Create | Credentials | What it does |
|---|---|---|---|
| Snapshot market data | `NewRestClient("", "", "")` | No | Public REST calls such as tickers and order book |
| Account, orders, positions | `NewRestClient(key, secret, passphrase)` | Yes | Signed private REST calls |
| Live public market feed | `NewPublicWSClient()` | No | Subscribes to public WebSocket channels |
| Private fills and account feed | `NewPrivateWSClient(key, secret, passphrase)` | Yes | Logs in to a private WebSocket and subscribes to its channels |

There is no automatic switch between public and private APIs, production and demo endpoints, or REST and WebSocket. You choose the transport and credentials explicitly.

### A practical public market query

The market methods return strings for exchange-provided decimals and timestamps.
Keep those values as strings until your own domain layer deliberately converts
them with the precision rules your application needs. This complete helper
needs the `market` package in addition to the imports from the first example.

```go
import (
	"context"
	"fmt"
	"time"

	bitget "github.com/tigusigalpa/bitget-go"
	"github.com/tigusigalpa/bitget-go/models"
	"github.com/tigusigalpa/bitget-go/rest/market"
)

func printRecentMarket() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := bitget.NewRestClient("", "", "")
	candles, err := client.Market.GetCandles(ctx, market.GetCandlesOptions{
		Category: models.CategorySpot,
		Symbol:   "BTCUSDT",
		Interval: "1m",
		Limit:    "10",
	})
	if err != nil {
		return err
	}

	for _, candle := range candles {
		// candle = [timestamp, open, high, low, close, base volume, turnover]
		fmt.Printf("close=%s at %s\n", candle[4], candle[0])
	}

	fills, err := client.Market.GetPublicFills(ctx, models.CategorySpot, "BTCUSDT", "20")
	if err != nil {
		return err
	}
	for _, fill := range fills {
		fmt.Printf("%s %s @ %s (exec %s)\n", fill.Side, fill.Size, fill.Price, fill.ExecID)
	}

	return nil
}
```

`GetPublicFills` is a recent-data repair source, not a complete historical
trade ledger. For futures funding data, call `GetFundingRateHistory` with the
same category and symbol; use the corresponding instrument's `FundInterval`
instead of assuming a fixed settlement schedule.

## Demo credentials before production

Create a separate **Demo API key** in Bitget. Keep all three values outside source control:

- `BITGET_API_KEY`
- `BITGET_SECRET_KEY`
- `BITGET_PASSPHRASE`

Bitget requires the `paptrading: 1` header for demo REST requests. `WithDemoTrading()` adds it. The accompanying private examples additionally refuse to run until `BITGET_DEMO=1` is set; this is a guard against accidentally pointing copied code at a production account.

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

client := bitget.NewRestClient(
	os.Getenv("BITGET_API_KEY"),
	os.Getenv("BITGET_SECRET_KEY"),
	os.Getenv("BITGET_PASSPHRASE"),
	bitget.WithDemoTrading(),
)

assets, err := client.Account.GetAssets(ctx)
if err != nil {
	log.Fatal(err)
}
fmt.Println("demo USD equity:", assets.AccountEquity)
```

Run the complete safe demo-account example with:

```bash
BITGET_DEMO=1 BITGET_API_KEY=... BITGET_SECRET_KEY=... BITGET_PASSPHRASE=... go run ./examples/demo-account
```

Never put keys, passphrases, signatures, or `.env` files in a repository or issue. Bind an IP address to an API key where possible. See Bitget's [API-key security guidance](https://www.bitget.com/docs/uta/quick-start#access-setup).

## Configuration

The defaults target Bitget's regular UTA domains. A supplied `*http.Client` is used as-is; the SDK does not rewrite its timeout or transport.

| Option | Default | Use it for |
|---|---|---|
| `WithHTTPClient(*http.Client)` | Internal client with 15-second timeout | Proxy, custom TLS, tracing, or test transport |
| `WithTimeout(time.Duration)` | `15s` | Timeout of the internally created HTTP client |
| `WithBaseURL(string)` | `https://api.bitget.com` | A test server or Bitget VIP base origin |
| `WithDemoTrading()` | Off | REST requests made with a Demo API key |
| `WithLocale(string)` | `en-US` | Bitget's `locale` header |
| `WithLogger(Logger)` | No-op | Structured REST lifecycle logs |
| `WithWSURL(string)` | Public/private production URL | Demo, VIP, or test WebSocket endpoint |
| `WithWSAutoReconnect(bool)` | On | Disable automatic reconnect for a short-lived consumer |
| `WithWSLogger(Logger)` | No-op | Structured WebSocket lifecycle logs |

`WithBaseURL` expects an origin, not a path ending in `/api/v3`: endpoint methods add their own `/api/v3/...` path. Bitget documents the production, demo, and VIP domains in its [Quick Start](https://www.bitget.com/docs/uta/quick-start#domain-name).

```go
httpClient := &http.Client{Timeout: 30 * time.Second, Transport: myTransport}
client := bitget.NewRestClient("", "", "", bitget.WithHTTPClient(httpClient))
```

## Implemented REST coverage

This is a focused Phase 1 surface. Methods not listed here are not implemented yet.

| Service | Methods |
|---|---|
| `Market` (public) | `GetInstruments`, `GetTickers`, `GetOrderBook`, `GetCandles`, `GetPublicFills`, `GetFundingRateHistory` |
| `Account` (private) | `GetAssets`, `GetSettings`, `SetLeverage` |
| `Trade` (private) | `PlaceOrder`, `ModifyOrder`, `CancelOrder`, `GetOpenOrders`, `GetOrderHistory`, `GetPositions` |

See [docs/endpoints.md](docs/endpoints.md) for the exact HTTP paths and Bitget documentation for each method. Model fields are deliberately forward-compatible strings where a future API value should not be rejected by a closed enum.

### Pages, cursors, and cancellation

Order-list endpoints return a `Cursor`. Pass it back only after you have processed the current page. Give every request a deadline appropriate for your application.

```go
page, err := client.Trade.GetOpenOrders(ctx, trade.GetOpenOrdersOptions{
	Category: models.CategorySpot,
	Limit:    "100",
})
if err != nil {
	return err
}

for _, order := range page.List {
	// Persist or handle order before asking for the next page.
}
if page.Cursor != "" {
	// Request the next page with Cursor: page.Cursor.
}
```

Do not automatically retry an order placement after a timeout: the exchange may have accepted it even when your process did not receive a response. Use a unique `ClientOid` and then query the order state. For read-only requests, build your own retry policy around `ErrRateLimited`, context cancellation, and the endpoint-specific limits.

## Error handling

The package keeps Bitget's response code and message in `*bitget.Error`; use `errors.Is` for stable classifications and `errors.As` for details. A 429 response retains both `ErrRateLimited` and its API error payload when Bitget supplies one.

```go
assets, err := client.Account.GetAssets(ctx)
if err != nil {
	if errors.Is(err, bitget.ErrRateLimited) {
		// Back off according to the endpoint's documented limit.
		return
	}

	var apiErr *bitget.Error
	if errors.As(err, &apiErr) {
		log.Printf("Bitget rejected the request: code=%s message=%s", apiErr.Code, apiErr.Message)
		return
	}

	log.Printf("transport or decoding failure: %v", err)
}
```

Available classifications include `ErrUnauthorized`, `ErrInvalidSignature`, `ErrInvalidTimestamp`, `ErrPermissionDenied`, `ErrRateLimited`, `ErrInvalidParameter`, `ErrInsufficientFunds`, `ErrOrderNotFound`, and `ErrInternalServer`.

## WebSocket lifecycle

Use a context that you can cancel and always close the client. The client sends `ping`, expects `pong`, reconnects with exponential backoff after an unexpected disconnect, and resubscribes its active channels. It does not promise exactly-once or globally ordered delivery: make downstream processing idempotent, and deduplicate events using exchange IDs when a channel provides them.

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()

ws := bitget.NewPublicWSClient()
if err := ws.Connect(ctx); err != nil {
	log.Fatal(err)
}
defer ws.Close()

pushes, err := ws.Subscribe(ctx, models.WSArg{
	InstType: "SPOT",
	Topic:    "ticker",
	Symbol:   "BTCUSDT",
})
if err != nil {
	log.Fatal(err)
}

for {
	select {
	case <-ctx.Done():
		return
	case push, ok := <-pushes:
		if !ok {
			return
		}
		fmt.Println(string(push.Data))
	}
}
```

Each subscription has a 100-message buffer. If the consumer cannot keep up, the SDK drops the newest push and emits a warning through the configured WebSocket logger. For high-volume channels, process messages promptly, keep the handler non-blocking, and use a durable queue in your own application if loss is unacceptable.

For a lossless, provenance-sensitive path, configure `WithRawFrameHandler`.
It is called synchronously with an exact copied payload, immediate local receipt
time, and connection generation before the SDK decodes JSON or writes to a
subscriber buffer. Pair it with `WithWSEventHandler` to observe subscription
requests, Bitget ACKs/errors and terminal raw-observer failures by channel
`arg`. These callbacks deliberately apply backpressure; keep their work short
or hand off to your own durable queue. Mark a subscription ready only after a
`WSSubscribed` event for the current connection generation; an error with an
`arg` closes and removes that rejected subscription.

### Wait for a subscription ACK

`Subscribe` confirms that the SDK wrote the request; it does not mean Bitget
has accepted it. For a single, non-reconnecting consumer, wait for
`WSSubscribed` before treating the feed as ready. The example also surfaces a
provider rejection instead of silently continuing with an empty channel.

```go
// Imports: context, fmt, sync, plus bitget and models.
// The caller must close the returned client when it no longer needs the feed.
func subscribeWhenReady(ctx context.Context) (*bitget.WSClient, <-chan models.WSPush, error) {
	ready := make(chan struct{})
	providerError := make(chan error, 1)
	var readyOnce sync.Once

	ws := bitget.NewPublicWSClient(
		bitget.WithWSAutoReconnect(false),
		bitget.WithWSEventHandler(func(event bitget.WSLifecycleEvent) {
			switch event.Type {
			case bitget.WSSubscribed:
				if event.Arg != nil && event.Arg.Topic == "ticker" && event.Arg.Symbol == "BTCUSDT" {
					readyOnce.Do(func() { close(ready) })
				}
			case bitget.WSProviderError:
				select {
				case providerError <- fmt.Errorf("Bitget rejected subscription %#v: %s %s", event.Arg, event.Code, event.Message):
				default: // Keep the synchronous lifecycle callback non-blocking.
				}
			}
		}),
	)

	if err := ws.Connect(ctx); err != nil {
		return nil, nil, err
	}

	pushes, err := ws.Subscribe(ctx, models.WSArg{InstType: "SPOT", Topic: "ticker", Symbol: "BTCUSDT"})
	if err != nil {
		_ = ws.Close()
		return nil, nil, err
	}

	select {
	case <-ready:
		// The provider accepted this subscription.
		return ws, pushes, nil
	case err := <-providerError:
		_ = ws.Close()
		return nil, nil, err
	case <-ctx.Done():
		_ = ws.Close()
		return nil, nil, ctx.Err()
	}
}
```

For reconnecting consumers, track the latest `WSConnected` generation and
accept a `WSSubscribed` event only from that generation. A reconnect requires
a fresh ACK; do not treat a prior connection's acknowledgment as current.

### Preserve exact received messages

The regular `WSPush` route is intentionally convenient and bounded. If you
need audit-grade provenance or must hand messages to a durable queue without a
decode/re-encode cycle, use the raw handler. It receives a private byte copy
before the SDK parses JSON.

```go
// Imports: errors, log, plus bitget.
func newRawClient(rawFrames chan<- bitget.RawFrame) *bitget.WSClient {
	return bitget.NewPublicWSClient(
		bitget.WithRawFrameHandler(func(frame bitget.RawFrame) error {
			// The caller must drain this channel or replace it with a durable queue.
			select {
			case rawFrames <- frame:
				return nil
			default:
				return errors.New("raw-frame consumer is overloaded")
			}
		}),
		bitget.WithWSEventHandler(func(event bitget.WSLifecycleEvent) {
			if event.Type == bitget.WSTerminal {
				log.Printf("raw frame consumer stopped: %v", event.Err)
			}
		}),
	)
}
```

Returning an error from the raw handler deliberately produces a terminal
lifecycle event and disconnects the current connection. That makes overload or
storage failure visible to the caller rather than silently losing canonical
data. Do not mutate `frame.Payload` and do not synchronously perform slow I/O
unless deliberate backpressure is what you want.

The public `examples/websocket` program is credential-free. The private `examples/websocket-private` program uses Bitget's Demo private endpoint explicitly and decodes the typed `fast-fill` payload.

Bitget documents a maximum of 10 WebSocket messages per second, recommends fewer than 50 channel subscriptions per connection, and publishes connection/subscription caps. Design a connection pool and resubscription cadence around those limits rather than opening a connection per symbol. [WebSocket connection guidance](https://www.bitget.com/docs/uta/quick-start#websocket)

## Runnable examples

| Program | What it demonstrates | Credentials |
|---|---|---|
| [`examples/rest`](examples/rest/main.go) | Public ticker and safe order-book access | None |
| [`examples/demo-account`](examples/demo-account/main.go) | Demo-only signed account read | Demo key + `BITGET_DEMO=1` |
| [`examples/websocket`](examples/websocket/main.go) | Public ticker subscription, signal cancellation | None |
| [`examples/websocket-private`](examples/websocket-private/main.go) | Demo private `fast-fill` subscription and payload decoding | Demo key + `BITGET_DEMO=1` |

## Development

```bash
go test ./...
go test -race ./...
go vet ./...
gofmt -l .
```

Integration tests are opt-in and never receive a credential in CI:

```bash
go test -tags=integration ./...
```

The public integration test contacts Bitget. The private integration test is skipped unless all three `BITGET_*` variables are present, and uses demo mode.

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request. For exact endpoint coverage, see [docs/endpoints.md](docs/endpoints.md). For a security report, see [SECURITY.md](SECURITY.md).

### Before opening a pull request

Run the same checks used for the library's core behaviour:

```bash
go test ./...
go test -race ./...
golint ./...
go vet ./...
```

Add a focused test whenever an endpoint, a signed request, or a WebSocket
lifecycle branch changes. Tests use local HTTP/WebSocket servers by default;
the optional integration suite is the only test path that contacts Bitget.

## License and attribution

MIT — see [LICENSE](LICENSE). Maintained by [Igor Sazonov](https://github.com/tigusigalpa). This project is not affiliated with Bitget.
