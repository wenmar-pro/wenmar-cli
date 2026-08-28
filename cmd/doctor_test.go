package cmd

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/config"
)

func TestDoctor_AllPass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/account" {
			writeJSON(w, http.StatusOK, map[string]any{"id": 1, "name": "Main Shop"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "test-token", BaseURL: srv.URL})

	out, err := executeWithConfig(configPath, "doctor")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "token") {
		t.Errorf("expected token check in output, got: %s", out)
	}
	if !strings.Contains(out, "connectivity") {
		t.Errorf("expected connectivity check in output, got: %s", out)
	}
}

func TestDoctor_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": 1, "name": "Main Shop"})
	}))
	defer srv.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "test-token", BaseURL: srv.URL})

	out, err := executeWithConfig(configPath, "doctor", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"ok": true`) {
		t.Errorf("expected ok:true in JSON output, got: %s", out)
	}
}

func TestDoctor_NoToken(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")

	_, err := executeWithConfig(configPath, "doctor")
	// Should error — no token configured
	if err == nil {
		t.Error("expected error when no token is configured")
	}
}
