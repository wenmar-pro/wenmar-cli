package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestLogin_FullFlow tests the complete OAuth flow against a mock server.
// It can't test the browser opening (that's platform-specific), but it
// tests everything else: PKCE generation, auth URL construction, callback
// handling, and code exchange.
func TestLogin_FullFlow(t *testing.T) {
	// We can't test the full flow with browser opening in a unit test,
	// but we can test the individual pieces work together by calling
	// the internal functions directly.

	// 1. Generate PKCE
	verifier, err := GenerateVerifier()
	if err != nil {
		t.Fatalf("GenerateVerifier: %v", err)
	}
	challenge := GenerateChallenge(verifier)
	state, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState: %v", err)
	}

	if verifier == "" || challenge == "" || state == "" {
		t.Fatal("PKCE generation returned empty values")
	}

	// 2. Set up a mock OAuth server that handles both authorize and token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/authorize":
			// Simulate auto-approve: redirect to the redirect_uri with a code
			redirectURI := r.URL.Query().Get("redirect_uri")
			responseState := r.URL.Query().Get("state")
			redirectURL := fmt.Sprintf("%s?code=mock-auth-code&state=%s", redirectURI, responseState)
			http.Redirect(w, r, redirectURL, http.StatusFound)

		case "/oauth/token":
			if err := r.ParseForm(); err != nil {
				t.Errorf("ParseForm: %v", err)
				return
			}
			// Verify PKCE verifier is sent
			if r.PostForm.Get("code_verifier") != verifier {
				t.Errorf("code_verifier mismatch")
			}
			if r.PostForm.Get("code") != "mock-auth-code" {
				t.Errorf("code = %q, want mock-auth-code", r.PostForm.Get("code"))
			}

			json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "test-access-token",
				"refresh_token": "test-refresh-token",
				"token_type":    "Bearer",
				"expires_in":    7200,
			})

		default:
			t.Errorf("unexpected request to %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 3. Start a callback listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	redirectURI := "http://" + listener.Addr().String() + "/callback"

	// 4. Simulate the browser hitting the authorize endpoint and following
	//    the redirect to our callback server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the callback waiter
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		code, err := WaitForCallback(ctx, state, listener)
		if err != nil {
			errCh <- err
			return
		}
		codeCh <- code
	}()

	// Simulate browser: hit authorize endpoint, follow redirect to callback
	go func() {
		time.Sleep(50 * time.Millisecond)

		// Build the authorize URL with PKCE
		authURL := server.URL + "/oauth/authorize?" + url.Values{
			"client_id":             {"wenmar-cli"},
			"redirect_uri":          {redirectURI},
			"response_type":         {"code"},
			"state":                 {state},
			"code_challenge":        {challenge},
			"code_challenge_method": {"S256"},
		}.Encode()

		// Hit the authorize endpoint (follow redirects manually to hit our callback)
		resp, err := http.Get(authURL)
		if err != nil {
			// If the error is because the callback server responded and closed,
			// that's fine — the redirect went to our localhost server
			if !strings.Contains(err.Error(), "connection reset") {
				t.Errorf("authorize request failed: %v", err)
			}
			return
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body)
	}()

	// 5. Wait for the callback to capture the code
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		t.Fatalf("WaitForCallback failed: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for callback")
	}

	if code != "mock-auth-code" {
		t.Fatalf("code = %q, want mock-auth-code", code)
	}

	// 6. Exchange the code for a token
	token, err := ExchangeCode(ctx, server.URL+"/oauth/token", code, redirectURI, "wenmar-cli", verifier)
	if err != nil {
		t.Fatalf("ExchangeCode failed: %v", err)
	}

	if token.AccessToken != "test-access-token" {
		t.Errorf("AccessToken = %q, want test-access-token", token.AccessToken)
	}
	if token.RefreshToken != "test-refresh-token" {
		t.Errorf("RefreshToken = %q, want test-refresh-token", token.RefreshToken)
	}
	if token.ExpiresAt == nil {
		t.Fatal("ExpiresAt is nil")
	}
}
