package models

// Instrument is a single trading pair's specification.
//
// Docs: https://www.bitget.com/api-doc/uta/public/Instruments
type Instrument struct {
	Symbol                     string `json:"symbol"`
	Category                   string `json:"category"`
	BaseCoin                   string `json:"baseCoin"`
	QuoteCoin                  string `json:"quoteCoin"`
	IsRwa                      string `json:"isRwa"`
	IsReality                  string `json:"isReality"`
	BuyLimitPriceRatio         string `json:"buyLimitPriceRatio"`
	SellLimitPriceRatio        string `json:"sellLimitPriceRatio"`
	MinOrderQty                string `json:"minOrderQty"`
	MaxOrderQty                string `json:"maxOrderQty"`
	MinOrderAmount             string `json:"minOrderAmount"`
	PricePrecision             string `json:"pricePrecision"`
	QuantityPrecision          string `json:"quantityPrecision"`
	QuotePrecision             string `json:"quotePrecision"`
	PriceMultiplier            string `json:"priceMultiplier"`
	QuantityMultiplier         string `json:"quantityMultiplier"`
	Type                       string `json:"type"`
	MakerFeeRate               string `json:"makerFeeRate"`
	TakerFeeRate               string `json:"takerFeeRate"`
	FeeRateUpRatio             string `json:"feeRateUpRatio"`
	OpenCostUpRatio            string `json:"openCostUpRatio"`
	MaxSymbolOrderNum          string `json:"maxSymbolOrderNum"`
	MaxProductOrderNum         string `json:"maxProductOrderNum"`
	MaxPositionNum             string `json:"maxPositionNum"`
	Status                     string `json:"status"`
	OffTime                    string `json:"offTime"`
	LimitOpenTime              string `json:"limitOpenTime"`
	DeliveryTime               string `json:"deliveryTime"`
	DeliveryStartTime          string `json:"deliveryStartTime"`
	DeliveryPeriod             string `json:"deliveryPeriod"`
	LaunchTime                 string `json:"launchTime"`
	FundInterval               string `json:"fundInterval"`
	MinLeverage                string `json:"minLeverage"`
	MaxLeverage                string `json:"maxLeverage"`
	MaxMarketOrderQty          string `json:"maxMarketOrderQty"`
	MaintainTime               string `json:"maintainTime"`
	IsIsolatedBaseBorrowable   string `json:"isIsolatedBaseBorrowable"`
	IsIsolatedQuotedBorrowable string `json:"isIsolatedQuotedBorrowable"`
	WarningRiskRatio           string `json:"warningRiskRatio"`
	LiquidationRiskRatio       string `json:"liquidationRiskRatio"`
	MaxCrossedLeverage         string `json:"maxCrossedLeverage"`
	MaxIsolatedLeverage        string `json:"maxIsolatedLeverage"`
	UserMinBorrow              string `json:"userMinBorrow"`
	AreaSymbol                 string `json:"areaSymbol"`
	SymbolType                 string `json:"symbolType"`
}

// Ticker is 24h market statistics for a single symbol.
//
// Docs: https://www.bitget.com/api-doc/uta/public/Tickers
type Ticker struct {
	Category            string `json:"category"`
	Symbol              string `json:"symbol"`
	LastPrice           string `json:"lastPrice"`
	OpenPrice24h        string `json:"openPrice24h"`
	HighPrice24h        string `json:"highPrice24h"`
	LowPrice24h         string `json:"lowPrice24h"`
	Ask1Price           string `json:"ask1Price"`
	Bid1Price           string `json:"bid1Price"`
	Bid1Size            string `json:"bid1Size"`
	Ask1Size            string `json:"ask1Size"`
	Price24hPcnt        string `json:"price24hPcnt"`
	Volume24h           string `json:"volume24h"`
	Turnover24h         string `json:"turnover24h"`
	PlatformTurnover24h string `json:"platformTurnover24h"`
	IndexPrice          string `json:"indexPrice"`
	MarkPrice           string `json:"markPrice"`
	FundingRate         string `json:"fundingRate"`
	OpenInterest        string `json:"openInterest"`
	DeliveryStartTime   string `json:"deliveryStartTime"`
	DeliveryTime        string `json:"deliveryTime"`
	DeliveryStatus      string `json:"deliveryStatus"`
	Ts                  string `json:"ts"`
}

// OrderBookLevel is a single [price, quantity] entry.
type OrderBookLevel [2]string

// OrderBook is a snapshot of the bid/ask depth for a symbol.
//
// Docs: https://www.bitget.com/api-doc/uta/public/OrderBook
type OrderBook struct {
	Asks []OrderBookLevel `json:"a"`
	Bids []OrderBookLevel `json:"b"`
	Ts   string           `json:"ts"`
}
