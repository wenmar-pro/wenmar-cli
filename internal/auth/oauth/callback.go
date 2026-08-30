package oauth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
)

// WaitForCallback starts an HTTP server on the given listener and blocks
// until the OAuth provider redirects to it with a code, or the context
// is cancelled. It validates the state parameter and returns the
// authorization code.
//
// The listener should be net.Listen("tcp", "127.0.0.1:0") for a random
// free port. The caller should construct the redirect_uri from
// listener.Addr() before starting the flow.
func WaitForCallback(ctx context.Context, expectedState string, listener net.Listener) (string, error) {
	type result struct {
		code string
		err  error
	}
	resultCh := make(chan result, 1)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}

			query := r.URL.Query()

			// Check for OAuth error response
			if errCode := query.Get("error"); errCode != "" {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "Authorization error: %s", errCode)
				resultCh <- result{err: fmt.Errorf("authorization error: %s", errCode)}
				return
			}

			// Validate state
			state := query.Get("state")
			if state != expectedState {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "State mismatch")
				resultCh <- result{err: errors.New("state mismatch: callback state did not match the login request (possible CSRF); retry the login")}
				return
			}

			// Extract code
			code := query.Get("code")
			if code == "" {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "Missing code parameter")
				resultCh <- result{err: fmt.Errorf("missing code parameter")}
				return
			}

			// Success — show a simple page to the user
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "<html><body><h2>Authentication successful</h2><p>You can close this tab and return to the terminal.</p></body></html>")

			resultCh <- result{code: code}
		}),
	}

	// Start serving in a goroutine
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	// Wait for callback, context cancellation, or server error
	select {
	case res := <-resultCh:
		_ = server.Shutdown(context.Background())
		if res.err != nil {
			return "", res.err
		}
		return res.code, nil
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())
		return "", ctx.Err()
	case err := <-serveErr:
		if err != nil && err != http.ErrServerClosed {
			return "", fmt.Errorf("callback server error: %w", err)
		}
		return "", fmt.Errorf("callback server closed unexpectedly")
	}
}
