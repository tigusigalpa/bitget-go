package market

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tigusigalpa/bitget-go/models"
)

func TestHistoryServiceValidationAndCancellation(t *testing.T) {
	calls := 0
	client := NewClient(func(context.Context, string, string, map[string]string, interface{}) error {
		calls++
		return nil
	})
	for _, limit := range []string{"0", "101", "1000", "-1", "bad", "1.5"} {
		_, receipt, err := client.GetHistoryCandlesWithReceipt(context.Background(), GetHistoryCandlesOptions{Category: "SPOT", Symbol: "BTCUSDT", Interval: "1m", Limit: limit})
		require.ErrorContains(t, err, "between 1 and 100")
		assert.Nil(t, receipt)
		_, receipt, err = client.GetLiquidationsWithReceipt(context.Background(), GetLiquidationsOptions{Category: "USDT-FUTURES", Limit: limit})
		require.Error(t, err)
		assert.Nil(t, receipt)
	}
	for _, opts := range []GetHistoryCandlesOptions{
		{}, {Category: "SPOT", Symbol: "BTCUSDT"}, {Category: "SPOT", Interval: "1m"}, {Symbol: "BTCUSDT", Interval: "1m"},
	} {
		_, err := client.GetHistoryCandles(context.Background(), opts)
		require.ErrorContains(t, err, "require category, symbol and interval")
	}
	_, err := client.GetLiquidations(context.Background(), GetLiquidationsOptions{})
	require.ErrorContains(t, err, "require category")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.GetHistoryCandles(ctx, GetHistoryCandlesOptions{})
	assert.ErrorIs(t, err, context.Canceled)
	_, err = client.GetLiquidations(ctx, GetLiquidationsOptions{})
	assert.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, calls)
}

func TestNewHistoryMethodsSupportLegacyInjectedService(t *testing.T) {
	calls := 0
	client := NewClient(func(ctx context.Context, method, path string, query map[string]string, result interface{}) error {
		calls++
		assert.NotNil(t, ctx.Value(historyContextKey{}))
		assert.Equal(t, "GET", method)
		assert.Equal(t, map[string]string{"category": "SPOT", "symbol": "BTCUSDT", "interval": "1m", "limit": "100"}, query)
		assert.Equal(t, "/api/v3/market/history-candles", path)
		return json.Unmarshal([]byte(`[["1","2","3","4","5","6","7"]]`), result)
	})
	ctx := context.WithValue(context.Background(), historyContextKey{}, "injected")
	data, receipt, err := client.GetHistoryCandlesWithReceipt(ctx, GetHistoryCandlesOptions{Category: "SPOT", Symbol: "BTCUSDT", Interval: "1m", Limit: "100"})
	require.NoError(t, err)
	assert.Nil(t, receipt, "do not manufacture raw evidence from typed results")
	require.Len(t, data, 1)
	assert.Equal(t, "5", data[0][4])
	assert.Equal(t, 1, calls)
}

type historyContextKey struct{}

func TestHistoryServiceReceiptErrorsAndOptionalSelectors(t *testing.T) {
	wantErr := errors.New("transport failed")
	client := NewClientWithReceipts(nil, func(ctx context.Context, method, path string, query map[string]string, _ interface{}) (*models.RESTReceipt, error) {
		assert.NotNil(t, ctx.Value(historyContextKey{}))
		assert.Equal(t, "GET", method)
		if path == "/api/v3/market/liquidations" {
			assert.Equal(t, map[string]string{"category": "USDT-FUTURES"}, query)
		} else {
			assert.Equal(t, map[string]string{"category": "SPOT", "symbol": "BTCUSDT", "interval": "1m"}, query)
		}
		return nil, wantErr
	})
	ctx := context.WithValue(context.Background(), historyContextKey{}, "injected")
	_, err := client.GetHistoryCandles(ctx, GetHistoryCandlesOptions{Category: "SPOT", Symbol: "BTCUSDT", Interval: "1m"})
	assert.ErrorIs(t, err, wantErr)
	_, err = client.GetLiquidations(ctx, GetLiquidationsOptions{Category: "USDT-FUTURES"})
	assert.ErrorIs(t, err, wantErr)
}

func TestHistoryCandlesStrictParser(t *testing.T) {
	for _, payload := range []string{`null`, `[null]`, `[[]]`, `{}`, `[["1", "2"]]`, `[[1,"2","3","4","5","6","7"]]`, `[[null,"2","3","4","5","6","7"]]`} {
		var data historyCandles
		assert.Error(t, json.Unmarshal([]byte(payload), &data))
	}
	var data historyCandles
	require.NoError(t, json.Unmarshal([]byte(`[]`), &data))
	assert.NotNil(t, data)
}

func TestLiquidationsStrictPageParser(t *testing.T) {
	for _, payload := range []string{`null`, `{}`, `{"list":null,"cursor":""}`, `{"list":[],"cursor":null}`, `{"list":[],"cursor":123}`, `{"list":{},"cursor":""}`, `{"list":[{"price":1}],"cursor":""}`} {
		var page liquidationsPage
		assert.Error(t, json.Unmarshal([]byte(payload), &page), payload)
	}
	var page liquidationsPage
	require.NoError(t, json.Unmarshal([]byte(`{"list":[],"cursor":""}`), &page))
	assert.NotNil(t, page.List)
}

func TestLiquidationsLegacyConstructorPreservesPageAndCursor(t *testing.T) {
	client := NewClient(func(_ context.Context, method, path string, query map[string]string, result interface{}) error {
		assert.Equal(t, "GET", method)
		assert.Equal(t, "/api/v3/market/liquidations", path)
		assert.Equal(t, map[string]string{"category": "USDT-FUTURES"}, query)
		return json.Unmarshal([]byte(`{"list":[{"symbol":"BTCUSDT","price":"1.0000000001","amount":"0.00001000","ts":"9007199254740993"}],"cursor":"opaque/+"}`), result)
	})
	page, err := client.GetLiquidations(context.Background(), GetLiquidationsOptions{Category: "USDT-FUTURES"})
	require.NoError(t, err)
	require.Len(t, page.List, 1)
	assert.Equal(t, "opaque/+", page.Cursor)
	assert.Equal(t, "1.0000000001", page.List[0].Price)
	assert.Equal(t, "0.00001000", page.List[0].Amount)
	assert.Equal(t, "9007199254740993", page.List[0].Ts)
}
