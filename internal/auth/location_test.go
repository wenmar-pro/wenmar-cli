package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/config"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

func TestResolveAndSaveLocationID_UsesTokenLocation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id":   1,
			"name": "Acme Shop",
			"locations": []map[string]any{
				{"id": 42, "name": "Downtown", "url": "http://example.com/locations/42.json", "app_url": "http://example.com/locations/42"},
				{"id": 43, "name": "Uptown", "url": "http://example.com/locations/43.json", "app_url": "http://example.com/locations/43"},
			},
		})
	}))
	defer ts.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")

	client, _ := wenmar.NewClient(wenmar.Config{BaseURL: ts.URL}, wenmar.NewStaticTokenProvider("tok"))
	var out bytes.Buffer

	locationID, _, err := ResolveAndSaveLocationID(context.Background(), client, configPath, "42", false, &out, strings.NewReader("y\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if locationID != "42" {
		t.Errorf("expected location 42, got %s", locationID)
	}

	cfg, _ := config.LoadFrom(configPath)
	if cfg.LocationID != "42" {
		t.Errorf("expected saved location_id 42, got %s", cfg.LocationID)
	}
}

func TestResolveAndSaveLocationID_FlagOverridesTokenLocation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"id":   1,
			"name": "Acme Shop",
			"locations": []map[string]any{
				{"id": 42, "name": "Downtown", "url": "...", "app_url": "..."},
			},
		})
	}))
	defer ts.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	client, _ := wenmar.NewClient(wenmar.Config{BaseURL: ts.URL}, wenmar.NewStaticTokenProvider("tok"))
	var out bytes.Buffer

	locationID, _, err := ResolveAndSaveLocationID(context.Background(), client, configPath, "42", false, &out, strings.NewReader("y\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if locationID != "42" {
		t.Errorf("expected flag location 42, got %s", locationID)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
