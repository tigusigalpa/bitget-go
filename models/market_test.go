package models

import (
	"encoding/json"
	"testing"
)

func TestOrderBook_UnmarshalsAsksAndBids(t *testing.T) {
	raw := `{"a":[["50000.5","1.2"]],"b":[["49999.5","0.8"]],"ts":"1622185200000"}`
	var ob OrderBook
	if err := json.Unmarshal([]byte(raw), &ob); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(ob.Asks) != 1 || ob.Asks[0][0] != "50000.5" || ob.Asks[0][1] != "1.2" {
		t.Fatalf("unexpected asks: %+v", ob.Asks)
	}
	if len(ob.Bids) != 1 || ob.Bids[0][0] != "49999.5" {
		t.Fatalf("unexpected bids: %+v", ob.Bids)
	}
	if ob.Ts != "1622185200000" {
		t.Fatalf("unexpected ts: %s", ob.Ts)
	}
}

func TestBitgetResponse_UnmarshalsGenericData(t *testing.T) {
	raw := `{"code":"00000","msg":"success","requestTime":1622185200000,"data":[{"symbol":"BTCUSDT"}]}`
	var resp BitgetResponse[[]Ticker]
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != "00000" || len(resp.Data) != 1 || resp.Data[0].Symbol != "BTCUSDT" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
