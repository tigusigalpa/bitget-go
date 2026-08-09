package bitget

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
