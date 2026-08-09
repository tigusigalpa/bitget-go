package bitget

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSign_KnownVectors verifies the HMAC-SHA256/Base64 signature against
// vectors computed independently via `openssl dgst -sha256 -hmac`, covering
// a GET without query params, a GET with query params, and a POST with a
// JSON body.
//
// Docs: https://www.bitget.com/api-doc/uta/guide
func TestSign_KnownVectors(t *testing.T) {
	c := NewClient("api-key", "test-secret-key", "passphrase")

	tests := []struct {
		name        string
		timestamp   string
		method      string
		requestPath string
		body        string
		want        string
	}{
		{
			name:        "GET without query",
			timestamp:   "1622185200000",
			method:      "GET",
			requestPath: "/api/v3/account/assets",
			body:        "",
			want:        "K74C/9e7Xifxt2o7mhHMrKVlCcS5+9ltdOp8IwDCeEc=",
		},
		{
			name:        "GET with query",
			timestamp:   "1622185200000",
			method:      "GET",
			requestPath: "/api/v3/market/tickers?category=SPOT&symbol=BTCUSDT",
			body:        "",
			want:        "uKM5T0Vj9c0Nt+nn8ALucHdTdnVMPCX4Hk/j1+lt6LI=",
		},
		{
			name:        "POST with JSON body",
			timestamp:   "1622185200000",
			method:      "POST",
			requestPath: "/api/v3/trade/place-order",
			body:        `{"category":"SPOT","symbol":"BTCUSDT","side":"buy"}`,
			want:        "PoEyXzTthV6aMBFloYTi5i1DHbf1m4nwXER8JR5W0AE=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.sign(tt.timestamp, tt.method, tt.requestPath, tt.body)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildQueryString_SortsAscendingByKey(t *testing.T) {
	got := buildQueryString(map[string]string{"symbol": "BTCUSDT", "category": "SPOT"})
	assert.Equal(t, "category=SPOT&symbol=BTCUSDT", got)
}

func TestBuildQueryString_OmitsEmptyValues(t *testing.T) {
	got := buildQueryString(map[string]string{"symbol": "BTCUSDT", "cursor": ""})
	assert.Equal(t, "symbol=BTCUSDT", got)
}

func TestDo_SetsAuthHeaders(t *testing.T) {
	var gotHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"00000","msg":"success","requestTime":1,"data":{}}`))
	}))
	defer server.Close()

	c := NewClient("api-key", "secret-key", "passphrase", WithBaseURL(server.URL), WithDemoTrading())
	err := c.do(context.Background(), http.MethodGet, "/api/v3/account/assets", nil, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "api-key", gotHeaders.Get("ACCESS-KEY"))
	assert.Equal(t, "passphrase", gotHeaders.Get("ACCESS-PASSPHRASE"))
	assert.NotEmpty(t, gotHeaders.Get("ACCESS-SIGN"))
	assert.NotEmpty(t, gotHeaders.Get("ACCESS-TIMESTAMP"))
	assert.Equal(t, "1", gotHeaders.Get("paptrading"))
}

func TestDoPublic_DoesNotSetAuthHeaders(t *testing.T) {
	var gotHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"00000","msg":"success","requestTime":1,"data":[]}`))
	}))
	defer server.Close()

	c := NewClient("", "", "", WithBaseURL(server.URL))
	err := c.doPublic(context.Background(), http.MethodGet, "/api/v3/market/tickers", map[string]string{"category": "SPOT"}, nil)
	require.NoError(t, err)

	assert.Empty(t, gotHeaders.Get("ACCESS-KEY"))
	assert.Empty(t, gotHeaders.Get("ACCESS-SIGN"))
}

func TestDo_DecodesResultData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"00000","msg":"success","requestTime":1,"data":{"uid":"12345"}}`))
	}))
	defer server.Close()

	c := NewClient("api-key", "secret-key", "passphrase", WithBaseURL(server.URL))
	var result struct {
		UID string `json:"uid"`
	}
	err := c.do(context.Background(), http.MethodGet, "/api/v3/account/settings", nil, nil, &result)
	require.NoError(t, err)
	assert.Equal(t, "12345", result.UID)
}

func TestDo_MapsSentinelError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"40001","msg":"invalid API key","requestTime":1,"data":null}`))
	}))
	defer server.Close()

	c := NewClient("bad-key", "secret-key", "passphrase", WithBaseURL(server.URL))
	err := c.do(context.Background(), http.MethodGet, "/api/v3/account/assets", nil, nil, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))

	var bitgetErr *BitgetError
	require.True(t, errors.As(err, &bitgetErr))
	assert.Equal(t, "40001", bitgetErr.Code)
	assert.Equal(t, "invalid API key", bitgetErr.Message)
}

func TestDo_MapsRateLimitOn429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"code":"429","msg":"too many requests"}`))
	}))
	defer server.Close()

	c := NewClient("api-key", "secret-key", "passphrase", WithBaseURL(server.URL))
	err := c.do(context.Background(), http.MethodGet, "/api/v3/account/assets", nil, nil, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrRateLimited))
}

func TestDo_ReturnsErrorForUnexpectedHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":"00000","msg":"success","requestTime":1,"data":{}}`))
	}))
	defer server.Close()

	c := NewClient("", "", "", WithBaseURL(server.URL))
	err := c.doPublic(context.Background(), http.MethodGet, "/api/v3/market/tickers", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected HTTP status 500")
}

func TestDo_RejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", maxResponseBodySize+1)))
	}))
	defer server.Close()

	c := NewClient("", "", "", WithBaseURL(server.URL))
	err := c.doPublic(context.Background(), http.MethodGet, "/api/v3/market/tickers", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "response body exceeds")
}
