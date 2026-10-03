// Command websocket_private demonstrates a Demo private WebSocket
// subscription. It keeps the connection open until Ctrl+C and does not place
// or cancel orders.
//
// Run:
//
//	BITGET_DEMO=1 BITGET_API_KEY=... BITGET_SECRET_KEY=... BITGET_PASSPHRASE=... go run ./examples/websocket-private
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	bitget "github.com/tigusigalpa/bitget-go"
	"github.com/tigusigalpa/bitget-go/models"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The demo endpoint is explicit. Do not rely on the production default for
	// a copied private-stream example.
	ws := bitget.NewPrivateWSClient(apiKey, secretKey, passphrase, bitget.WithWSURL(bitget.DemoPrivateWSURL))
	if err := ws.Connect(ctx); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer func() {
		if err := ws.Close(); err != nil {
			log.Printf("close WebSocket: %v", err)
		}
	}()

	pushes, err := ws.Subscribe(ctx, models.WSArg{
		InstType: "UTA",
		Topic:    "fast-fill",
		Symbol:   "default",
	})
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	fmt.Println("listening for demo fast-fill events; press Ctrl+C to exit")
	for {
		select {
		case <-ctx.Done():
			return
		case push, ok := <-pushes:
			if !ok {
				return
			}

			var fills []models.FastFill
			if err := json.Unmarshal(push.Data, &fills); err != nil {
				log.Printf("decode fast-fill payload: %v; raw=%s", err, push.Data)
				continue
			}
			for _, fill := range fills {
				fmt.Printf("fill %s %s %s @ %s\n", fill.Symbol, fill.Side, fill.ExecQty, fill.ExecPrice)
			}
		}
	}
}
