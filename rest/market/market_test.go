package market

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/tigusigalpa/bitget-go/models"
)

type marketCall struct {
	method string
	path   string
	query  map[string]string
}

func TestClientEndpoints(t *testing.T) {
	var calls []marketCall
	client := NewClient(func(_ context.Context, method, path string, query map[string]string, result interface{}) error {
		calls = append(calls, marketCall{method: method, path: path, query: query})
		switch result := result.(type) {
		case *[]models.Instrument:
			*result = []models.Instrument{{Symbol: "BTCUSDT"}}
		case *[]models.Ticker:
			*result = []models.Ticker{{Symbol: "BTCUSDT"}}
		case *models.OrderBook:
			*result = models.OrderBook{Bids: []models.OrderBookLevel{{"1", "2"}}}
		case *[]models.Candle:
			*result = []models.Candle{{"1", "2", "3", "4", "5", "6", "7"}}
		case *[]models.PublicFill:
			*result = []models.PublicFill{{ExecID: "fill-1"}}
		case *models.FundingRateHistory:
			*result = models.FundingRateHistory{ResultList: []models.FundingRate{{FundingRate: "0.0001"}}}
		}
		return nil
	})

	instruments, err := client.GetInstruments(context.Background(), models.CategorySpot, "BTCUSDT")
	if err != nil || len(instruments) != 1 || instruments[0].Symbol != "BTCUSDT" {
		t.Fatalf("GetInstruments() = %#v, %v", instruments, err)
	}
	tickers, err := client.GetTickers(context.Background(), models.CategorySpot, "")
	if err != nil || len(tickers) != 1 || tickers[0].Symbol != "BTCUSDT" {
		t.Fatalf("GetTickers() = %#v, %v", tickers, err)
	}
	book, err := client.GetOrderBook(context.Background(), models.CategorySpot, "BTCUSDT", "")
	if err != nil || len(book.Bids) != 1 {
		t.Fatalf("GetOrderBook() = %#v, %v", book, err)
	}
	_, err = client.GetOrderBook(context.Background(), models.CategorySpot, "BTCUSDT", "100")
	if err != nil {
		t.Fatal(err)
	}
	candles, err := client.GetCandles(context.Background(), GetCandlesOptions{
		Category: "SPOT", Symbol: "BTCUSDT", Interval: "1m", StartTime: "1", EndTime: "2", CandleType: "market", Limit: "100",
	})
	if err != nil || len(candles) != 1 || candles[0][4] != "5" {
		t.Fatalf("GetCandles() = %#v, %v", candles, err)
	}
	fills, err := client.GetPublicFills(context.Background(), "SPOT", "BTCUSDT", "100")
	if err != nil || len(fills) != 1 || fills[0].ExecID != "fill-1" {
		t.Fatalf("GetPublicFills() = %#v, %v", fills, err)
	}
	funding, err := client.GetFundingRateHistory(context.Background(), GetFundingRateHistoryOptions{Category: "USDT-FUTURES", Symbol: "BTCUSDT", Cursor: "2", Limit: "10"})
	if err != nil || len(funding.ResultList) != 1 || funding.ResultList[0].FundingRate != "0.0001" {
		t.Fatalf("GetFundingRateHistory() = %#v, %v", funding, err)
	}

	want := []marketCall{
		{http.MethodGet, "/api/v3/market/instruments", map[string]string{"category": "SPOT", "symbol": "BTCUSDT"}},
		{http.MethodGet, "/api/v3/market/tickers", map[string]string{"category": "SPOT"}},
		{http.MethodGet, "/api/v3/market/orderbook", map[string]string{"category": "SPOT", "symbol": "BTCUSDT"}},
		{http.MethodGet, "/api/v3/market/orderbook", map[string]string{"category": "SPOT", "symbol": "BTCUSDT", "limit": "100"}},
		{http.MethodGet, "/api/v3/market/candles", map[string]string{"category": "SPOT", "symbol": "BTCUSDT", "interval": "1m", "startTime": "1", "endTime": "2", "type": "market", "limit": "100"}},
		{http.MethodGet, "/api/v3/market/fills", map[string]string{"category": "SPOT", "symbol": "BTCUSDT", "limit": "100"}},
		{http.MethodGet, "/api/v3/market/history-fund-rate", map[string]string{"category": "USDT-FUTURES", "symbol": "BTCUSDT", "cursor": "2", "limit": "10"}},
	}
	if len(calls) != len(want) {
		t.Fatalf("calls = %d, want %d", len(calls), len(want))
	}
	for i := range want {
		if calls[i].method != want[i].method || calls[i].path != want[i].path || !sameQuery(calls[i].query, want[i].query) {
			t.Errorf("call %d = %#v, want %#v", i, calls[i], want[i])
		}
	}
}

func TestClientEndpointsPropagateErrors(t *testing.T) {
	errExpected := errors.New("request failed")
	client := NewClient(func(context.Context, string, string, map[string]string, interface{}) error { return errExpected })

	if _, err := client.GetInstruments(context.Background(), models.CategorySpot, ""); !errors.Is(err, errExpected) {
		t.Errorf("GetInstruments() error = %v", err)
	}
	if _, err := client.GetTickers(context.Background(), models.CategorySpot, ""); !errors.Is(err, errExpected) {
		t.Errorf("GetTickers() error = %v", err)
	}
	if _, err := client.GetOrderBook(context.Background(), models.CategorySpot, "BTCUSDT", ""); !errors.Is(err, errExpected) {
		t.Errorf("GetOrderBook() error = %v", err)
	}
	if _, err := client.GetCandles(context.Background(), GetCandlesOptions{}); !errors.Is(err, errExpected) {
		t.Errorf("GetCandles() error = %v", err)
	}
	if _, err := client.GetPublicFills(context.Background(), "SPOT", "BTCUSDT", ""); !errors.Is(err, errExpected) {
		t.Errorf("GetPublicFills() error = %v", err)
	}
	if _, err := client.GetFundingRateHistory(context.Background(), GetFundingRateHistoryOptions{}); !errors.Is(err, errExpected) {
		t.Errorf("GetFundingRateHistory() error = %v", err)
	}
}

func sameQuery(got, want map[string]string) bool {
	if len(got) != len(want) {
		return false
	}
	for key, wantValue := range want {
		if got[key] != wantValue {
			return false
		}
	}
	return true
}
