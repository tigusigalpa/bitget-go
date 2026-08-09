// Package models contains typed request/response structures for the
// Bitget UTA v3 API. All price/quantity/PnL/fee fields are kept as strings,
// matching the wire format exactly, to avoid IEEE-754 rounding errors.
//
// Docs: https://www.bitget.com/api-doc/uta/intro
package models

// BitgetResponse is the generic envelope wrapping every Bitget REST
// response.
//
// Docs: https://www.bitget.com/api-doc/uta/guide
type BitgetResponse[T any] struct {
	Code        string `json:"code"`
	Msg         string `json:"msg"`
	RequestTime int64  `json:"requestTime"`
	Data        T      `json:"data"`
}

// Category is a Bitget product type. Kept as a plain string (not a typed
// enum) to stay forward-compatible with new categories Bitget may add.
//
// Common values: SPOT, MARGIN, USDT-FUTURES, COIN-FUTURES, USDC-FUTURES.
type Category = string

const (
	CategorySpot        Category = "SPOT"
	CategoryMargin      Category = "MARGIN"
	CategoryUSDTFutures Category = "USDT-FUTURES"
	CategoryCoinFutures Category = "COIN-FUTURES"
	CategoryUSDCFutures Category = "USDC-FUTURES"
)

// Side is an order direction: "buy" or "sell".
type Side = string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

// PosSide is a hedge-mode futures position side: "long" or "short".
type PosSide = string

const (
	PosSideLong  PosSide = "long"
	PosSideShort PosSide = "short"
)

// OrderType is an order execution type: "limit" or "market".
type OrderType = string

const (
	OrderTypeLimit  OrderType = "limit"
	OrderTypeMarket OrderType = "market"
)

// TimeInForce controls how the unfilled portion of an order is handled.
type TimeInForce = string

const (
	TimeInForceIOC      TimeInForce = "ioc"
	TimeInForceFOK      TimeInForce = "fok"
	TimeInForceGTC      TimeInForce = "gtc"
	TimeInForcePostOnly TimeInForce = "post_only"
	TimeInForceRPI      TimeInForce = "rpi"
)

// MarginMode is "crossed" or "isolated".
type MarginMode = string

const (
	MarginModeCrossed  MarginMode = "crossed"
	MarginModeIsolated MarginMode = "isolated"
)
