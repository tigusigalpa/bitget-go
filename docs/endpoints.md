# Endpoint Coverage Map

Every SDK method below links to its exact Bitget UTA v3 documentation page.
Anything not listed here is **not implemented** in this Phase 1 release.

## Market (Public)

| SDK Method | HTTP | Path | Docs |
|---|---|---|---|
| `Market.GetInstruments` | GET | `/api/v3/market/instruments` | https://www.bitget.com/api-doc/uta/public/Instruments |
| `Market.GetTickers` | GET | `/api/v3/market/tickers` | https://www.bitget.com/api-doc/uta/public/Tickers |
| `Market.GetOrderBook` | GET | `/api/v3/market/orderbook` | https://www.bitget.com/api-doc/uta/public/OrderBook |
| `Market.GetCandles` | GET | `/api/v3/market/candles` | https://www.bitget.com/docs/catalog/market/market-data |
| `Market.GetPublicFills` | GET | `/api/v3/market/fills` | https://www.bitget.com/docs/catalog/market/market-data |
| `Market.GetFundingRateHistory` | GET | `/api/v3/market/history-fund-rate` | https://www.bitget.com/legacy-docs/uta/public/Get-History-Funding-Rate |

## Account (Private)

| SDK Method | HTTP | Path | Docs |
|---|---|---|---|
| `Account.GetAssets` | GET | `/api/v3/account/assets` | https://www.bitget.com/api-doc/uta/account/Get-Account |
| `Account.GetSettings` | GET | `/api/v3/account/settings` | https://www.bitget.com/api-doc/uta/account/Get-Account-Setting |
| `Account.SetLeverage` | POST | `/api/v3/account/set-leverage` | https://www.bitget.com/api-doc/uta/account/Change-Leverage |

## Trade (Private)

| SDK Method | HTTP | Path | Docs |
|---|---|---|---|
| `Trade.PlaceOrder` | POST | `/api/v3/trade/place-order` | https://www.bitget.com/api-doc/uta/trade/Place-Order |
| `Trade.ModifyOrder` | POST | `/api/v3/trade/modify-order` | https://www.bitget.com/api-doc/uta/trade/Modify-Order |
| `Trade.CancelOrder` | POST | `/api/v3/trade/cancel-order` | https://www.bitget.com/api-doc/uta/trade/Cancel-Order |
| `Trade.GetOpenOrders` | GET | `/api/v3/trade/unfilled-orders` | https://www.bitget.com/api-doc/uta/trade/Get-Order-Pending |
| `Trade.GetOrderHistory` | GET | `/api/v3/trade/history-orders` | https://www.bitget.com/api-doc/uta/trade/Get-Order-History |
| `Trade.GetPositions` | GET | `/api/v3/position/current-position` | https://www.bitget.com/api-doc/uta/trade/Get-Position |

## WebSocket

| Channel | Type | Docs |
|---|---|---|
| `fast-fill` | Private | https://www.bitget.com/api-doc/uta/websocket/private/Fast-Fill-Channel |

The `WSClient` transport (connect/login/subscribe/unsubscribe/reconnect) is
channel-agnostic — `Subscribe(ctx, models.WSArg{...})` works with any public
or private channel Bitget exposes, including ones not listed above. Only the
`fast-fill` channel has a typed payload struct (`models.FastFill`) in this
Phase 1 release; for other channels, decode `WSPush.Data` yourself.

For consumers that require provenance rather than the lossy typed push buffer,
`WithRawFrameHandler` receives an exact copied frame synchronously before JSON
decoding, together with local receipt time and a connection generation.
`WithWSEventHandler` receives synchronous subscribe/unsubscribe ACKs and
provider errors correlated to the server-supplied `arg`. A raw-frame handler
error is reported as a terminal lifecycle event and disconnects the client, so
callers can invalidate any derived market state.

## Not implemented (Phase 1)

Everything else in the UTA v3 API surface — including, but not limited to:
Asset/transfer endpoints, Trading Bot, Copy Trading, RFQ, Rubik market
statistics, Spread trading, Affiliate, Fiat, Finance/earn products, batch
order endpoints, plan/trigger (conditional) orders, and all other public and
private WebSocket channels (candles, trades, order book, account, position,
order, and public trade channels).

Contributions adding coverage are welcome — see [CONTRIBUTING.md](../CONTRIBUTING.md).
