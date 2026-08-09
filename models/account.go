package models

// AssetHolding is a single coin balance within the unified account.
//
// Docs: https://www.bitget.com/api-doc/uta/account/Get-Account
type AssetHolding struct {
	Coin      string `json:"coin"`
	Equity    string `json:"equity"`
	UsdValue  string `json:"usdValue"`
	Balance   string `json:"balance"`
	Available string `json:"available"`
	Debt      string `json:"debt"`
	Locked    string `json:"locked"`
	Bonus     string `json:"bonus"`
}

// AccountAssets is the unified account's aggregate equity/margin snapshot.
//
// Docs: https://www.bitget.com/api-doc/uta/account/Get-Account
type AccountAssets struct {
	AccountEquity     string         `json:"accountEquity"`
	UsdtEquity        string         `json:"usdtEquity"`
	BtcEquity         string         `json:"btcEquity"`
	UnrealisedPnl     string         `json:"unrealisedPnl"`
	UsdtUnrealisedPnl string         `json:"usdtUnrealisedPnl"`
	BtcUnrealizedPnl  string         `json:"btcUnrealizedPnl"`
	EffEquity         string         `json:"effEquity"`
	Mmr               string         `json:"mmr"`
	Imr               string         `json:"imr"`
	MgnRatio          string         `json:"mgnRatio"`
	PositionMgnRatio  string         `json:"positionMgnRatio"`
	PositionValue     string         `json:"positionValue"`
	Leverage          string         `json:"leverage"`
	Assets            []AssetHolding `json:"assets"`
}

// SymbolConfig is a per-symbol futures leverage/margin-mode setting.
//
// Docs: https://www.bitget.com/api-doc/uta/account/Get-Account-Setting
type SymbolConfig struct {
	Category   string `json:"category"`
	Symbol     string `json:"symbol"`
	MarginMode string `json:"marginMode"`
	Leverage   string `json:"leverage"`
}

// CoinConfig is a per-coin margin-trading leverage setting.
//
// Docs: https://www.bitget.com/api-doc/uta/account/Get-Account-Setting
type CoinConfig struct {
	Coin     string `json:"coin"`
	Leverage string `json:"leverage"`
}

// AccountSettings describes the unified account's mode and per-symbol/coin
// leverage configuration.
//
// Docs: https://www.bitget.com/api-doc/uta/account/Get-Account-Setting
type AccountSettings struct {
	UID              string         `json:"uid"`
	AccountMode      string         `json:"accountMode"`
	AssetMode        string         `json:"assetMode"`
	AccountLevel     string         `json:"accountLevel"`
	HoldMode         string         `json:"holdMode"`
	StpMode          string         `json:"stpMode"`
	SymbolConfigList []SymbolConfig `json:"symbolConfigList"`
	CoinConfigList   []CoinConfig   `json:"coinConfigList"`
}

// SetLeverageRequest configures leverage for futures or margin trading.
//
// Docs: https://www.bitget.com/api-doc/uta/account/Change-Leverage
type SetLeverageRequest struct {
	Category      Category   `json:"category"`
	Symbol        string     `json:"symbol,omitempty"`
	Leverage      string     `json:"leverage,omitempty"`
	Coin          string     `json:"coin,omitempty"`
	PosSide       PosSide    `json:"posSide,omitempty"`
	MarginMode    MarginMode `json:"marginMode,omitempty"`
	LongLeverage  string     `json:"longLeverage,omitempty"`
	ShortLeverage string     `json:"shortLeverage,omitempty"`
}
