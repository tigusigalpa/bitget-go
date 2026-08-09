package bitget

import (
	"github.com/tigusigalpa/bitget-go/rest/account"
	"github.com/tigusigalpa/bitget-go/rest/market"
	"github.com/tigusigalpa/bitget-go/rest/trade"
)

// RestClient is the main entry point for the REST API, grouping every
// service under one struct. Construct it with NewRestClient.
type RestClient struct {
	*Client

	// Market provides public, unauthenticated market-data endpoints.
	Market *market.Client
	// Account provides private account/balance/leverage endpoints.
	Account *account.Client
	// Trade provides private order-placement and position endpoints.
	Trade *trade.Client
}

// NewRestClient creates a fully wired Bitget UTA v3 REST client. apiKey,
// secretKey, and passphrase come from the API key created in the Bitget
// web console; see https://www.bitget.com/api-doc/uta/guide.
//
// Public endpoints (RestClient.Market) work even with empty credentials.
func NewRestClient(apiKey, secretKey, passphrase string, opts ...Option) *RestClient {
	client := NewClient(apiKey, secretKey, passphrase, opts...)
	return &RestClient{
		Client:  client,
		Market:  market.NewClient(client.doPublic),
		Account: account.NewClient(client.do),
		Trade:   trade.NewClient(client.do),
	}
}
