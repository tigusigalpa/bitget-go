package models

// PlaceOrderRequest places a new spot, margin, or futures order.
//
// Docs: https://www.bitget.com/api-doc/uta/trade/Place-Order
type PlaceOrderRequest struct {
	Category     Category    `json:"category"`
	Symbol       string      `json:"symbol"`
	Qty          string      `json:"qty"`
	Price        string      `json:"price,omitempty"`
	Side         Side        `json:"side"`
	OrderType    OrderType   `json:"orderType"`
	TimeInForce  TimeInForce `json:"timeInForce,omitempty"`
	PosSide      PosSide     `json:"posSide,omitempty"`
	ClientOid    string      `json:"clientOid,omitempty"`
	ReduceOnly   string      `json:"reduceOnly,omitempty"`
	StpMode      string      `json:"stpMode,omitempty"`
	TpTriggerBy  string      `json:"tpTriggerBy,omitempty"`
	SlTriggerBy  string      `json:"slTriggerBy,omitempty"`
	TakeProfit   string      `json:"takeProfit,omitempty"`
	StopLoss     string      `json:"stopLoss,omitempty"`
	TpOrderType  OrderType   `json:"tpOrderType,omitempty"`
	SlOrderType  OrderType   `json:"slOrderType,omitempty"`
	TpLimitPrice string      `json:"tpLimitPrice,omitempty"`
	SlLimitPrice string      `json:"slLimitPrice,omitempty"`
	MarginMode   MarginMode  `json:"marginMode,omitempty"`
}

// OrderRef identifies an order by exchange-assigned or client-assigned ID.
// Returned by PlaceOrder, ModifyOrder, and CancelOrder.
type OrderRef struct {
	OrderID   string `json:"orderId"`
	ClientOid string `json:"clientOid"`
}

// ModifyOrderRequest amends the price/quantity/TP-SL of an open order.
// Either OrderID or ClientOid must be set; if both are set, OrderID wins.
//
// Docs: https://www.bitget.com/api-doc/uta/trade/Modify-Order
type ModifyOrderRequest struct {
	OrderID      string    `json:"orderId,omitempty"`
	ClientOid    string    `json:"clientOid,omitempty"`
	Qty          string    `json:"qty,omitempty"`
	Price        string    `json:"price,omitempty"`
	AutoCancel   string    `json:"autoCancel,omitempty"`
	Symbol       string    `json:"symbol,omitempty"`
	Category     Category  `json:"category,omitempty"`
	TpTriggerBy  string    `json:"tpTriggerBy,omitempty"`
	SlTriggerBy  string    `json:"slTriggerBy,omitempty"`
	TakeProfit   string    `json:"takeProfit,omitempty"`
	StopLoss     string    `json:"stopLoss,omitempty"`
	TpOrderType  OrderType `json:"tpOrderType,omitempty"`
	SlOrderType  OrderType `json:"slOrderType,omitempty"`
	TpLimitPrice string    `json:"tpLimitPrice,omitempty"`
	SlLimitPrice string    `json:"slLimitPrice,omitempty"`
}

// CancelOrderRequest cancels a single open order. Either OrderID or
// ClientOid must be set; if both are set, OrderID wins.
//
// Docs: https://www.bitget.com/api-doc/uta/trade/Cancel-Order
type CancelOrderRequest struct {
	OrderID   string   `json:"orderId,omitempty"`
	ClientOid string   `json:"clientOid,omitempty"`
	Category  Category `json:"category,omitempty"`
}

// FeeDetail is a single fee line item charged against an order.
type FeeDetail struct {
	FeeCoin string `json:"feeCoin"`
	Fee     string `json:"fee"`
}

// Order is a full order record, returned by the open-orders and
// order-history endpoints.
//
// Docs: https://www.bitget.com/api-doc/uta/trade/Get-Order-Pending
// Docs: https://www.bitget.com/api-doc/uta/trade/Get-Order-History
type Order struct {
	OrderID      string      `json:"orderId"`
	ClientOid    string      `json:"clientOid"`
	Category     string      `json:"category"`
	Symbol       string      `json:"symbol"`
	OrderType    string      `json:"orderType"`
	Side         string      `json:"side"`
	Price        string      `json:"price"`
	Qty          string      `json:"qty"`
	Amount       string      `json:"amount"`
	CumExecQty   string      `json:"cumExecQty"`
	CumExecValue string      `json:"cumExecValue"`
	AvgPrice     string      `json:"avgPrice"`
	TimeInForce  string      `json:"timeInForce"`
	OrderStatus  string      `json:"orderStatus"`
	PosSide      string      `json:"posSide"`
	HoldMode     string      `json:"holdMode"`
	DelegateType string      `json:"delegateType"`
	MarginMode   string      `json:"marginMode"`
	StpMode      string      `json:"stpMode"`
	TakeProfit   string      `json:"takeProfit"`
	StopLoss     string      `json:"stopLoss"`
	TpTriggerBy  string      `json:"tpTriggerBy"`
	SlTriggerBy  string      `json:"slTriggerBy"`
	TpOrderType  string      `json:"tpOrderType"`
	SlOrderType  string      `json:"slOrderType"`
	TpLimitPrice string      `json:"tpLimitPrice"`
	SlLimitPrice string      `json:"slLimitPrice"`
	ReduceOnly   string      `json:"reduceOnly"`
	FeeDetail    []FeeDetail `json:"feeDetail"`
	CreatedTime  string      `json:"createdTime"`
	UpdatedTime  string      `json:"updatedTime"`
}

// OrderList is the paginated envelope returned by the open-orders and
// order-history endpoints.
type OrderList struct {
	List   []Order `json:"list"`
	Cursor string  `json:"cursor"`
}

// Position is an open futures position.
//
// Docs: https://www.bitget.com/api-doc/uta/trade/Get-Position
type Position struct {
	Category         string `json:"category"`
	Symbol           string `json:"symbol"`
	MarginCoin       string `json:"marginCoin"`
	PosSide          string `json:"posSide"`
	PositionBalance  string `json:"positionBalance"`
	Available        string `json:"available"`
	Frozen           string `json:"frozen"`
	Total            string `json:"total"`
	Leverage         string `json:"leverage"`
	CurRealisedPnl   string `json:"curRealisedPnl"`
	AvgPrice         string `json:"avgPrice"`
	MarginMode       string `json:"marginMode"`
	PositionStatus   string `json:"positionStatus"`
	HoldMode         string `json:"holdMode"`
	UnrealisedPnl    string `json:"unrealisedPnl"`
	LiquidationPrice string `json:"liquidationPrice"`
	Mmr              string `json:"mmr"`
	ProfitRate       string `json:"profitRate"`
	MarkPrice        string `json:"markPrice"`
	BreakEvenPrice   string `json:"breakEvenPrice"`
	TotalFunding     string `json:"totalFunding"`
	OpenFeeTotal     string `json:"openFeeTotal"`
	CloseFeeTotal    string `json:"closeFeeTotal"`
	CashDividend     string `json:"cashDividend"`
	CreatedTime      string `json:"createdTime"`
	UpdatedTime      string `json:"updatedTime"`
}

// PositionList wraps the list of positions returned by Get Position Info.
type PositionList struct {
	List []Position `json:"list"`
}
