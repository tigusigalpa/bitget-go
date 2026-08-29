package account

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/tigusigalpa/bitget-go/models"
)

func TestClientEndpoints(t *testing.T) {
	var paths []string
	client := NewClient(func(_ context.Context, method, path string, _ map[string]string, body interface{}, result interface{}) error {
		if method != http.MethodGet && method != http.MethodPost {
			t.Errorf("method = %s", method)
		}
		paths = append(paths, path)
		switch result := result.(type) {
		case *models.AccountAssets:
			result.AccountEquity = "10"
		case *models.AccountSettings:
			result.UID = "user-1"
		}
		if path == "/api/v3/account/set-leverage" && body.(models.SetLeverageRequest).Leverage != "5" {
			t.Errorf("unexpected request body: %#v", body)
		}
		return nil
	})

	assets, err := client.GetAssets(context.Background())
	if err != nil || assets.AccountEquity != "10" {
		t.Fatalf("GetAssets() = %#v, %v", assets, err)
	}
	settings, err := client.GetSettings(context.Background())
	if err != nil || settings.UID != "user-1" {
		t.Fatalf("GetSettings() = %#v, %v", settings, err)
	}
	if err := client.SetLeverage(context.Background(), models.SetLeverageRequest{Leverage: "5"}); err != nil {
		t.Fatal(err)
	}

	want := []string{"/api/v3/account/assets", "/api/v3/account/settings", "/api/v3/account/set-leverage"}
	for i := range want {
		if paths[i] != want[i] {
			t.Errorf("path %d = %s, want %s", i, paths[i], want[i])
		}
	}
}

func TestClientEndpointsPropagateErrors(t *testing.T) {
	errExpected := errors.New("request failed")
	client := NewClient(func(context.Context, string, string, map[string]string, interface{}, interface{}) error {
		return errExpected
	})

	if _, err := client.GetAssets(context.Background()); !errors.Is(err, errExpected) {
		t.Errorf("GetAssets() error = %v", err)
	}
	if _, err := client.GetSettings(context.Background()); !errors.Is(err, errExpected) {
		t.Errorf("GetSettings() error = %v", err)
	}
	if err := client.SetLeverage(context.Background(), models.SetLeverageRequest{}); !errors.Is(err, errExpected) {
		t.Errorf("SetLeverage() error = %v", err)
	}
}
