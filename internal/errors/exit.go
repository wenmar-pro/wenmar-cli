package errors

import (
	"errors"
	"net"

	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

const (
	ExitSuccess    = 0
	ExitGeneric    = 1
	ExitAuth       = 2
	ExitNotFound   = 3
	ExitValidation = 4
	ExitRateLimit  = 5
	ExitServer     = 6
	ExitConflict   = 7
	ExitForbidden  = 8
	ExitPartial    = 9
	ExitOffline    = 10
)

// PartialError signals a truncated response that was rejected without
// --allow-partial.
type PartialError struct {
	Message string
}

func (e *PartialError) Error() string { return e.Message }

func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}

	var partialErr *PartialError
	if errors.As(err, &partialErr) {
		return ExitPartial
	}

	var apiErr *wenmar.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case "unauthorized":
			return ExitAuth
		case "not_found":
			return ExitNotFound
		case "validation_failed":
			return ExitValidation
		case "rate_limited":
			return ExitRateLimit
		case "forbidden":
			return ExitForbidden
		case "conflict":
			return ExitConflict
		case "internal_error":
			return ExitServer
		default:
			if apiErr.StatusCode >= 500 {
				return ExitServer
			}
			if apiErr.StatusCode == 403 {
				return ExitForbidden
			}
			if apiErr.StatusCode == 409 {
				return ExitConflict
			}
			return ExitGeneric
		}
	}

	// Network unreachable / DNS failure.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return ExitOffline
	}

	return ExitGeneric
}
