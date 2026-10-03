// Command rest_example demonstrates public market data. It needs no API
// credentials and never changes an account.
//
// Run:
//
//	go run ./examples/rest
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	bitget "github.com/tigusigalpa/bitget-go"
	"github.com/tigusigalpa/bitget-go/models"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Public endpoints work with empty credentials.
	client := bitget.NewRestClient("", "", "")

	tickers, err := client.Market.GetTickers(ctx, models.CategorySpot, "BTCUSDT")
	if err != nil {
		log.Fatalf("get tickers: %v", err)
	}
	if len(tickers) == 0 {
		log.Fatal("Bitget returned no ticker for BTCUSDT")
	}
	t := tickers[0]
	fmt.Printf("%s: last=%s bid=%s ask=%s\n", t.Symbol, t.LastPrice, t.Bid1Price, t.Ask1Price)

	book, err := client.Market.GetOrderBook(ctx, models.CategorySpot, "BTCUSDT", "5")
	if err != nil {
		log.Fatalf("get order book: %v", err)
	}
	if len(book.Bids) > 0 {
		fmt.Printf("best bid: price=%s quantity=%s\n", book.Bids[0][0], book.Bids[0][1])
	}
	if len(book.Asks) > 0 {
		fmt.Printf("best ask: price=%s quantity=%s\n", book.Asks[0][0], book.Asks[0][1])
	}
}
