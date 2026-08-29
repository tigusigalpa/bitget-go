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

func TestWSClientLocalErrorBranches(t *testing.T) {
	arg := models.WSArg{InstType: "SPOT", Topic: "ticker", Symbol: "BTCUSDT"}

	closed := NewPublicWSClient()
	require.NoError(t, closed.Close())
	assert.ErrorContains(t, closed.Connect(context.Background()), "closed")

	client := NewPublicWSClient(WithWSAutoReconnect(false))
	assert.False(t, client.autoReconnect)
	assert.ErrorContains(t, client.writeJSON(struct{}{}), "not connected")
	assert.ErrorContains(t, client.writeMessage(websocket.TextMessage, []byte("ping")), "not connected")

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.Subscribe(canceled, arg)
	require.ErrorIs(t, err, context.Canceled)

	sub := &wsSubscription{arg: arg, ch: make(chan models.WSPush, 1)}
	client.subscriptions[wsSubKey(arg)] = sub
	pushes, err := client.Subscribe(context.Background(), arg)
	require.NoError(t, err)
	assert.Equal(t, (<-chan models.WSPush)(sub.ch), pushes)
	assert.NoError(t, client.Unsubscribe(models.WSArg{InstType: "SPOT", Topic: "trades"}))
	assert.ErrorContains(t, client.Unsubscribe(arg), "not connected")
	_, open := <-sub.ch
	assert.False(t, open)

	client.closeCurrentConnection()
	client.startReconnect()
	assert.False(t, client.reconnecting)
	assert.NoError(t, client.Close())
	client.pingPump()
	client.readPump()
	assert.NoError(t, client.Close())
}

func TestWSClientHandlesProtocolMessages(t *testing.T) {
	client := NewPublicWSClient(WithWSAutoReconnect(false))
	loginWait := make(chan error, 1)
	client.loginWait = loginWait
	client.handleMessage([]byte(`{"event":"login","code":"0"}`))
	require.NoError(t, <-loginWait)

	client.loginWait = loginWait
	client.handleMessage([]byte(`{"event":"login","code":"30005","msg":"bad credentials"}`))
	assert.ErrorContains(t, <-loginWait, "bad credentials")

	client.loginWait = loginWait
	client.handleMessage([]byte(`{"event":"error","code":"30001","msg":"bad request"}`))
	assert.ErrorContains(t, <-loginWait, "bad request")
	client.loginWait = nil

	arg := models.WSArg{InstType: "SPOT", Topic: "ticker", Symbol: "BTCUSDT"}
	sub := &wsSubscription{arg: arg, ch: make(chan models.WSPush, 1)}
	client.subscriptions[wsSubKey(arg)] = sub
	client.handleMessage([]byte(`not json`))
	client.handleMessage([]byte(`{"arg":{"instType":"SPOT","topic":"ticker","symbol":"ETHUSDT"},"data":[]}`))
	client.handleMessage([]byte(`{"arg":{"instType":"SPOT","topic":"ticker","symbol":"BTCUSDT"},"action":"snapshot","data":[{"last":"1"}]}`))
	push := <-sub.ch
	assert.Equal(t, "snapshot", push.Action)

	sub.ch <- models.WSPush{}
	client.handleMessage([]byte(`{"arg":{"instType":"SPOT","topic":"ticker","symbol":"BTCUSDT"},"data":[]}`))
	<-sub.ch
	assert.True(t, sub.send(models.WSPush{}))
	sub.close()
	assert.False(t, sub.send(models.WSPush{}))
}

func TestWSClientReconnectHelpers(t *testing.T) {
	client := NewPublicWSClient()
	client.resubscribeAll()
	arg := models.WSArg{InstType: "SPOT", Topic: "ticker", Symbol: "BTCUSDT"}
	client.subscriptions[wsSubKey(arg)] = &wsSubscription{arg: arg, ch: make(chan models.WSPush, 1)}
	client.resubscribeAll()

	client.reconnecting = true
	close(client.done)
	client.reconnectLoop()
	assert.False(t, client.reconnecting)
}

func TestPrivateWSClientConnectsAndReceivesPushes(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var login models.WSLoginRequest
		require.NoError(t, conn.ReadJSON(&login))
		require.Equal(t, "key", login.Args[0].APIKey)
		require.NoError(t, conn.WriteJSON(models.WSEvent{Event: "login", Code: "0"}))

		var subscribe models.WSSubscribeRequest
		require.NoError(t, conn.ReadJSON(&subscribe))
		require.Equal(t, "subscribe", subscribe.Op)
		require.NoError(t, conn.WriteJSON(models.WSPush{
			Arg:    subscribe.Args[0],
			Action: "snapshot",
			Data:   []byte(`[{"price":"1"}]`),
		}))

		var unsubscribe models.WSSubscribeRequest
		require.NoError(t, conn.ReadJSON(&unsubscribe))
		require.Equal(t, "unsubscribe", unsubscribe.Op)
	}))
	defer server.Close()

	client := NewPrivateWSClient("key", "secret", "pass", WithWSURL("ws"+strings.TrimPrefix(server.URL, "http")), WithWSAutoReconnect(false))
	defer client.Close()
	require.NoError(t, client.Connect(context.Background()))
	assert.ErrorContains(t, client.Connect(context.Background()), "already connected")

	arg := models.WSArg{InstType: "UTA", Topic: "fast-fill", Symbol: "default"}
	pushes, err := client.Subscribe(context.Background(), arg)
	require.NoError(t, err)
	select {
	case push := <-pushes:
		assert.Equal(t, "snapshot", push.Action)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for push")
	}
	require.NoError(t, client.Unsubscribe(arg))
}
