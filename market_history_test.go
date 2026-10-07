package bitget

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tigusigalpa/bitget-go/models"
	"github.com/tigusigalpa/bitget-go/rest/market"
)

func historyFixture(t *testing.T, name string) []byte {
	t.Helper()
	payload, err := os.ReadFile("testdata/market-history/" + name)
	require.NoError(t, err)
	return payload
}

func TestHistoryCandlesBoundariesAndRounding(t *testing.T) {
	for _, scenario := range []struct {
		name, fixture, end string
		timestamps         []string
	}{
		{"exclusive", "candles-exclusive.json", "1609459320000", []string{"1609459260000"}},
		{"inclusive", "candles-inclusive.json", "1609459320000", []string{"1609459200000", "1609459260000", "1609459320000"}},
		{"one millisecond rounding", "candles-rounded-extra.json", "1609459320001", []string{"1609459140000", "1609459200000", "1609459260000"}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			payload := historyFixture(t, scenario.fixture)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/v3/market/history-candles", r.URL.Path)
				assert.Equal(t, "1609459200000", r.URL.Query().Get("startTime"))
				assert.Equal(t, scenario.end, r.URL.Query().Get("endTime"))
				assert.Equal(t, "2", r.URL.Query().Get("limit"))
				assert.Equal(t, "mark", r.URL.Query().Get("type"))
				assert.Equal(t, "1m", r.URL.Query().Get("interval"))
				assert.Equal(t, "USDT-FUTURES", r.URL.Query().Get("category"))
				assert.Equal(t, "BTCUSDT", r.URL.Query().Get("symbol"))
				assert.Empty(t, r.Header.Get("ACCESS-KEY"))
				assert.Empty(t, r.Header.Get("ACCESS-SIGN"))
				_, _ = w.Write(payload)
			}))
			defer server.Close()
			client := NewRestClient("unused-key", "unused-secret", "unused-pass", WithBaseURL(server.URL))
			before := time.Now()
			candles, receipt, err := client.Market.GetHistoryCandlesWithReceipt(context.Background(), market.GetHistoryCandlesOptions{
				Category: models.CategoryUSDTFutures, Symbol: "BTCUSDT", Interval: "1m",
				CandleType: "mark", StartTime: "1609459200000", EndTime: scenario.end, Limit: "2",
			})
			require.NoError(t, err)
			require.NotNil(t, receipt)
			assert.Equal(t, payload, receipt.Payload())
			assert.True(t, receipt.Complete())
			assert.Equal(t, http.StatusOK, receipt.StatusCode())
			assert.False(t, receipt.ReceivedAt().Before(before))
			assert.False(t, receipt.ReceivedAt().After(time.Now()))
			require.Len(t, candles, len(scenario.timestamps))
			for i, timestamp := range scenario.timestamps {
				assert.Equal(t, timestamp, candles[i][0])
			}
			if scenario.name == "exclusive" {
				assert.Equal(t, "29000.00000001", candles[0][1])
				assert.Equal(t, "0.0000000100", candles[0][5])
			}
		})
	}
}

func TestLiquidationsPartialPagination(t *testing.T) {
	page1 := historyFixture(t, "liquidations-page-1.json")
	page2 := historyFixture(t, "liquidations-page-2.json")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		assert.Equal(t, "/api/v3/market/liquidations", r.URL.Path)
		assert.Equal(t, "COIN-FUTURES", r.URL.Query().Get("category"))
		assert.Equal(t, "BTCUSD", r.URL.Query().Get("symbol"))
		assert.Equal(t, "100", r.URL.Query().Get("limit"))
		if r.URL.Query().Get("cursor") == "" {
			_, _ = w.Write(page1)
		} else {
			assert.Equal(t, "opaque/+page=2", r.URL.Query().Get("cursor"))
			_, _ = w.Write(page2)
		}
	}))
	defer server.Close()
	client := NewRestClient("", "", "", WithBaseURL(server.URL))
	opts := market.GetLiquidationsOptions{Category: models.CategoryCoinFutures, Symbol: "BTCUSD", Limit: "100"}
	first, receipt, err := client.Market.GetLiquidationsWithReceipt(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, first.List, 2)
	assert.Equal(t, first.List[0], first.List[1], "indistinguishable observations must not be deduplicated")
	assert.Equal(t, "0.0000000100", first.List[0].Amount)
	assert.Equal(t, "1609459260000", first.List[0].Ts)
	assert.Equal(t, page1, receipt.Payload())
	opts.Cursor = first.Cursor
	second, secondReceipt, err := client.Market.GetLiquidationsWithReceipt(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, second.List, 1)
	assert.Empty(t, second.Cursor)
	assert.Equal(t, "2.0000000000", second.List[0].Amount)
	assert.Equal(t, page2, secondReceipt.Payload())
	assert.Equal(t, opts.Cursor, secondReceipt.Request().Query["cursor"])
	assert.EqualValues(t, 2, requests.Load(), "methods fetch exactly one page per call")
}

func TestReceiptErrorsAndStrictHistoryData(t *testing.T) {
	for _, scenario := range []struct {
		name, payload string
		status        int
		want          string
	}{
		{"rate limit", string(historyFixture(t, "rate-limit.json")), 429, "rate limit"},
		{"provider error", `{"code":"40018","msg":"invalid selector","data":null}`, 400, "invalid selector"},
		{"unknown provider error", `{"code":"99999","msg":"rejected","data":null}`, 200, "rejected"},
		{"http failure", `upstream unavailable`, 503, "unexpected HTTP status"},
		{"truncated json", string(historyFixture(t, "truncated.json")), 200, "decode response envelope"},
		{"missing code", `{"data":[]}`, 200, "missing code"},
		{"missing data", `{"code":"00000"}`, 200, "missing data"},
		{"null data", `{"code":"00000","data":null}`, 200, "data is null"},
		{"short row", `{"code":"00000","data":[["1","2"]]}`, 200, "expected 7"},
		{"extra field", `{"code":"00000","data":[["1","2","3","4","5","6","7","8"]]}`, 200, "expected 7"},
		{"number instead of string", `{"code":"00000","data":[[1,"2","3","4","5","6","7"]]}`, 200, "decode response data"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(scenario.status)
				_, _ = w.Write([]byte(scenario.payload))
			}))
			defer server.Close()
			client := NewRestClient("", "", "", WithBaseURL(server.URL))
			data, receipt, err := client.Market.GetHistoryCandlesWithReceipt(context.Background(), market.GetHistoryCandlesOptions{Category: "SPOT", Symbol: "BTCUSDT", Interval: "1m"})
			require.ErrorContains(t, err, scenario.want)
			assert.Nil(t, data)
			require.NotNil(t, receipt)
			assert.Equal(t, scenario.payload, string(receipt.Payload()))
			assert.Equal(t, scenario.status, receipt.StatusCode())
			assert.True(t, receipt.Complete(), "complete HTTP bytes do not imply successful JSON or market coverage")
			if scenario.status == 429 {
				assert.ErrorIs(t, err, ErrRateLimited)
			}
			if scenario.name == "provider error" {
				assert.ErrorIs(t, err, ErrInvalidParameter)
			}
			if scenario.name == "rate limit" || scenario.name == "provider error" || scenario.name == "unknown provider error" {
				var providerErr *Error
				require.ErrorAs(t, err, &providerErr)
				assert.Equal(t, scenario.payload, string(providerErr.Raw))
				providerErr.Raw[0] = 'x'
				assert.Equal(t, scenario.payload, string(receipt.Payload()), "error payload does not alias receipt")
			}
		})
	}
}

type historyRoundTripper func(*http.Request) (*http.Response, error)

func (f historyRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestReceiptSizeBoundAndInjectedTransport(t *testing.T) {
	for _, oversized := range []bool{false, true} {
		t.Run(map[bool]string{false: "exact limit", true: "oversized"}[oversized], func(t *testing.T) {
			prefix := `{"code":"00000","data":[]}`
			length := MaxRESTReceiptBytes
			if oversized {
				length++
			}
			payload := prefix + strings.Repeat(" ", length-len(prefix))
			httpClient := &http.Client{Transport: historyRoundTripper(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, "/api/v3/market/history-candles", r.URL.Path)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(payload)), Header: make(http.Header)}, nil
			})}
			client := NewRestClient("", "", "", WithHTTPClient(httpClient))
			data, receipt, err := client.Market.GetHistoryCandlesWithReceipt(context.Background(), market.GetHistoryCandlesOptions{Category: "SPOT", Symbol: "BTCUSDT", Interval: "1m", Limit: "100"})
			require.NotNil(t, receipt)
			assert.Len(t, receipt.Payload(), MaxRESTReceiptBytes)
			assert.Equal(t, payload[:MaxRESTReceiptBytes], string(receipt.Payload()))
			assert.Equal(t, !oversized, receipt.Complete())
			if oversized {
				require.ErrorContains(t, err, "response body exceeds")
				assert.Nil(t, data)
			} else {
				require.NoError(t, err)
				assert.Empty(t, data)
			}
			assert.Same(t, httpClient, client.httpClient)
		})
	}
}

type historyErrorBody struct {
	reader   io.Reader
	readErr  error
	closeErr error
	closed   bool
}

func (b *historyErrorBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	if err == io.EOF && b.readErr != nil {
		return n, b.readErr
	}
	return n, err
}

func (b *historyErrorBody) Close() error { b.closed = true; return b.closeErr }

func TestReceiptInterruptedReadAndCloseFailure(t *testing.T) {
	readFailure, closeFailure := errors.New("body interrupted"), errors.New("close failed")
	for _, readErr := range []error{nil, readFailure} {
		body := &historyErrorBody{reader: strings.NewReader(`{"code":"00000","data":[]}`), readErr: readErr, closeErr: closeFailure}
		client := NewRestClient("", "", "", WithHTTPClient(&http.Client{Transport: historyRoundTripper(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: body, Header: make(http.Header)}, nil
		})}))
		data, receipt, err := client.Market.GetHistoryCandlesWithReceipt(context.Background(), market.GetHistoryCandlesOptions{Category: "SPOT", Symbol: "BTCUSDT", Interval: "1m"})
		require.ErrorIs(t, err, closeFailure)
		if readErr != nil {
			assert.ErrorIs(t, err, readFailure)
		}
		assert.Nil(t, data)
		assert.True(t, body.closed)
		require.NotNil(t, receipt)
		assert.Equal(t, readErr == nil, receipt.Complete())
		assert.Equal(t, `{"code":"00000","data":[]}`, string(receipt.Payload()))
	}
}

type historyCancelBody struct {
	ctx     context.Context
	started chan struct{}
	first   bool
}

func (b *historyCancelBody) Read(p []byte) (int, error) {
	if !b.first {
		b.first = true
		close(b.started)
		return copy(p, `{"code":`), nil
	}
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}

func (*historyCancelBody) Close() error { return nil }

func TestHistoryCancellationDuringRead(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	started := make(chan struct{})
	client := NewRestClient("", "", "", WithHTTPClient(&http.Client{Transport: historyRoundTripper(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: &historyCancelBody{ctx: r.Context(), started: started}, Header: make(http.Header)}, nil
	})}))
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		data, receipt, err := client.Market.GetHistoryCandlesWithReceipt(ctx, market.GetHistoryCandlesOptions{Category: "SPOT", Symbol: "BTCUSDT", Interval: "1m"})
		assert.ErrorIs(t, err, context.Canceled)
		assert.Nil(t, data)
		if assert.NotNil(t, receipt) {
			assert.False(t, receipt.Complete())
			assert.Equal(t, `{"code":`, string(receipt.Payload()))
		}
	}()
	select {
	case <-started:
		cancel()
	case <-ctx.Done():
		t.Fatal("body read did not start")
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("request did not stop after cancellation")
	}
}

func TestReceiptTruncatedHTTPAndTransportFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "1000")
		_, _ = w.Write([]byte(`{"code":`))
	}))
	defer server.Close()
	client := NewRestClient("", "", "", WithBaseURL(server.URL))
	opts := market.GetHistoryCandlesOptions{Category: "SPOT", Symbol: "BTCUSDT", Interval: "1m"}
	data, receipt, err := client.Market.GetHistoryCandlesWithReceipt(context.Background(), opts)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.Nil(t, data)
	require.NotNil(t, receipt)
	assert.False(t, receipt.Complete())
	assert.Equal(t, `{"code":`, string(receipt.Payload()))

	failure := errors.New("dial failed")
	client = NewRestClient("", "", "", WithHTTPClient(&http.Client{Transport: historyRoundTripper(func(*http.Request) (*http.Response, error) {
		return nil, failure
	})}))
	data, receipt, err = client.Market.GetHistoryCandlesWithReceipt(context.Background(), opts)
	assert.ErrorIs(t, err, failure)
	assert.Nil(t, data)
	assert.Nil(t, receipt)
}

func TestSafeReceiptMetadata(t *testing.T) {
	client := NewClient("api-key", "secret", "passphrase", WithBaseURL("https://user:password@example.test"), WithHTTPClient(&http.Client{Transport: historyRoundTripper(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "api-key", r.Header.Get("ACCESS-KEY"))
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":"00000","data":[]}`)), Header: make(http.Header)}, nil
	})}))
	query := map[string]string{"category": "SPOT", "symbol": "BTCUSDT", "token": "secret-token", "ACCESS-KEY": "key", "cursor": "opaque/+"}
	var data []models.Candle
	receipt, err := client.requestWithReceipt(context.Background(), "get", "/api/v3/market/history-candles", query, nil, &data, true, true)
	require.NoError(t, err)
	metadata := receipt.Request()
	assert.Equal(t, "GET", metadata.Method)
	assert.Equal(t, "/api/v3/market/history-candles", metadata.Path)
	assert.Equal(t, map[string]string{"category": "SPOT", "symbol": "BTCUSDT", "cursor": "opaque/+"}, metadata.Query)
	query["category"] = "mutated"
	metadata.Query["symbol"] = "mutated"
	assert.Equal(t, "SPOT", receipt.Request().Query["category"])
	assert.Equal(t, "BTCUSDT", receipt.Request().Query["symbol"])
}

func TestExistingMarketMethodsStillUseOriginalRoutes(t *testing.T) {
	recentFills := historyFixture(t, "recent-fills.json")
	var routes []string
	var routesMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routesMu.Lock()
		routes = append(routes, r.URL.Path)
		routesMu.Unlock()
		switch r.URL.Path {
		case "/api/v3/market/candles":
			assert.Equal(t, "1000", r.URL.Query().Get("limit"))
			_, _ = w.Write(historyFixture(t, "candles-inclusive.json"))
		case "/api/v3/market/history-fund-rate":
			assert.Equal(t, "2", r.URL.Query().Get("cursor"))
			_, _ = w.Write([]byte(`{"code":"00000","data":{"resultList":[{"symbol":"BTCUSDT","fundingRate":"0.00001000","fundingRateTimestamp":"1609459200000"}]}}`))
		case "/api/v3/market/fills":
			assert.Empty(t, r.URL.Query().Get("cursor"))
			_, _ = w.Write(recentFills)
		}
	}))
	defer server.Close()
	client := NewRestClient("", "", "", WithBaseURL(server.URL))
	candles, err := client.Market.GetCandles(context.Background(), market.GetCandlesOptions{Category: "SPOT", Symbol: "BTCUSDT", Interval: "1m", Limit: "1000"})
	require.NoError(t, err)
	require.Len(t, candles, 3)
	funding, err := client.Market.GetFundingRateHistory(context.Background(), market.GetFundingRateHistoryOptions{Category: "USDT-FUTURES", Symbol: "BTCUSDT", Cursor: "2", Limit: "100"})
	require.NoError(t, err)
	require.Len(t, funding.ResultList, 1)
	assert.Equal(t, "0.00001000", funding.ResultList[0].FundingRate)
	fills, err := client.Market.GetPublicFills(context.Background(), "COIN-FUTURES", "BTCUSD", "100")
	require.NoError(t, err)
	require.Len(t, fills, 1)
	assert.Equal(t, "9007199254740993001", fills[0].ExecID)
	assert.Equal(t, "9007199254740993002", fills[0].ExecLinkID)
	assert.Equal(t, "1609459260000", fills[0].Ts)
	assert.Equal(t, "0.0000000100", fills[0].Size)
	routesMu.Lock()
	defer routesMu.Unlock()
	assert.Equal(t, []string{"/api/v3/market/candles", "/api/v3/market/history-fund-rate", "/api/v3/market/fills"}, routes)
}
