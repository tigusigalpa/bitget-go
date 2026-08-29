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

	want := []marketCall{
		{http.MethodGet, "/api/v3/market/instruments", map[string]string{"category": "SPOT", "symbol": "BTCUSDT"}},
		{http.MethodGet, "/api/v3/market/tickers", map[string]string{"category": "SPOT"}},
		{http.MethodGet, "/api/v3/market/orderbook", map[string]string{"category": "SPOT", "symbol": "BTCUSDT"}},
		{http.MethodGet, "/api/v3/market/orderbook", map[string]string{"category": "SPOT", "symbol": "BTCUSDT", "limit": "100"}},
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
