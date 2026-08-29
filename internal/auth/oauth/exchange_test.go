package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExchangeCode_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.PostForm.Get("grant_type") != "authorization_code" {
			t.Errorf("grant_type = %q, want authorization_code", r.PostForm.Get("grant_type"))
		}
		if r.PostForm.Get("code") != "test-code" {
			t.Errorf("code = %q, want test-code", r.PostForm.Get("code"))
		}
		if r.PostForm.Get("client_id") != "wenmar-cli" {
			t.Errorf("client_id = %q, want wenmar-cli", r.PostForm.Get("client_id"))
		}
		if r.PostForm.Get("redirect_uri") != "http://127.0.0.1:12345/callback" {
			t.Errorf("redirect_uri = %q, want http://127.0.0.1:12345/callback", r.PostForm.Get("redirect_uri"))
		}
		if r.PostForm.Get("code_verifier") != "test-verifier" {
			t.Errorf("code_verifier = %q, want test-verifier", r.PostForm.Get("code_verifier"))
		}

		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-abc",
			"refresh_token": "refresh-def",
			"token_type":    "Bearer",
			"expires_in":    7200,
		})
	}))
	defer server.Close()

	token, err := ExchangeCode(context.Background(),
		server.URL+"/oauth/token",
		"test-code",
		"http://127.0.0.1:12345/callback",
		"wenmar-cli",
		"test-verifier",
	)
	if err != nil {
		t.Fatalf("ExchangeCode failed: %v", err)
	}

	if token.AccessToken != "access-abc" {
		t.Errorf("AccessToken = %q, want access-abc", token.AccessToken)
	}
	if token.RefreshToken != "refresh-def" {
		t.Errorf("RefreshToken = %q, want refresh-def", token.RefreshToken)
	}
	if token.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want Bearer", token.TokenType)
	}
	if token.ExpiresAt == nil {
		t.Fatal("ExpiresAt is nil")
	}
}

func TestExchangeCode_InvalidGrant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
	}))
	defer server.Close()

	_, err := ExchangeCode(context.Background(), server.URL, "bad-code", "http://127.0.0.1:12345/callback", "wenmar-cli", "verifier")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExchangeCode_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := ExchangeCode(context.Background(), server.URL, "code", "http://127.0.0.1:12345/callback", "wenmar-cli", "verifier")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExchangeCode_EmptyAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "",
			"expires_in":   7200,
		})
	}))
	defer server.Close()

	_, err := ExchangeCode(context.Background(), server.URL, "code", "http://127.0.0.1:12345/callback", "wenmar-cli", "verifier")
	if err == nil {
		t.Fatal("expected error for empty access_token, got nil")
	}
}