package errors

import (
	"errors"

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
)

func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
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
		case "internal_error", "unknown":
			return ExitServer
		default:
			if apiErr.StatusCode >= 500 {
				return ExitServer
			}
			return ExitGeneric
		}
	}

	return ExitGeneric
}
