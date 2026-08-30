package errors

import (
	"errors"
	"net"
	"net/url"
	"syscall"
	"testing"

	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

func TestExitCode_StatusFallbacksWhenCodeUnrecognized(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"401 unrecognized code", &wenmar.APIError{Code: "weird_proxy", StatusCode: 401}, 2},
		{"404 unrecognized code", &wenmar.APIError{Code: "", StatusCode: 404}, 3},
		{"422 unrecognized code", &wenmar.APIError{Code: "", StatusCode: 422}, 4},
		{"429 unrecognized code", &wenmar.APIError{Code: "slow_down", StatusCode: 429}, 5},
		{"502 unrecognized code", &wenmar.APIError{Code: "", StatusCode: 502}, 6},
		{"409 unrecognized code", &wenmar.APIError{Code: "dup", StatusCode: 409}, 7},
		{"403 unrecognized code", &wenmar.APIError{Code: "nope", StatusCode: 403}, 8},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCode(tc.err); got != tc.want {
				t.Errorf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

func TestExitCode_ConnectionRefusedIsOffline(t *testing.T) {
	// *url.Error wrapping ECONNREFUSED must map to 10, not 1.
	err := &url.Error{
		Op:  "Get",
		URL: "http://127.0.0.1:1/customers",
		Err: &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: syscall.ECONNREFUSED,
		},
	}
	if got := ExitCode(err); got != ExitOffline {
		t.Errorf("ExitCode(ECONNREFUSED) = %d, want %d", got, ExitOffline)
	}
}

func TestExitCode_Success(t *testing.T) {
	code := ExitCode(nil)
	if code != 0 {
		t.Errorf("expected 0 for nil error, got %d", code)
	}
}

func TestExitCode_AuthFailure(t *testing.T) {
	err := &wenmar.APIError{Code: "unauthorized", StatusCode: 401}
	code := ExitCode(err)
	if code != 2 {
		t.Errorf("expected 2 for auth failure, got %d", code)
	}
}

func TestExitCode_NotFound(t *testing.T) {
	err := &wenmar.APIError{Code: "not_found", StatusCode: 404}
	code := ExitCode(err)
	if code != 3 {
		t.Errorf("expected 3 for not found, got %d", code)
	}
}

func TestExitCode_Validation(t *testing.T) {
	err := &wenmar.APIError{Code: "validation_failed", StatusCode: 422}
	code := ExitCode(err)
	if code != 4 {
		t.Errorf("expected 4 for validation, got %d", code)
	}
}

func TestExitCode_ServerError(t *testing.T) {
	err := &wenmar.APIError{Code: "internal_error", StatusCode: 500}
	code := ExitCode(err)
	if code != 6 {
		t.Errorf("expected 6 for server error, got %d", code)
	}
}

func TestExitCode_UnknownError(t *testing.T) {
	err := errors.New("some generic error")
	code := ExitCode(err)
	if code != 1 {
		t.Errorf("expected 1 for generic error, got %d", code)
	}
}

func TestExitCode_UnknownAPIError(t *testing.T) {
	err := &wenmar.APIError{Code: "something_new", StatusCode: 500}
	code := ExitCode(err)
	if code != 6 {
		t.Errorf("expected 6 for unknown 5xx, got %d", code)
	}
}

func TestExitCode_Forbidden(t *testing.T) {
	err := &wenmar.APIError{Code: "forbidden", StatusCode: 403}
	code := ExitCode(err)
	if code != 8 {
		t.Errorf("expected 8 for forbidden, got %d", code)
	}
}

func TestExitCode_ForbiddenByStatus(t *testing.T) {
	err := &wenmar.APIError{Code: "unknown", StatusCode: 403}
	code := ExitCode(err)
	if code != 8 {
		t.Errorf("expected 8 for 403, got %d", code)
	}
}

func TestExitCode_Conflict(t *testing.T) {
	err := &wenmar.APIError{Code: "conflict", StatusCode: 409}
	code := ExitCode(err)
	if code != 7 {
		t.Errorf("expected 7 for conflict, got %d", code)
	}
}

func TestExitCode_ConflictByStatus(t *testing.T) {
	err := &wenmar.APIError{Code: "unknown", StatusCode: 409}
	code := ExitCode(err)
	if code != 7 {
		t.Errorf("expected 7 for 409, got %d", code)
	}
}

func TestExitCode_Partial(t *testing.T) {
	err := &PartialError{Message: "truncated"}
	code := ExitCode(err)
	if code != 9 {
		t.Errorf("expected 9 for partial, got %d", code)
	}
}

func TestExitCode_Offline(t *testing.T) {
	err := &net.DNSError{IsNotFound: true}
	code := ExitCode(err)
	if code != 10 {
		t.Errorf("expected 10 for network error, got %d", code)
	}
}
