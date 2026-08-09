// Package trade implements Bitget UTA v3's private order and position
// endpoints.
//
// Docs: https://www.bitget.com/api-doc/uta/trade/Place-Order
package trade

import (
	"context"
	"net/http"

	"github.com/tigusigalpa/bitget-go/models"
)

// doFunc matches the signature of Client.do, injected so this package has
// no dependency on the root package (avoiding an import cycle).
type doFunc func(ctx context.Context, method, path string, query map[string]string, body interface{}, result interface{}) error

// Client provides Bitget's private trade endpoints.
type Client struct {
	do doFunc
}

// NewClient wires a trade.Client to the root package's authenticated
// request function. Not normally called directly; use bitget.NewRestClient.
func NewClient(do doFunc) *Client {
	return &Client{do: do}
}

// PlaceOrder submits a new spot, margin, or futures order.
//
// Docs: https://www.bitget.com/api-doc/uta/trade/Place-Order
func (c *Client) PlaceOrder(ctx context.Context, req models.PlaceOrderRequest) (*models.OrderRef, error) {
	var result models.OrderRef
	if err := c.do(ctx, http.MethodPost, "/api/v3/trade/place-order", nil, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ModifyOrder amends the price, quantity, or TP/SL of an open order.
//
// Docs: https://www.bitget.com/api-doc/uta/trade/Modify-Order
func (c *Client) ModifyOrder(ctx context.Context, req models.ModifyOrderRequest) (*models.OrderRef, error) {
	var result models.OrderRef
	if err := c.do(ctx, http.MethodPost, "/api/v3/trade/modify-order", nil, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CancelOrder cancels a single open order.
//
// Docs: https://www.bitget.com/api-doc/uta/trade/Cancel-Order
func (c *Client) CancelOrder(ctx context.Context, req models.CancelOrderRequest) (*models.OrderRef, error) {
	var result models.OrderRef
	if err := c.do(ctx, http.MethodPost, "/api/v3/trade/cancel-order", nil, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetOpenOrdersOptions are the optional filters for GetOpenOrders.
type GetOpenOrdersOptions struct {
	Category  string
	Symbol    string
	StartTime string
	EndTime   string
	Limit     string
	Cursor    string
}

// GetOpenOrders lists currently unfilled/partially-filled orders.
//
// Docs: https://www.bitget.com/api-doc/uta/trade/Get-Order-Pending
func (c *Client) GetOpenOrders(ctx context.Context, opts GetOpenOrdersOptions) (*models.OrderList, error) {
	query := optionsToQuery(opts.Category, opts.Symbol, opts.StartTime, opts.EndTime, opts.Limit, opts.Cursor)
	var result models.OrderList
	if err := c.do(ctx, http.MethodGet, "/api/v3/trade/unfilled-orders", query, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetOrderHistoryOptions are the filters for GetOrderHistory. Category is
// required by the endpoint.
type GetOrderHistoryOptions struct {
	Category  string
	Symbol    string
	StartTime string
	EndTime   string
	Limit     string
	Cursor    string
}

// GetOrderHistory lists historical (filled/cancelled) orders. The
// startTime/endTime window may not exceed 30 days, within a 90-day
// lookback.
//
// Docs: https://www.bitget.com/api-doc/uta/trade/Get-Order-History
func (c *Client) GetOrderHistory(ctx context.Context, opts GetOrderHistoryOptions) (*models.OrderList, error) {
	query := optionsToQuery(opts.Category, opts.Symbol, opts.StartTime, opts.EndTime, opts.Limit, opts.Cursor)
	var result models.OrderList
	if err := c.do(ctx, http.MethodGet, "/api/v3/trade/history-orders", query, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPositions returns open futures positions for a product category,
// optionally filtered by symbol and/or position side.
//
// Docs: https://www.bitget.com/api-doc/uta/trade/Get-Position
func (c *Client) GetPositions(ctx context.Context, category, symbol, posSide string) (*models.PositionList, error) {
	query := map[string]string{"category": category}
	if symbol != "" {
		query["symbol"] = symbol
	}
	if posSide != "" {
		query["posSide"] = posSide
	}
	var result models.PositionList
	if err := c.do(ctx, http.MethodGet, "/api/v3/position/current-position", query, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func optionsToQuery(category, symbol, startTime, endTime, limit, cursor string) map[string]string {
	query := map[string]string{}
	if category != "" {
		query["category"] = category
	}
	if symbol != "" {
		query["symbol"] = symbol
	}
	if startTime != "" {
		query["startTime"] = startTime
	}
	if endTime != "" {
		query["endTime"] = endTime
	}
	if limit != "" {
		query["limit"] = limit
	}
	if cursor != "" {
		query["cursor"] = cursor
	}
	return query
}
