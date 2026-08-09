//go:build integration

package bitget

import (
	"context"
	"os"
	"testing"

	"github.com/tigusigalpa/bitget-go/models"
)

// TestIntegration_GetTickers hits Bitget's live public market-data endpoint.
// Run with: go test -tags=integration ./...
func TestIntegration_GetTickers(t *testing.T) {
	client := NewRestClient("", "", "")
	tickers, err := client.Market.GetTickers(context.Background(), models.CategorySpot, "BTCUSDT")
	if err != nil {
		t.Fatalf("get tickers: %v", err)
	}
	if len(tickers) == 0 {
		t.Fatal("expected at least one ticker")
	}
}

// TestIntegration_GetAccountAssets hits a private endpoint against Bitget's
// demo/paper-trading environment. Requires BITGET_API_KEY, BITGET_SECRET_KEY,
// and BITGET_PASSPHRASE env vars for a Demo API key; skipped otherwise.
func TestIntegration_GetAccountAssets(t *testing.T) {
	apiKey := os.Getenv("BITGET_API_KEY")
	secretKey := os.Getenv("BITGET_SECRET_KEY")
	passphrase := os.Getenv("BITGET_PASSPHRASE")
	if apiKey == "" || secretKey == "" || passphrase == "" {
		t.Skip("BITGET_API_KEY/BITGET_SECRET_KEY/BITGET_PASSPHRASE not set")
	}

	client := NewRestClient(apiKey, secretKey, passphrase, WithDemoTrading())
	if _, err := client.Account.GetAssets(context.Background()); err != nil {
		t.Fatalf("get account assets: %v", err)
	}
}
