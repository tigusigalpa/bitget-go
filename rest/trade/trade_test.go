package trade

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/tigusigalpa/bitget-go/models"
)

func TestClientEndpoints(t *testing.T) {
	var paths []string
	var queries []map[string]string
	client := NewClient(func(_ context.Context, method, path string, query map[string]string, _ interface{}, result interface{}) error {
		paths = append(paths, method+" "+path)
		queries = append(queries, query)
		switch result := result.(type) {
		case *models.OrderRef:
			result.OrderID = "order-1"
		case *models.OrderList:
			result.List = []models.Order{{OrderID: "order-1"}}
		case *models.PositionList:
			result.List = []models.Position{{Symbol: "BTCUSDT"}}
		}
		return nil
	})

	ctx := context.Background()
	if order, err := client.PlaceOrder(ctx, models.PlaceOrderRequest{}); err != nil || order.OrderID != "order-1" {
		t.Fatalf("PlaceOrder() = %#v, %v", order, err)
	}
	if order, err := client.ModifyOrder(ctx, models.ModifyOrderRequest{}); err != nil || order.OrderID != "order-1" {
		t.Fatalf("ModifyOrder() = %#v, %v", order, err)
	}
	if order, err := client.CancelOrder(ctx, models.CancelOrderRequest{}); err != nil || order.OrderID != "order-1" {
		t.Fatalf("CancelOrder() = %#v, %v", order, err)
	}
	opts := GetOpenOrdersOptions{Category: "SPOT", Symbol: "BTCUSDT", StartTime: "1", EndTime: "2", Limit: "10", Cursor: "next"}
	if orders, err := client.GetOpenOrders(ctx, opts); err != nil || len(orders.List) != 1 {
		t.Fatalf("GetOpenOrders() = %#v, %v", orders, err)
	}
	if orders, err := client.GetOrderHistory(ctx, GetOrderHistoryOptions(opts)); err != nil || len(orders.List) != 1 {
		t.Fatalf("GetOrderHistory() = %#v, %v", orders, err)
	}
	if positions, err := client.GetPositions(ctx, "USDT-FUTURES", "BTCUSDT", "long"); err != nil || len(positions.List) != 1 {
		t.Fatalf("GetPositions() = %#v, %v", positions, err)
	}

	wantPaths := []string{
		http.MethodPost + " /api/v3/trade/place-order",
		http.MethodPost + " /api/v3/trade/modify-order",
		http.MethodPost + " /api/v3/trade/cancel-order",
		http.MethodGet + " /api/v3/trade/unfilled-orders",
		http.MethodGet + " /api/v3/trade/history-orders",
		http.MethodGet + " /api/v3/position/current-position",
	}
	for i := range wantPaths {
		if paths[i] != wantPaths[i] {
			t.Errorf("path %d = %s, want %s", i, paths[i], wantPaths[i])
		}
	}
	if got := queries[3]; len(got) != 6 || got["cursor"] != "next" {
		t.Errorf("GetOpenOrders query = %#v", got)
	}
	if got := queries[5]; len(got) != 3 || got["posSide"] != "long" {
		t.Errorf("GetPositions query = %#v", got)
	}
}

func TestClientEndpointsPropagateErrors(t *testing.T) {
	errExpected := errors.New("request failed")
	client := NewClient(func(context.Context, string, string, map[string]string, interface{}, interface{}) error {
		return errExpected
	})
	ctx := context.Background()
	if _, err := client.PlaceOrder(ctx, models.PlaceOrderRequest{}); !errors.Is(err, errExpected) {
		t.Errorf("PlaceOrder() error = %v", err)
	}
	if _, err := client.ModifyOrder(ctx, models.ModifyOrderRequest{}); !errors.Is(err, errExpected) {
		t.Errorf("ModifyOrder() error = %v", err)
	}
	if _, err := client.CancelOrder(ctx, models.CancelOrderRequest{}); !errors.Is(err, errExpected) {
		t.Errorf("CancelOrder() error = %v", err)
	}
	if _, err := client.GetOpenOrders(ctx, GetOpenOrdersOptions{}); !errors.Is(err, errExpected) {
		t.Errorf("GetOpenOrders() error = %v", err)
	}
	if _, err := client.GetOrderHistory(ctx, GetOrderHistoryOptions{}); !errors.Is(err, errExpected) {
		t.Errorf("GetOrderHistory() error = %v", err)
	}
	if _, err := client.GetPositions(ctx, "", "", ""); !errors.Is(err, errExpected) {
		t.Errorf("GetPositions() error = %v", err)
	}
}

func TestOptionsToQuery(t *testing.T) {
	if got := optionsToQuery("", "", "", "", "", ""); len(got) != 0 {
		t.Errorf("empty optionsToQuery() = %#v", got)
	}
}
