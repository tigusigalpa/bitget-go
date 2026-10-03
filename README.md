# Bitget Go SDK

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/tigusigalpa/bitget-go.svg)](https://pkg.go.dev/github.com/tigusigalpa/bitget-go)
[![Tests](https://github.com/tigusigalpa/bitget-go/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/bitget-go/actions/workflows/test.yml)
[![Codecov](https://codecov.io/gh/tigusigalpa/bitget-go/graph/badge.svg)](https://codecov.io/gh/tigusigalpa/bitget-go)

An intentionally small, typed Go client for the [Bitget Unified Trading Account (UTA) API v3](https://www.bitget.com/docs/uta/quick-start). It takes care of request signing, response envelopes, WebSocket lifecycle, and numeric precision, while leaving trading decisions entirely in your application.

It is not an official Bitget SDK and it does not execute a request during import or client creation.

## Start here

Use a public REST example first. It needs no credentials and never touches an account:

```bash
go get github.com/tigusigalpa/bitget-go
go run ./examples/rest
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

## Choose the smallest client for the job

| Need | Create | Credentials | What it does |
|---|---|---|---|
| Snapshot market data | `NewRestClient("", "", "")` | No | Public REST calls such as tickers and order book |
| Account, orders, positions | `NewRestClient(key, secret, passphrase)` | Yes | Signed private REST calls |
| Live public market feed | `NewPublicWSClient()` | No | Subscribes to public WebSocket channels |
| Private fills and account feed | `NewPrivateWSClient(key, secret, passphrase)` | Yes | Logs in to a private WebSocket and subscribes to its channels |

There is no automatic switch between public and private APIs, production and demo endpoints, or REST and WebSocket. You choose the transport and credentials explicitly.

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

## License and attribution

MIT — see [LICENSE](LICENSE). Maintained by [Igor Sazonov](https://github.com/tigusigalpa). This project is not affiliated with Bitget.
