package errors

import (
	"errors"
	"testing"

	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

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
