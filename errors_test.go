package bitget

import "testing"

func TestErrorAndMapErrorCode(t *testing.T) {
	err := &Error{Code: "40001", Message: "invalid API key"}
	if got, want := err.Error(), "bitget: api error: code=40001, message=invalid API key"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	tests := map[string]error{
		"":      nil,
		"00000": nil,
		"40001": ErrUnauthorized,
		"40002": ErrInvalidSignature,
		"40004": ErrInvalidTimestamp,
		"40012": ErrPermissionDenied,
		"429":   ErrRateLimited,
		"40018": ErrInvalidParameter,
		"43012": ErrInsufficientFunds,
		"43025": ErrOrderNotFound,
		"50000": ErrInternalServer,
		"other": nil,
	}
	for code, want := range tests {
		if got := MapErrorCode(code); got != want {
			t.Errorf("MapErrorCode(%q) = %v, want %v", code, got, want)
		}
	}
}
