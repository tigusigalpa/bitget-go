package bitget

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tigusigalpa/bitget-go/models"
)

// TestWsLoginSign_KnownVector verifies the WebSocket login signature
// ("timestamp" + "GET" + "/user/verify") against a vector computed
// independently via `openssl dgst -sha256 -hmac`.
//
// Docs: https://www.bitget.com/api-doc/uta/guide
func TestWsLoginSign_KnownVector(t *testing.T) {
	got := wsLoginSign("test-secret-key", "1622185200000")
	assert.Equal(t, "SEw02Qwy8zXY/tbcv5X722/TFqcGZkar/n+Pcvkz450=", got)
}

func TestWsSubKey_IsStableForSameArgs(t *testing.T) {
	a := models.WSArg{InstType: "UTA", Topic: "fast-fill", Symbol: "default"}
	b := models.WSArg{InstType: "UTA", Topic: "fast-fill", Symbol: "default"}
	assert.Equal(t, wsSubKey(a), wsSubKey(b))
}

func TestWsSubKey_DiffersByTopic(t *testing.T) {
	a := models.WSArg{InstType: "UTA", Topic: "fast-fill", Symbol: "default"}
	b := models.WSArg{InstType: "UTA", Topic: "ticker", Symbol: "default"}
	assert.NotEqual(t, wsSubKey(a), wsSubKey(b))
}

func TestNewPublicWSClient_DefaultsURL(t *testing.T) {
	c := NewPublicWSClient()
	assert.Equal(t, DefaultPublicWSURL, c.url)
	assert.False(t, c.private)
}

func TestNewPrivateWSClient_DefaultsURLAndCredentials(t *testing.T) {
	c := NewPrivateWSClient("key", "secret", "pass")
	assert.Equal(t, DefaultPrivateWSURL, c.url)
	assert.True(t, c.private)
	assert.Equal(t, "key", c.apiKey)
}

func TestWithWSURL_Overrides(t *testing.T) {
	c := NewPublicWSClient(WithWSURL(DemoPublicWSURL))
	assert.Equal(t, DemoPublicWSURL, c.url)
}

func TestNilLoggersUseNoopLogger(t *testing.T) {
	restClient := NewClient("", "", "", WithLogger(nil))
	wsClient := NewPublicWSClient(WithWSLogger(nil))

	assert.NotPanics(t, func() {
		restClient.logger.Debug("request")
		wsClient.logger.Info("connected")
		NewSlogLogger(nil).Warn("ignored")
	})
}

func TestSubscribe_DoesNotRetainSubscriptionWhenWriteFails(t *testing.T) {
	client := NewPublicWSClient()
	arg := models.WSArg{InstType: "SPOT", Topic: "ticker", Symbol: "BTCUSDT"}

	pushes, err := client.Subscribe(context.Background(), arg)
	require.Error(t, err)
	assert.Nil(t, pushes)
	assert.Empty(t, client.subscriptions)
}

func TestPrivateWSConnect_ReturnsLoginErrorImmediately(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()
		_, _, err = conn.ReadMessage()
		require.NoError(t, err)
		require.NoError(t, conn.WriteJSON(models.WSEvent{Event: "error", Code: "30005", Msg: "login failed"}))
	}))
	defer server.Close()

	client := NewPrivateWSClient("key", "secret", "pass", WithWSURL("ws"+strings.TrimPrefix(server.URL, "http")))
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	started := time.Now()
	err := client.Connect(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "code=30005")
	assert.Less(t, time.Since(started), wsLoginTimeout)
}

func TestWSClient_SerializesConcurrentSubscriptions(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	client := NewPublicWSClient(WithWSURL("ws" + strings.TrimPrefix(server.URL, "http")))
	require.NoError(t, client.Connect(context.Background()))
	defer client.Close()

	var wg sync.WaitGroup
	errs := make(chan error, 32)
	for i := 0; i < cap(errs); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := client.Subscribe(context.Background(), models.WSArg{InstType: "SPOT", Topic: "ticker", Symbol: string(rune(i + 1))})
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		assert.NoError(t, err)
	}
}

func TestSubscription_CloseAndSendAreSafe(t *testing.T) {
	push := models.WSPush{Arg: models.WSArg{InstType: "SPOT", Topic: "ticker"}}
	for i := 0; i < 100; i++ {
		sub := &wsSubscription{ch: make(chan models.WSPush, 1)}
		done := make(chan struct{})
		go func() {
			sub.send(push)
			close(done)
		}()
		sub.close()
		<-done
	}
}
