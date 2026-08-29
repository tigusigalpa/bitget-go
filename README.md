# Bitget Go SDK

![Bitget Golang SDK](https://i.postimg.cc/j2ZkYg04/bitget-golang-github.jpg)

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/tigusigalpa/bitget-go.svg)](https://pkg.go.dev/github.com/tigusigalpa/bitget-go)
[![Tests](https://github.com/tigusigalpa/bitget-go/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/bitget-go/actions/workflows/test.yml)
[![Codecov](https://codecov.io/gh/tigusigalpa/bitget-go/graph/badge.svg)](https://codecov.io/gh/tigusigalpa/bitget-go)
[![CodeQL](https://github.com/tigusigalpa/bitget-go/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/bitget-go/actions/workflows/codeql.yml)

A clean, idiomatic Go SDK for the [Bitget Unified Trading Account (UTA) API v3](https://www.bitget.com/api-doc/uta/intro). Built for developers who want reliable market data, account management, and trading — without wrestling with raw HTTP signatures or silently losing precision to `float64`.

> 📖 **[Full documentation available on Wiki](https://github.com/tigusigalpa/bitget-go/wiki)**

A matching PHP/Laravel SDK lives at [tigusigalpa/bitget-php](https://github.com/tigusigalpa/bitget-php) if you also run services in that ecosystem.

---

## Why this SDK?

Bitget's API is powerful, but building against raw HTTP can be tedious: signature schemes, subtle parameter encoding, reconnecting WebSockets, and the eternal problem of floating-point rounding in financial data. This package handles the boilerplate so you can focus on your trading logic.

It is intentionally **dependency-light** and **Go-idiomatic**:

- `context.Context` is the first argument on every network call, so cancellation and timeouts behave the way you expect in Go.
- You can inject your own `*http.Client` for proxies, custom transports, or test doubles.
- Prices, quantities, PnL, and fees are returned as `string`, so you never lose a satoshi to `float64` rounding.
- Every endpoint returns a typed `models.BitgetResponse[T]` envelope instead of `interface{}`.
- WebSockets reconnect automatically with exponential backoff and resubscribe to your channels.
- Errors are plain Go sentinel errors (`errors.Is`) plus a typed `*BitgetError` (`errors.As`) for detailed API messages.
- Only one required runtime dependency: `gorilla/websocket`. `stretchr/testify` is test-only.

---

## Installation

```bash
go get github.com/tigusigalpa/bitget-go
```

Requires Go 1.21 or newer.

---

## Quick start

### 1. Get your API credentials

Log in to the [Bitget console](https://www.bitget.com/support/en-US/articles/360007388154-How-to-Create-API-Key), create an API key, and save:

- `BITGET_API_KEY`
- `BITGET_SECRET_KEY`
- `BITGET_PASSPHRASE`

For your own safety, start with a **Demo API key**. You can switch to production later by changing the credentials and removing demo mode.

### 2. Make your first call

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	bitget "github.com/tigusigalpa/bitget-go"
	"github.com/tigusigalpa/bitget-go/models"
)

func main() {
	client := bitget.NewRestClient(
		os.Getenv("BITGET_API_KEY"),
		os.Getenv("BITGET_SECRET_KEY"),
		os.Getenv("BITGET_PASSPHRASE"),
	)

	tickers, err := client.Market.GetTickers(context.Background(), models.CategorySpot, "BTCUSDT")
	if err != nil {
		log.Fatal(err)
	}

	if len(tickers) > 0 {
		fmt.Printf("BTC/USDT last price: %s\n", tickers[0].LastPrice)
	}
}
```

That's it — the SDK signs the request, sets the right headers, parses the response envelope, and gives you typed data.

---

## Configuration

`NewRestClient` accepts functional options so you can tune behavior without breaking the simple constructor:

| Option | Description | Default |
|---|---|---|
| `WithHTTPClient(*http.Client)` | Inject a custom HTTP client (proxy, custom TLS, tracing, etc.) | `&http.Client{Timeout: 15s}` |
| `WithBaseURL(string)` | Override the REST base URL | `https://api.bitget.com` |
| `WithDemoTrading()` | Send `paptrading: 1` on every request. Use with a Demo API key. | disabled |
| `WithTimeout(time.Duration)` | Timeout for the internally built HTTP client | `15s` |
| `WithLogger(Logger)` | Structured logger. Keys and signatures are never logged. | no-op |
| `WithLocale(string)` | `locale` header value | `en-US` |

Example with a custom HTTP client:

```go
client := bitget.NewRestClient(
	os.Getenv("BITGET_API_KEY"),
	os.Getenv("BITGET_SECRET_KEY"),
	os.Getenv("BITGET_PASSPHRASE"),
	bitget.WithHTTPClient(&http.Client{
		Timeout:   30 * time.Second,
		Transport: myProxyTransport,
	}),
	bitget.WithDemoTrading(),
)
```

---

## REST API coverage (Phase 1)

| Category | Methods | Official docs |
|---|---|---|
| **Market** (public) | `GetInstruments`, `GetTickers`, `GetOrderBook` | [Instruments](https://www.bitget.com/api-doc/uta/public/Instruments) · [Tickers](https://www.bitget.com/api-doc/uta/public/Tickers) · [OrderBook](https://www.bitget.com/api-doc/uta/public/OrderBook) |
| **Account** (private) | `GetAssets`, `GetSettings`, `SetLeverage` | [Get-Account](https://www.bitget.com/api-doc/uta/account/Get-Account) · [Get-Account-Setting](https://www.bitget.com/api-doc/uta/account/Get-Account-Setting) · [Change-Leverage](https://www.bitget.com/api-doc/uta/account/Change-Leverage) |
| **Trade** (private) | `PlaceOrder`, `ModifyOrder`, `CancelOrder`, `GetOpenOrders`, `GetOrderHistory`, `GetPositions` | [Place-Order](https://www.bitget.com/api-doc/uta/trade/Place-Order) · [Modify-Order](https://www.bitget.com/api-doc/uta/trade/Modify-Order) · [Cancel-Order](https://www.bitget.com/api-doc/uta/trade/Cancel-Order) · [Get-Order-Pending](https://www.bitget.com/api-doc/uta/trade/Get-Order-Pending) · [Get-Order-History](https://www.bitget.com/api-doc/uta/trade/Get-Order-History) · [Get-Position](https://www.bitget.com/api-doc/uta/trade/Get-Position) |

For the exact HTTP methods, paths, and query/body parameters, see [docs/endpoints.md](docs/endpoints.md).

### A note about demo trading

If you set `WithDemoTrading()`, every REST request carries the `paptrading: 1` header. Make sure you are using **Demo API credentials** — mixing demo mode with production credentials will fail. The trading example in [examples/rest/main.go](examples/rest/main.go) is gated behind both `BITGET_DEMO=1` and `BITGET_ENABLE_TRADING=1` so it cannot accidentally place a live order.

---

## WebSocket

Real-time data is where Go really shines. The SDK gives you a streaming channel and handles reconnection behind the scenes.

### Public channels

```go
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

for push := range pushes {
	fmt.Println(string(push.Data))
}
```

### Private channels

Use `NewPrivateWSClient(apiKey, secretKey, passphrase)`. Authentication happens automatically during `Connect`.

```go
ws := bitget.NewPrivateWSClient(
	os.Getenv("BITGET_API_KEY"),
	os.Getenv("BITGET_SECRET_KEY"),
	os.Getenv("BITGET_PASSPHRASE"),
)
```

On an unexpected disconnect, the client:

1. Backs off exponentially from 1 second up to a 60-second cap.
2. Reconnects.
3. Resubscribes every channel you previously opened.

Implemented private channel: [`fast-fill`](https://www.bitget.com/api-doc/uta/websocket/private/Fast-Fill-Channel). Other channels use the same `Subscribe`/`WSPush.Data` shape; check [docs/endpoints.md](docs/endpoints.md) to see which payloads are already typed and which you should decode from `push.Data` yourself.

---

## Error handling

The SDK returns plain errors you can check with the standard library:

```go
_, err := client.Account.GetAssets(ctx)
if err != nil {
	if errors.Is(err, bitget.ErrUnauthorized) {
		// Most likely the API key, secret, or passphrase is wrong.
		log.Println("authentication failed — check your credentials")
		return
	}

	var bitgetErr *bitget.Error
	if errors.As(err, &bitgetErr) {
		// Bitget returned a business-level error.
		log.Printf("Bitget error %s: %s", bitgetErr.Code, bitgetErr.Message)
		return
	}

	// Network or timeout issue.
	log.Printf("request failed: %v", err)
}
```

Common `errors.Is` checks include network timeouts and context cancellation, because the SDK propagates those transparently.

---

## Running the tests

```bash
# Unit tests — fast, offline, backed by httptest
go test ./...

# Integration tests against Bitget demo environment.
# Requires BITGET_API_KEY, BITGET_SECRET_KEY, and BITGET_PASSPHRASE to be set.
go test -tags=integration ./...
```

We strongly recommend running integration tests with **demo credentials** before you point any code at a live account.

---

## Examples

Two runnable examples are included:

- [`examples/rest/main.go`](examples/rest/main.go) — market data, account info, and placing a demo order.
- [`examples/websocket/main.go`](examples/websocket/main.go) — public and private WebSocket subscriptions.

Copy them, set your environment variables, and run:

```bash
BITGET_API_KEY=xxx BITGET_SECRET_KEY=xxx BITGET_PASSPHRASE=xxx go run examples/rest/main.go
```

---

## A few practical tips

1. **Use strings for money.** All numeric financial fields are `string` in the SDK. Use `math/big.Rat` or a decimal library of your choice; avoid `strconv.ParseFloat` when precision matters.
2. **Start on demo.** Even experienced traders should validate new code against demo keys first. Markets move fast; a bug in order size or symbol formatting can be expensive.
3. **Respect rate limits.** The SDK does not throttle for you. Bitget publishes rate-limit headers; if you need heavy polling, consider WebSockets instead of REST.
4. **Pass contexts with deadlines.** This is especially important for trading endpoints where a slow request may no longer be relevant by the time it completes.
5. **Check errors by type.** Use `errors.Is` for known sentinel errors and `errors.As` for `*BitgetError` to avoid fragile string matching.

---

## Contributing

Contributions, bug reports, and suggestions are welcome. Please see [CONTRIBUTING.md](CONTRIBUTING.md) for the workflow, code style, and how to add new endpoints or channels.

A good first issue is often adding a missing endpoint model or improving test coverage.

---

## Security

Found something that should not be public? Please email **sovletig@gmail.com** directly rather than opening a public issue. We will investigate and fix it as quickly as possible.

The SDK itself never logs API keys, secrets, or signatures, regardless of the logger you inject.

---

## License

MIT. See [LICENSE](LICENSE).

---

## Author

Igor Sazonov — [@tigusigalpa](https://github.com/tigusigalpa) — sovletig@gmail.com

## Useful links

- [Bitget UTA API documentation](https://www.bitget.com/api-doc/uta/intro)
- [GitHub repository](https://github.com/tigusigalpa/bitget-go)
- [GitHub Wiki documentation](https://github.com/tigusigalpa/bitget-go/wiki)
- [Issue tracker](https://github.com/tigusigalpa/bitget-go/issues)

---

*Not affiliated with Bitget. Trade carefully, test on demo first, and never commit API credentials to source control.*
