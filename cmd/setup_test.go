package cmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/config"
	authpkg "github.com/wenmar-pro/wenmar-sdk/go/pkg/auth"
)

func startFakeAPIReturning401(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/customers", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
	})
	return httptest.NewServer(mux)
}

func TestSetup_WritesConfigOnValidToken(t *testing.T) {
	ts := startFakeAPI(t, "test-token")
	defer ts.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")

	input := strings.NewReader("test-token\n\n")
	var output bytes.Buffer

	err := runSetup(input, &output, configPath, ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Token should be stored in the credential store (file fallback), not config.
	store := authpkg.NewCredentialStore()
	tok, err := store.GetToken(context.Background())
	if err != nil {
		t.Fatalf("token not stored in credential store: %v", err)
	}
	if tok.AccessToken != "test-token" {
		t.Errorf("expected token 'test-token', got '%s'", tok.AccessToken)
	}

	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	if cfg.BaseURL != ts.URL {
		t.Errorf("expected base_url '%s', got '%s'", ts.URL, cfg.BaseURL)
	}
	if cfg.AuthMethod != "static" {
		t.Errorf("expected auth_method 'static', got '%s'", cfg.AuthMethod)
	}

	if !strings.Contains(output.String(), "✓") {
		t.Error("expected success indicator in output")
	}
}

func TestSetup_FailsOnInvalidToken(t *testing.T) {
	ts := startFakeAPIReturning401(t)
	defer ts.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")

	input := strings.NewReader("bad-token\n\n")
	var output bytes.Buffer

	err := runSetup(input, &output, configPath, ts.URL)
	if err == nil {
		t.Error("expected error for invalid token")
	}

	if _, err := config.LoadFrom(configPath); err == nil {
		t.Error("config should not be written on failure")
	}
}

func TestAuthStatus_ShowsMaskedToken(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{
		BaseURL: "http://localhost:99999", // won't connect, but masking test doesn't need it
	})

	// Use the --token flag path (highest precedence) so the test is
	// independent of any keyring/credential-store state.
	oldToken := tokenFlag
	tokenFlag = "abcdefghijklmnop"
	defer func() { tokenFlag = oldToken }()

	var output bytes.Buffer
	_ = runAuthStatus(&output, configPath)

	if strings.Contains(output.String(), "abcdefghijklmnop") {
		t.Error("token should be masked, not shown in full")
	}
	if !strings.Contains(output.String(), "abcd") {
		t.Error("expected masked token prefix in output")
	}
}

func TestAuthLogout_DeletesConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "test"})

	err := runAuthLogout(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("config file should be deleted")
	}
}
