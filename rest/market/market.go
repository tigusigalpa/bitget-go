// Package market implements Bitget UTA v3's public market-data endpoints.
//
// Docs: https://www.bitget.com/api-doc/uta/public/Instruments
package market

import (
	"context"
	"net/http"

	"github.com/tigusigalpa/bitget-go/models"
)

// doPublicFunc matches the signature of Client.doPublic, injected so this
// package has no dependency on the root package (avoiding an import cycle).
type doPublicFunc func(ctx context.Context, method, path string, query map[string]string, result interface{}) error

// Client provides Bitget's public market-data endpoints.
type Client struct {
	doPublic doPublicFunc
}

// NewClient wires a market.Client to the root package's unauthenticated
// request function. Not normally called directly; use bitget.NewRestClient.
func NewClient(doPublic doPublicFunc) *Client {
	return &Client{doPublic: doPublic}
}

// GetInstruments returns trading-pair specifications for a product
// category, optionally filtered to a single symbol.
//
// Docs: https://www.bitget.com/api-doc/uta/public/Instruments
func (c *Client) GetInstruments(ctx context.Context, category string, symbol string) ([]models.Instrument, error) {
	query := map[string]string{"category": category}
	if symbol != "" {
		query["symbol"] = symbol
	}
	var result []models.Instrument
	if err := c.doPublic(ctx, http.MethodGet, "/api/v3/market/instruments", query, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetTickers returns 24h market statistics for a product category,
// optionally filtered to a single symbol.
//
// Docs: https://www.bitget.com/api-doc/uta/public/Tickers
func (c *Client) GetTickers(ctx context.Context, category string, symbol string) ([]models.Ticker, error) {
	query := map[string]string{"category": category}
	if symbol != "" {
		query["symbol"] = symbol
	}
	var result []models.Ticker
	if err := c.doPublic(ctx, http.MethodGet, "/api/v3/market/tickers", query, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetOrderBook returns the bid/ask depth snapshot for a symbol. limit
// controls the depth level (default 5, max 1000); pass "" to use the
// server default.
//
// Docs: https://www.bitget.com/api-doc/uta/public/OrderBook
func (c *Client) GetOrderBook(ctx context.Context, category, symbol, limit string) (*models.OrderBook, error) {
	query := map[string]string{"category": category, "symbol": symbol}
	if limit != "" {
		query["limit"] = limit
	}
	var result models.OrderBook
	if err := c.doPublic(ctx, http.MethodGet, "/api/v3/market/orderbook", query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetCandlesOptions contains the filters for GetCandles. Category, Symbol,
// and Interval are required by Bitget; timestamps are Unix milliseconds.
type GetCandlesOptions struct {
	Category   string
	Symbol     string
	Interval   string
	StartTime  string
	EndTime    string
	CandleType string
	Limit      string
}

// GetCandles returns up to 1,000 candlesticks. Bitget does not guarantee the
// newest candle is closed, so callers that require closed bars must verify it
// against their own clock and fetch it again after the interval boundary.
//
// Docs: https://www.bitget.com/docs/catalog/market/market-data
func (c *Client) GetCandles(ctx context.Context, opts GetCandlesOptions) ([]models.Candle, error) {
	query := optionalQuery(map[string]string{
		"category":  opts.Category,
		"symbol":    opts.Symbol,
		"interval":  opts.Interval,
		"startTime": opts.StartTime,
		"endTime":   opts.EndTime,
		"type":      opts.CandleType,
		"limit":     opts.Limit,
	})
	var result []models.Candle
	if err := c.doPublic(ctx, http.MethodGet, "/api/v3/market/candles", query, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetPublicFills returns recent public fills for a symbol. execId is Bitget's
// execution identity; the endpoint is a recent-data repair/fallback source,
// not a complete trade-history feed.
//
// Docs: https://www.bitget.com/docs/catalog/market/market-data
func (c *Client) GetPublicFills(ctx context.Context, category, symbol, limit string) ([]models.PublicFill, error) {
	query := optionalQuery(map[string]string{
		"category": category,
		"symbol":   symbol,
		"limit":    limit,
	})
	var result []models.PublicFill
	if err := c.doPublic(ctx, http.MethodGet, "/api/v3/market/fills", query, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetFundingRateHistoryOptions contains the filters for GetFundingRateHistory.
// Category and Symbol are required. Cursor is Bitget's page number and Limit
// is capped at 100 by Bitget.
type GetFundingRateHistoryOptions struct {
	Category string
	Symbol   string
	Cursor   string
	Limit    string
}

// GetFundingRateHistory returns realized funding-rate records. The settlement
// interval varies by symbol and is available from GetInstruments.
//
// Docs: https://www.bitget.com/legacy-docs/uta/public/Get-History-Funding-Rate
func (c *Client) GetFundingRateHistory(ctx context.Context, opts GetFundingRateHistoryOptions) (*models.FundingRateHistory, error) {
	query := optionalQuery(map[string]string{
		"category": opts.Category,
		"symbol":   opts.Symbol,
		"cursor":   opts.Cursor,
		"limit":    opts.Limit,
	})
	var result models.FundingRateHistory
	if err := c.doPublic(ctx, http.MethodGet, "/api/v3/market/history-fund-rate", query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func optionalQuery(values map[string]string) map[string]string {
	query := make(map[string]string, len(values))
	for key, value := range values {
		if value != "" {
			query[key] = value
		}
	}
	return query
}
