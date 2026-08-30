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
			// Status-code fallbacks keep the documented exit-code contract
			// intact when the server sends an unrecognized error Code.
			switch apiErr.StatusCode {
			case 401:
				return ExitAuth
			case 404:
				return ExitNotFound
			case 422:
				return ExitValidation
			case 429:
				return ExitRateLimit
			case 403:
				return ExitForbidden
			case 409:
				return ExitConflict
			}
			if apiErr.StatusCode >= 500 {
				return ExitServer
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
