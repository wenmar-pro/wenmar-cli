package oauth

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	authpkg "github.com/wenmar-pro/wenmar-sdk/go/pkg/auth"
)

// ClientID is the fixed Doorkeeper application UID for the Wenmar CLI.
// This is a public client (confidential: false) — the UID is not secret.
const ClientID = "wenmar-cli"

// LoginTimeout is the maximum time to wait for the user to complete
// the browser-based authorization.
const LoginTimeout = 5 * time.Minute

// Login runs the full OAuth Authorization Code + PKCE flow:
//  1. Generate PKCE verifier, challenge, and state
//  2. Start a localhost callback server on a random port
//  3. Open the browser to the authorization endpoint
//  4. Wait for the callback with the authorization code
//  5. Exchange the code for an access token
//
// baseURL is the API base URL (e.g., "https://app.wenmarpro.com" or
// "http://localhost:3000"). The OAuth endpoints are derived as
// {baseURL}/oauth/authorize and {baseURL}/oauth/token.
func Login(ctx context.Context, baseURL string) (*authpkg.Token, error) {
	return LoginWithClientID(ctx, baseURL, ClientID)
}

// LoginWithClientID is Login with an injectable client ID (for testing).
func LoginWithClientID(ctx context.Context, baseURL, clientID string) (*authpkg.Token, error) {
	// 1. Generate PKCE
	verifier := GenerateVerifier()
	challenge := GenerateChallenge(verifier)
	state := GenerateState()

	// 2. Start callback server on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start callback server: %w", err)
	}
	redirectURI := "http://" + listener.Addr().String() + "/callback"

	// 3. Build the authorization URL
	authURL, err := buildAuthURL(baseURL, clientID, redirectURI, challenge, state)
	if err != nil {
		listener.Close()
		return nil, err
	}

	// 4. Open browser
	if err := openBrowser(authURL); err != nil {
		// If we can't open the browser, print the URL and continue
		// The caller can check stderr for the URL
		fmt.Fprintf(stderr(), "Could not open browser automatically. Please open this URL:\n%s\n", authURL)
	}

	// 5. Wait for callback with timeout
	callbackCtx, cancel := context.WithTimeout(ctx, LoginTimeout)
	defer cancel()

	code, err := WaitForCallback(callbackCtx, state, listener)
	if err != nil {
		return nil, fmt.Errorf("OAuth callback failed: %w", err)
	}

	// 6. Exchange code for token
	tokenEndpoint := baseURL + "/oauth/token"
	token, err := ExchangeCode(ctx, tokenEndpoint, code, redirectURI, clientID, verifier)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	return token, nil
}

// buildAuthURL constructs the authorization endpoint URL with PKCE parameters.
func buildAuthURL(baseURL, clientID, redirectURI, challenge, state string) (string, error) {
	u, err := url.Parse(baseURL + "/oauth/authorize")
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
	}

	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// stderr returns os.Stderr. Wrapped for testability.
var stderr = func() *os.File {
	return os.Stderr
}
