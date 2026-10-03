// Command demo_account reads account data with a Demo API key. It refuses to
// run unless BITGET_DEMO=1 is set, so copied code cannot query a production
// account by accident.
//
// Run:
//
//	BITGET_DEMO=1 BITGET_API_KEY=... BITGET_SECRET_KEY=... BITGET_PASSPHRASE=... go run ./examples/demo-account
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	bitget "github.com/tigusigalpa/bitget-go"
)

func main() {
	if os.Getenv("BITGET_DEMO") != "1" {
		log.Fatal("refusing to run: set BITGET_DEMO=1 and use a Demo API key")
	}

	apiKey := os.Getenv("BITGET_API_KEY")
	secretKey := os.Getenv("BITGET_SECRET_KEY")
	passphrase := os.Getenv("BITGET_PASSPHRASE")
	if apiKey == "" || secretKey == "" || passphrase == "" {
		log.Fatal("set BITGET_API_KEY, BITGET_SECRET_KEY, and BITGET_PASSPHRASE")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := bitget.NewRestClient(apiKey, secretKey, passphrase, bitget.WithDemoTrading())
	assets, err := client.Account.GetAssets(ctx)
	if err != nil {
		log.Fatalf("get demo account assets: %v", err)
	}

	fmt.Printf("demo account equity (USD): %s\n", assets.AccountEquity)
	for _, detail := range assets.Assets {
		fmt.Printf("%s available=%s equity=%s\n", detail.Coin, detail.Available, detail.Equity)
	}
}
