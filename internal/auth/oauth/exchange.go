package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	authpkg "github.com/wenmar-pro/wenmar-sdk/go/pkg/auth"
)

// noRedirectClient is an HTTP client that does not follow redirects.
// A 3xx from the token endpoint is a security risk, not a redirect to follow.
var noRedirectClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// ExchangeCode exchanges an authorization code for an access token via the
// OAuth 2.0 authorization_code grant with PKCE.
func ExchangeCode(ctx context.Context, tokenEndpoint, code, redirectURI, clientID, codeVerifier string) (*authpkg.Token, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := noRedirectClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read token exchange response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(body, &errResp)
		if errResp.Error != "" {
			return nil, fmt.Errorf("token exchange failed: %s", errResp.Error)
		}
		return nil, fmt.Errorf("token exchange failed: HTTP %d", resp.StatusCode)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse token exchange response: %w", err)
	}

	if result.AccessToken == "" {
		return nil, errors.New("token exchange response missing access_token")
	}

	tokenType := result.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}

	expiresAt := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return &authpkg.Token{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    &expiresAt,
		TokenType:    tokenType,
	}, nil
}