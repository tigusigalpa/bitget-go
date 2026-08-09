// Package bitget is an idiomatic Go SDK for the Bitget Unified Trading
// Account (UTA) API v3.
//
// Docs: https://www.bitget.com/api-doc/uta/intro
package bitget

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Client is the low-level authenticated HTTP transport shared by every
// service under RestClient. Most callers should construct a *RestClient via
// NewRestClient instead of using Client directly.
type Client struct {
	apiKey      string
	secretKey   string
	passphrase  string
	baseURL     string
	demoTrading bool
	locale      string

	httpClient *http.Client
	timeout    time.Duration
	logger     Logger
}

// NewClient creates a low-level Client. apiKey/secretKey/passphrase are the
// credentials generated for an API key in the Bitget web console
// (https://www.bitget.com/api-doc/uta/guide). Most callers want
// NewRestClient instead, which additionally wires up all API services.
func NewClient(apiKey, secretKey, passphrase string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		secretKey:  secretKey,
		passphrase: passphrase,
		baseURL:    DefaultBaseURL,
		locale:     "en-US",
		timeout:    DefaultTimeout,
		logger:     noopLogger{},
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
	}
	return c
}

// envelope is Bitget's standard response wrapper.
//
// Docs: https://www.bitget.com/api-doc/uta/guide
type envelope struct {
	Code        string          `json:"code"`
	Msg         string          `json:"msg"`
	RequestTime int64           `json:"requestTime"`
	Data        json.RawMessage `json:"data"`
}

// sign implements Bitget's HMAC-SHA256 + Base64 request signature.
//
// Pre-hash string: timestamp + method.toUpperCase() + requestPath [+ "?" + queryString] + body
//
// Docs: https://www.bitget.com/api-doc/uta/guide
func (c *Client) sign(timestamp, method, requestPath, body string) string {
	prehash := timestamp + strings.ToUpper(method) + requestPath + body
	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(prehash))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// buildQueryString sorts params ascending by key (required by Bitget's
// signature algorithm) and url-encodes them.
func buildQueryString(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	values := url.Values{}
	for _, k := range keys {
		values.Set(k, params[k])
	}
	return values.Encode()
}

func timestampMillis() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 10)
}

// doPublic issues an unauthenticated request against a public endpoint
// (e.g. market data). No signature headers are sent.
func (c *Client) doPublic(ctx context.Context, method, path string, query map[string]string, result interface{}) error {
	return c.request(ctx, method, path, query, nil, result, false)
}

// do issues an authenticated request, signing it with the configured API
// credentials.
func (c *Client) do(ctx context.Context, method, path string, query map[string]string, body interface{}, result interface{}) error {
	return c.request(ctx, method, path, query, body, result, true)
}

func (c *Client) request(ctx context.Context, method, path string, query map[string]string, body interface{}, result interface{}, signed bool) error {
	queryString := buildQueryString(query)
	requestPath := path
	fullURL := c.baseURL + path
	if queryString != "" {
		fullURL += "?" + queryString
	}

	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("bitget: marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), fullURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("bitget: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("locale", c.locale)

	if signed {
		timestamp := timestampMillis()
		signPath := requestPath
		if queryString != "" {
			signPath += "?" + queryString
		}
		signature := c.sign(timestamp, method, signPath, string(bodyBytes))
		req.Header.Set("ACCESS-KEY", c.apiKey)
		req.Header.Set("ACCESS-SIGN", signature)
		req.Header.Set("ACCESS-TIMESTAMP", timestamp)
		req.Header.Set("ACCESS-PASSPHRASE", c.passphrase)
		if c.demoTrading {
			req.Header.Set("paptrading", "1")
		}
	}

	c.logger.Debug("bitget: request", "method", method, "path", requestPath)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("bitget: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("bitget: read response body: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("%w: HTTP 429", ErrRateLimited)
	}

	var env envelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return fmt.Errorf("bitget: decode response envelope (status %d): %w", resp.StatusCode, err)
	}

	if env.Code != "" && env.Code != "00000" {
		bitgetErr := &BitgetError{Code: env.Code, Message: env.Msg, Raw: respBody}
		if sentinel := MapErrorCode(env.Code); sentinel != nil {
			return fmt.Errorf("%w: %w", sentinel, bitgetErr)
		}
		return bitgetErr
	}

	if result != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, result); err != nil {
			return fmt.Errorf("bitget: decode response data: %w", err)
		}
	}

	return nil
}
