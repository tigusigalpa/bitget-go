package market

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/tigusigalpa/bitget-go/models"
)

// GetHistoryCandlesOptions selects one deep-history candle page. Category,
// Symbol and Interval are required. Limit is 1..100; empty uses Bitget's default.
// StartTime/EndTime are passed as exact Unix millisecond strings. Bitget describes
// them as after/before bounds, but does not guarantee equality semantics. A query
// spans at most 90 days, even though this route can reach data older than 90 days.
type GetHistoryCandlesOptions struct {
	Category   string
	Symbol     string
	Interval   string
	CandleType string
	StartTime  string
	EndTime    string
	Limit      string
}

// GetHistoryCandles fetches /api/v3/market/history-candles, not recent candles.
// It preserves the provider's order and all returned candles, including an extra
// early interval caused by endTime rounding. It does not trim, round, paginate,
// deduplicate or assert complete coverage. See GetHistoryCandlesWithReceipt for
// raw response evidence.
//
// Docs: https://www.bitget.com/docs/catalog/market/market-data#get-klinecandlestick-history
func (c *Client) GetHistoryCandles(ctx context.Context, opts GetHistoryCandlesOptions) ([]models.Candle, error) {
	data, _, err := c.GetHistoryCandlesWithReceipt(ctx, opts)
	return data, err
}

// GetHistoryCandlesWithReceipt also returns bounded immutable response evidence
// on API, decoding and body-read errors. The receipt is nil for local validation
// or a failure before an HTTP response. Callers must check err before using data.
func (c *Client) GetHistoryCandlesWithReceipt(ctx context.Context, opts GetHistoryCandlesOptions) ([]models.Candle, *models.RESTReceipt, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if opts.Category == "" || opts.Symbol == "" || opts.Interval == "" {
		return nil, nil, fmt.Errorf("market: history candles require category, symbol and interval")
	}
	if err := validatePageLimit(opts.Limit); err != nil {
		return nil, nil, err
	}
	query := optionalQuery(map[string]string{
		"category": opts.Category, "symbol": opts.Symbol, "interval": opts.Interval,
		"type": opts.CandleType, "startTime": opts.StartTime, "endTime": opts.EndTime, "limit": opts.Limit,
	})
	var data historyCandles
	receipt, err := c.historyRequest(ctx, "/api/v3/market/history-candles", query, &data)
	if err != nil {
		return nil, receipt, err
	}
	return []models.Candle(data), receipt, nil
}

// GetLiquidationsOptions selects one partial liquidation page. Category is
// required; Symbol is optional. Cursor is opaque and must come from the previous
// response. Limit is 1..100; an empty string uses Bitget's default.
type GetLiquidationsOptions struct {
	Category string
	Symbol   string
	Cursor   string
	Limit    string
}

// GetLiquidations returns explicitly partial liquidation observations: the
// provider exposes only the last three days with possible delay and no stable
// event ID or documented amount units/REST aggregation guarantee. Neither the
// response cursor nor a timestamp is an execution identity. Do not interpret
// these pages as a complete liquidation history.
//
// Docs: https://www.bitget.com/docs/catalog/market/derivatives#get-liquidations-history
func (c *Client) GetLiquidations(ctx context.Context, opts GetLiquidationsOptions) (*models.PartialLiquidations, error) {
	data, _, err := c.GetLiquidationsWithReceipt(ctx, opts)
	return data, err
}

// GetLiquidationsWithReceipt also preserves the exact bounded response envelope,
// including unknown provider fields and the original opaque cursor.
func (c *Client) GetLiquidationsWithReceipt(ctx context.Context, opts GetLiquidationsOptions) (*models.PartialLiquidations, *models.RESTReceipt, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if opts.Category == "" {
		return nil, nil, fmt.Errorf("market: liquidations require category")
	}
	if err := validatePageLimit(opts.Limit); err != nil {
		return nil, nil, err
	}
	query := optionalQuery(map[string]string{"category": opts.Category, "symbol": opts.Symbol, "cursor": opts.Cursor, "limit": opts.Limit})
	var data liquidationsPage
	receipt, err := c.historyRequest(ctx, "/api/v3/market/liquidations", query, &data)
	if err != nil {
		return nil, receipt, err
	}
	result := models.PartialLiquidations(data)
	return &result, receipt, nil
}

func (c *Client) historyRequest(ctx context.Context, path string, query map[string]string, result interface{}) (*models.RESTReceipt, error) {
	if c.doPublicWithReceipt != nil {
		return c.doPublicWithReceipt(ctx, http.MethodGet, path, query, result)
	}
	// Legacy custom service constructors still work, but cannot manufacture raw
	// evidence from decoded results. NewRestClient always supplies receipt support.
	return nil, c.doPublic(ctx, http.MethodGet, path, query, result)
}

func validatePageLimit(limit string) error {
	if limit == "" {
		return nil
	}
	value, err := strconv.Atoi(limit)
	if err != nil || value < 1 || value > 100 {
		return fmt.Errorf("market: history page limit must be between 1 and 100")
	}
	return nil
}

// Strict parsing is local to the new method, so GetCandles stays compatible.
type historyCandles []models.Candle

func (c *historyCandles) UnmarshalJSON(payload []byte) error {
	var rows [][]json.RawMessage
	if err := json.Unmarshal(payload, &rows); err != nil {
		return err
	}
	if rows == nil {
		return fmt.Errorf("market: expected history candle array, got null")
	}
	result := make(historyCandles, len(rows))
	for i, row := range rows {
		if len(row) != 7 {
			return fmt.Errorf("market: history candle %d has %d fields, expected 7", i, len(row))
		}
		for j, value := range row {
			value = bytes.TrimSpace(value)
			if len(value) == 0 || value[0] != '"' {
				return fmt.Errorf("market: history candle %d field %d is not a numeric string", i, j)
			}
			if err := json.Unmarshal(value, &result[i][j]); err != nil {
				return err
			}
		}
	}
	*c = result
	return nil
}

type liquidationsPage models.PartialLiquidations

func (p *liquidationsPage) UnmarshalJSON(payload []byte) error {
	var fields struct {
		List   json.RawMessage `json:"list"`
		Cursor *string         `json:"cursor"`
	}
	if err := json.Unmarshal(payload, &fields); err != nil {
		return err
	}
	if len(fields.List) == 0 || bytes.Equal(bytes.TrimSpace(fields.List), []byte("null")) || fields.Cursor == nil {
		return fmt.Errorf("market: liquidation page requires list array and cursor string")
	}
	var list []models.Liquidation
	if err := json.Unmarshal(fields.List, &list); err != nil {
		return err
	}
	*p = liquidationsPage{List: list, Cursor: *fields.Cursor}
	return nil
}
