// Package account implements Bitget UTA v3's private account endpoints.
//
// Docs: https://www.bitget.com/api-doc/uta/account/Get-Account
package account

import (
	"context"
	"net/http"

	"github.com/tigusigalpa/bitget-go/models"
)

// doFunc matches the signature of Client.do, injected so this package has
// no dependency on the root package (avoiding an import cycle).
type doFunc func(ctx context.Context, method, path string, query map[string]string, body interface{}, result interface{}) error

// Client provides Bitget's private account endpoints.
type Client struct {
	do doFunc
}

// NewClient wires an account.Client to the root package's authenticated
// request function. Not normally called directly; use bitget.NewRestClient.
func NewClient(do doFunc) *Client {
	return &Client{do: do}
}

// GetAssets returns the unified account's aggregate equity, margin, and
// per-coin balances.
//
// Docs: https://www.bitget.com/api-doc/uta/account/Get-Account
func (c *Client) GetAssets(ctx context.Context) (*models.AccountAssets, error) {
	var result models.AccountAssets
	if err := c.do(ctx, http.MethodGet, "/api/v3/account/assets", nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSettings returns the unified account's mode plus per-symbol/coin
// leverage configuration.
//
// Docs: https://www.bitget.com/api-doc/uta/account/Get-Account-Setting
func (c *Client) GetSettings(ctx context.Context) (*models.AccountSettings, error) {
	var result models.AccountSettings
	if err := c.do(ctx, http.MethodGet, "/api/v3/account/settings", nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetLeverage configures leverage for futures or margin trading.
//
// Docs: https://www.bitget.com/api-doc/uta/account/Change-Leverage
func (c *Client) SetLeverage(ctx context.Context, req models.SetLeverageRequest) error {
	return c.do(ctx, http.MethodPost, "/api/v3/account/set-leverage", nil, req, nil)
}
