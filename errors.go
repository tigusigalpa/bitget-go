package bitget

import (
	"errors"
	"fmt"
)

// BitgetError represents a structured error returned by the Bitget UTA API,
// preserving the exact code/message pair sent back in the response envelope.
//
// Docs: https://www.bitget.com/api-doc/uta/guide
type BitgetError struct {
	Code    string
	Message string
	Raw     []byte
}

func (e *BitgetError) Error() string {
	return fmt.Sprintf("bitget: api error: code=%s, message=%s", e.Code, e.Message)
}

// Sentinel errors that callers can match with errors.Is, wrapped alongside
// the detailed *BitgetError (retrievable via errors.As) on every failed call.
var (
	ErrUnauthorized      = errors.New("bitget: unauthorized: invalid API credentials")
	ErrInvalidSignature  = errors.New("bitget: invalid signature")
	ErrInvalidTimestamp  = errors.New("bitget: invalid or expired timestamp")
	ErrPermissionDenied  = errors.New("bitget: permission denied for this API key")
	ErrRateLimited       = errors.New("bitget: rate limit exceeded")
	ErrInvalidParameter  = errors.New("bitget: invalid request parameter")
	ErrInsufficientFunds = errors.New("bitget: insufficient balance")
	ErrOrderNotFound     = errors.New("bitget: order not found")
	ErrInternalServer    = errors.New("bitget: internal server error")
)

// MapErrorCode maps a Bitget API response "code" to a sentinel error so
// callers can use errors.Is without parsing raw codes themselves. Unknown
// codes return nil (caller should fall back to the raw *BitgetError).
func MapErrorCode(code string) error {
	switch code {
	case "":
		return nil
	case "00000":
		return nil
	case "40001", "40006", "40009", "40037":
		return ErrUnauthorized
	case "40002", "40003":
		return ErrInvalidSignature
	case "40004", "40005":
		return ErrInvalidTimestamp
	case "40012", "40014", "40017":
		return ErrPermissionDenied
	case "429", "40429", "30007":
		return ErrRateLimited
	case "40018", "40019", "40020", "40021", "40022", "40023", "22001":
		return ErrInvalidParameter
	case "43012", "45006":
		return ErrInsufficientFunds
	case "43025", "43001":
		return ErrOrderNotFound
	case "50000", "50001", "50002":
		return ErrInternalServer
	default:
		return nil
	}
}
