// Command websocket_example demonstrates connecting to Bitget's public
// WebSocket, subscribing to a channel, and handling reconnects.
//
// Run:
//
//	go run ./examples/websocket
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	bitget "github.com/tigusigalpa/bitget-go"
	"github.com/tigusigalpa/bitget-go/models"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := bitget.NewSlogLogger(slog.Default())
	ws := bitget.NewPublicWSClient(bitget.WithWSLogger(logger))

	if err := ws.Connect(ctx); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer func() {
		if err := ws.Close(); err != nil {
			log.Printf("close WebSocket: %v", err)
		}
	}()

	pushes, err := ws.Subscribe(ctx, models.WSArg{
		InstType: "SPOT",
		Topic:    "ticker",
		Symbol:   "BTCUSDT",
	})
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	fmt.Println("subscribed to SPOT BTCUSDT ticker, press Ctrl+C to exit")
	for {
		select {
		case <-ctx.Done():
			return
		case push, ok := <-pushes:
			if !ok {
				return
			}
			fmt.Printf("push: action=%s data=%s\n", push.Action, string(push.Data))
		}
	}
}
