package models

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRESTReceiptOwnsCopiesAndSupportsConcurrentReaders(t *testing.T) {
	payload := []byte("exact original bytes")
	metadata := RESTRequestMetadata{Method: "GET", Path: "/api/v3/market/history-candles", Query: map[string]string{"limit": "100"}}
	now := time.Now()
	receipt := NewRESTReceipt(metadata, payload, 200, now, true)
	payload[0] = 'x'
	metadata.Query["limit"] = "200"
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bytes := receipt.Payload()
			request := receipt.Request()
			assert.Equal(t, "exact original bytes", string(bytes))
			assert.Equal(t, "100", request.Query["limit"])
			bytes[0] = 'x'
			request.Query["limit"] = "1"
		}()
	}
	wg.Wait()
	assert.Equal(t, "exact original bytes", string(receipt.Payload()))
	assert.Equal(t, "100", receipt.Request().Query["limit"])
	assert.Equal(t, 200, receipt.StatusCode())
	assert.Equal(t, now, receipt.ReceivedAt())
	assert.True(t, receipt.Complete())
}
