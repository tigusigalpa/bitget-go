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
