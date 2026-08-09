# Bitget Go SDK

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/tigusigalpa/bitget-go.svg)](https://pkg.go.dev/github.com/tigusigalpa/bitget-go)

An idiomatic, dependency-light Go SDK for the [Bitget Unified Trading Account (UTA) API v3](https://www.bitget.com/api-doc/uta/intro). Phase 1 covers Market, Account, and Trade REST services plus a reconnecting WebSocket client, with `context.Context` throughout and an injectable `*http.Client`.

**Package:** a matching PHP/Laravel SDK is available at [tigusigalpa/bitget-php](https://github.com/tigusigalpa/bitget-php).

## What's inside

- `context.Context` as the first parameter on every network call
- An injectable `*http.Client` (proxies, custom transports, test doubles)
- Strings for every price/quantity/PnL/fee field — no float rounding errors
- A generic response envelope (`models.BitgetResponse[T]`) and typed models for every implemented endpoint
- A `gorilla/websocket`-based client with exponential-backoff reconnection, text ping/pong heartbeats, and automatic resubscription after reconnect
- Sentinel errors (`errors.Is`) plus a detailed `*BitgetError` (`errors.As`) on every failure
- Zero required third-party dependencies outside `gorilla/websocket` (`stretchr/testify` is test-only)

## Install

```bash
go get github.com/tigusigalpa/bitget-go
```

## Quick start

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
	fmt.Println(tickers[0].LastPrice)
}
```

Runnable examples: [examples/rest](examples/rest/main.go), [examples/websocket](examples/websocket/main.go).

## Configuration

| Option | Description | Default |
|---|---|---|
| `WithHTTPClient(*http.Client)` | Inject a custom HTTP client | `&http.Client{Timeout: 15s}` |
| `WithBaseURL(string)` | Override the REST base URL | `https://api.bitget.com` |
| `WithDemoTrading()` | Send `paptrading: 1` on every request (use with a Demo API key) | disabled |
| `WithTimeout(time.Duration)` | Timeout for the internally built HTTP client | `15s` |
| `WithLogger(Logger)` | Structured logger (never logs keys/signatures) | no-op |
| `WithLocale(string)` | `locale` header value | `en-US` |

## REST API coverage (Phase 1)

| Category | Methods | Docs |
|---|---|---|
| Market (public) | `GetInstruments`, `GetTickers`, `GetOrderBook` | [Instruments](https://www.bitget.com/api-doc/uta/public/Instruments) · [Tickers](https://www.bitget.com/api-doc/uta/public/Tickers) · [OrderBook](https://www.bitget.com/api-doc/uta/public/OrderBook) |
| Account (private) | `GetAssets`, `GetSettings`, `SetLeverage` | [Get-Account](https://www.bitget.com/api-doc/uta/account/Get-Account) · [Get-Account-Setting](https://www.bitget.com/api-doc/uta/account/Get-Account-Setting) · [Change-Leverage](https://www.bitget.com/api-doc/uta/account/Change-Leverage) |
| Trade (private) | `PlaceOrder`, `ModifyOrder`, `CancelOrder`, `GetOpenOrders`, `GetOrderHistory`, `GetPositions` | [Place-Order](https://www.bitget.com/api-doc/uta/trade/Place-Order) · [Modify-Order](https://www.bitget.com/api-doc/uta/trade/Modify-Order) · [Cancel-Order](https://www.bitget.com/api-doc/uta/trade/Cancel-Order) · [Get-Order-Pending](https://www.bitget.com/api-doc/uta/trade/Get-Order-Pending) · [Get-Order-History](https://www.bitget.com/api-doc/uta/trade/Get-Order-History) · [Get-Position](https://www.bitget.com/api-doc/uta/trade/Get-Position) |

Full mapping with HTTP methods and paths: [docs/endpoints.md](docs/endpoints.md).

## WebSocket

```go
ws := bitget.NewPublicWSClient()
if err := ws.Connect(ctx); err != nil {
	log.Fatal(err)
}
defer ws.Close()

pushes, err := ws.Subscribe(ctx, models.WSArg{InstType: "SPOT", Topic: "ticker", Symbol: "BTCUSDT"})
if err != nil {
	log.Fatal(err)
}
for push := range pushes {
	fmt.Println(string(push.Data))
}
```

For private channels (e.g. order fills), use `NewPrivateWSClient(apiKey, secretKey, passphrase)` — `Connect` authenticates automatically. On an unexpected disconnect, the client reconnects with exponential backoff (1s → 60s cap) and resubscribes every previously active channel.

Implemented private channel: [`fast-fill`](https://www.bitget.com/api-doc/uta/websocket/private/Fast-Fill-Channel). Other channels work through the same `Subscribe`/`WSPush.Data` API; see [docs/endpoints.md](docs/endpoints.md) for what's typed vs. what you decode yourself.

## Demo trading

Set `WithDemoTrading()` together with a **Demo API key** from the Bitget console to send `paptrading: 1` on every REST request, or use `DemoPublicWSURL`/`DemoPrivateWSURL` for WebSocket. Always validate new code against demo credentials before pointing it at a live account — see the trading example in [examples/rest/main.go](examples/rest/main.go), which requires both `BITGET_DEMO=1` and `BITGET_ENABLE_TRADING=1` before it will place an order.

## Errors

```go
_, err := client.Account.GetAssets(ctx)
if errors.Is(err, bitget.ErrUnauthorized) {
	// invalid API key/secret/passphrase
}

var bitgetErr *bitget.BitgetError
if errors.As(err, &bitgetErr) {
	fmt.Println(bitgetErr.Code, bitgetErr.Message)
}
```

## Tests

```bash
go test ./...                       # unit tests (offline, httptest-backed)
go test -tags=integration ./...     # integration tests against Bitget demo (requires env credentials)
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

Found a vulnerability? Email sovletig@gmail.com directly — please don't open a public issue.

## License

MIT. See [LICENSE](LICENSE).

## Author

Igor Sazonov — [@tigusigalpa](https://github.com/tigusigalpa) — sovletig@gmail.com

## Links

- [Bitget UTA API docs](https://www.bitget.com/api-doc/uta/intro)
- [Repository](https://github.com/tigusigalpa/bitget-go)
- [Issues](https://github.com/tigusigalpa/bitget-go/issues)

---

*Not affiliated with Bitget. Test on demo before going live.*
