package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/config"
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

	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	if cfg.Token != "test-token" {
		t.Errorf("expected token 'test-token', got '%s'", cfg.Token)
	}
	if cfg.BaseURL != ts.URL {
		t.Errorf("expected base_url '%s', got '%s'", ts.URL, cfg.BaseURL)
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
		Token:   "abcdefghijklmnop",
		BaseURL: "http://localhost:99999", // won't connect, but masking test doesn't need it
	})

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
