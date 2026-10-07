// Command history fetches one public deep-history candle page and reports its
// bounded receipt. Run: go run ./examples/history
package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"time"

	bitget "github.com/tigusigalpa/bitget-go"
	"github.com/tigusigalpa/bitget-go/models"
	"github.com/tigusigalpa/bitget-go/rest/market"
)

func main() {
	start := flag.String("start", "1609459200000", "unchanged Unix millisecond start selector")
	end := flag.String("end", "1609465200000", "unchanged Unix millisecond end selector")
	flag.Parse()
	if err := run(*start, *end); err != nil {
		log.Fatal(err)
	}
}

func run(start, end string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := bitget.NewRestClient("", "", "")
	candles, receipt, err := client.Market.GetHistoryCandlesWithReceipt(ctx, market.GetHistoryCandlesOptions{
		Category: models.CategorySpot, Symbol: "BTCUSDT", Interval: "1m", CandleType: "market",
		StartTime: start, EndTime: end, Limit: "100",
	})
	if receipt != nil {
		payload := receipt.Payload()
		fmt.Printf("HTTP=%d received=%s complete_body=%t bytes=%d sha256=%x\n",
			receipt.StatusCode(), receipt.ReceivedAt().Format(time.RFC3339Nano), receipt.Complete(), len(payload), sha256.Sum256(payload))
	}
	if err != nil {
		return err
	}
	for _, candle := range candles {
		fmt.Printf("timestamp=%s close=%s base_volume=%s quote_turnover=%s\n", candle[0], candle[4], candle[5], candle[6])
	}
	return nil
}
