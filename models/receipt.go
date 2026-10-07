package models

import "time"

// RESTRequestMetadata identifies a request without credentials, headers, host,
// or URL userinfo. Query contains only the SDK's allowlisted market selectors.
type RESTRequestMetadata struct {
	Method string
	Path   string
	Query  map[string]string
}

// RESTReceipt preserves the exact bounded HTTP response body before JSON
// decoding. Its fields are private; accessors return copies of mutable values.
// Complete means the body was fully read within the byte bound, not that the
// response succeeded or that the provider's historical coverage is complete.
// Bytes are the HTTP body exposed by net/http (after automatic decompression),
// not TLS/HTTP framing. A nil receipt means no HTTP response was received.
type RESTReceipt struct {
	request    RESTRequestMetadata
	payload    string
	statusCode int
	receivedAt time.Time
	complete   bool
}

// NewRESTReceipt takes an owned snapshot of payload and request metadata.
// It is primarily used by the SDK transport; constructing a receipt yourself
// does not authenticate its provenance or enforce the transport's byte bound.
func NewRESTReceipt(request RESTRequestMetadata, payload []byte, statusCode int, receivedAt time.Time, complete bool) *RESTReceipt {
	request.Query = copyReceiptQuery(request.Query)
	return &RESTReceipt{request: request, payload: string(payload), statusCode: statusCode, receivedAt: receivedAt, complete: complete}
}

// Request returns a copy of the credential-free request metadata.
func (r *RESTReceipt) Request() RESTRequestMetadata {
	request := r.request
	request.Query = copyReceiptQuery(request.Query)
	return request
}

// Payload returns a fresh copy of the exact received response body.
func (r *RESTReceipt) Payload() []byte { return []byte(r.payload) }

// StatusCode returns the HTTP response status.
func (r *RESTReceipt) StatusCode() int { return r.statusCode }

// ReceivedAt returns local time immediately after the bounded body read,
// before closing the body or decoding JSON. It is not exchange event time.
func (r *RESTReceipt) ReceivedAt() time.Time { return r.receivedAt }

// Complete reports whether the entire HTTP body fit in the receipt bound and
// was read without error. Oversized or interrupted bodies return false.
func (r *RESTReceipt) Complete() bool { return r.complete }

func copyReceiptQuery(query map[string]string) map[string]string {
	copy := make(map[string]string, len(query))
	for key, value := range query {
		copy[key] = value
	}
	return copy
}
