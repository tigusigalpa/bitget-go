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

	"github.com/tigusigalpa/bitget-go/models"
)

const maxResponseBodySize = 10 << 20

// MaxRESTReceiptBytes is the maximum number of response-body bytes retained in
// a RESTReceipt. An oversized response fails with an incomplete bounded receipt.
const MaxRESTReceiptBytes = maxResponseBodySize

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
	_, err := c.requestWithReceipt(ctx, method, path, query, body, result, signed, false)
	return err
}

func (c *Client) doPublicWithReceipt(ctx context.Context, method, path string, query map[string]string, result interface{}) (*models.RESTReceipt, error) {
	return c.requestWithReceipt(ctx, method, path, query, nil, result, false, true)
}

func (c *Client) requestWithReceipt(ctx context.Context, method, path string, query map[string]string, body interface{}, result interface{}, signed, capture bool) (*models.RESTReceipt, error) {
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
			return nil, fmt.Errorf("bitget: marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), fullURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("bitget: build request: %w", err)
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
		return nil, fmt.Errorf("bitget: do request: %w", err)
	}
	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize+1))
	receivedAt := time.Now()
	var receipt *models.RESTReceipt
	if capture {
		payload := respBody
		if len(payload) > MaxRESTReceiptBytes {
			payload = payload[:MaxRESTReceiptBytes]
		}
		receipt = models.NewRESTReceipt(safeRESTRequestMetadata(req, query), payload, resp.StatusCode, receivedAt,
			readErr == nil && len(respBody) <= maxResponseBodySize)
	}
	closeErr := resp.Body.Close()
	if readErr != nil {
		if closeErr != nil {
			return receipt, fmt.Errorf("bitget: read response body: %w; close response body: %w", readErr, closeErr)
		}
		return receipt, fmt.Errorf("bitget: read response body: %w", readErr)
	}
	if closeErr != nil {
		return receipt, fmt.Errorf("bitget: close response body: %w", closeErr)
	}
	if len(respBody) > maxResponseBodySize {
		return receipt, fmt.Errorf("bitget: response body exceeds %d bytes", maxResponseBodySize)
	}

	var env envelope
	decodeErr := json.Unmarshal(respBody, &env)
	if resp.StatusCode == http.StatusTooManyRequests {
		if decodeErr == nil && env.Code != "" && env.Code != "00000" {
			return receipt, fmt.Errorf("%w: %w", ErrRateLimited, &Error{Code: env.Code, Message: env.Msg, Raw: respBody})
		}
		return receipt, fmt.Errorf("%w: HTTP 429", ErrRateLimited)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if decodeErr != nil || env.Code == "" || env.Code == "00000" {
			return receipt, fmt.Errorf("bitget: unexpected HTTP status %s", resp.Status)
		}
	}
	if decodeErr != nil {
		return receipt, fmt.Errorf("bitget: decode response envelope (status %d): %w", resp.StatusCode, decodeErr)
	}
	if capture && env.Code == "" {
		return receipt, fmt.Errorf("bitget: response envelope is missing code")
	}

	if env.Code != "" && env.Code != "00000" {
		bitgetErr := &Error{Code: env.Code, Message: env.Msg, Raw: respBody}
		if sentinel := MapErrorCode(env.Code); sentinel != nil {
			return receipt, fmt.Errorf("%w: %w", sentinel, bitgetErr)
		}
		return receipt, bitgetErr
	}

	if result != nil && len(env.Data) > 0 {
		if capture && bytes.Equal(bytes.TrimSpace(env.Data), []byte("null")) {
			return receipt, fmt.Errorf("bitget: response data is null")
		}
		if err := json.Unmarshal(env.Data, result); err != nil {
			return receipt, fmt.Errorf("bitget: decode response data: %w", err)
		}
	} else if capture && result != nil {
		return receipt, fmt.Errorf("bitget: response envelope is missing data")
	}

	return receipt, nil
}

func safeRESTRequestMetadata(req *http.Request, query map[string]string) models.RESTRequestMetadata {
	safeQuery := make(map[string]string)
	for _, key := range []string{"category", "symbol", "interval", "type", "startTime", "endTime", "limit", "cursor"} {
		if value := query[key]; value != "" {
			safeQuery[key] = value
		}
	}
	return models.RESTRequestMetadata{Method: req.Method, Path: req.URL.EscapedPath(), Query: safeQuery}
}
