// Command rest_example demonstrates public market data plus an optional,
// explicitly gated authenticated call.
//
// Run:
//
//	go run ./examples/rest
//
// To also exercise a private (authenticated) endpoint, set:
//
//	BITGET_API_KEY, BITGET_SECRET_KEY, BITGET_PASSPHRASE
//
// and opt in to demo/paper trading (never live) with BITGET_DEMO=1.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	bitget "github.com/tigusigalpa/bitget-go"
	"github.com/tigusigalpa/bitget-go/models"
)

func main() {
	ctx := context.Background()

	var opts []bitget.Option
	if os.Getenv("BITGET_DEMO") == "1" {
		opts = append(opts, bitget.WithDemoTrading())
	}

	client := bitget.NewRestClient(
		os.Getenv("BITGET_API_KEY"),
		os.Getenv("BITGET_SECRET_KEY"),
		os.Getenv("BITGET_PASSPHRASE"),
		opts...,
	)

	// Public endpoint: works without credentials.
	tickers, err := client.Market.GetTickers(ctx, models.CategorySpot, "BTCUSDT")
	if err != nil {
		log.Fatalf("get tickers: %v", err)
	}
	for _, t := range tickers {
		fmt.Printf("%s last=%s bid=%s ask=%s\n", t.Symbol, t.LastPrice, t.Bid1Price, t.Ask1Price)
	}

	book, err := client.Market.GetOrderBook(ctx, models.CategorySpot, "BTCUSDT", "5")
	if err != nil {
		log.Fatalf("get order book: %v", err)
	}
	fmt.Printf("best bid=%v best ask=%v\n", book.Bids[0], book.Asks[0])

	// Private endpoint: requires real credentials, so it's opt-in only.
	if os.Getenv("BITGET_API_KEY") == "" {
		fmt.Println("set BITGET_API_KEY/BITGET_SECRET_KEY/BITGET_PASSPHRASE to also fetch account assets")
		return
	}
	assets, err := client.Account.GetAssets(ctx)
	if err != nil {
		log.Fatalf("get account assets: %v", err)
	}
	fmt.Printf("account equity (USD): %s\n", assets.AccountEquity)

	// Order placement is destructive, so it requires two explicit opt-ins:
	// demo/paper trading AND BITGET_ENABLE_TRADING=1. Never wire this up to
	// production credentials without removing the demo gate deliberately.
	if os.Getenv("BITGET_DEMO") != "1" || os.Getenv("BITGET_ENABLE_TRADING") != "1" {
		fmt.Println("set BITGET_DEMO=1 and BITGET_ENABLE_TRADING=1 to also place a demo limit order")
		return
	}
	order, err := client.Trade.PlaceOrder(ctx, models.PlaceOrderRequest{
		Category:  models.CategorySpot,
		Symbol:    "BTCUSDT",
		Side:      models.SideBuy,
		OrderType: models.OrderTypeLimit,
		Price:     "10000", // deliberately far below market so it won't fill
		Qty:       "0.001",
	})
	if err != nil {
		log.Fatalf("place order: %v", err)
	}
	fmt.Printf("placed demo order: %s\n", order.OrderID)
}
