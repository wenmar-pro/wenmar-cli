package cmd

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/config"
)

func TestLocationUse_SetsDefaultLocation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id":   1,
			"name": "Acme Shop",
			"locations": []map[string]any{
				{"id": 42, "name": "Downtown", "url": "...", "app_url": "..."},
				{"id": 43, "name": "Uptown", "url": "...", "app_url": "..."},
			},
		})
	}))
	defer ts.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "test-token", BaseURL: ts.URL})

	out, err := executeWithConfig(configPath, "location", "use", "--location", "43")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Uptown") {
		t.Errorf("expected location name in output, got %s", out)
	}

	cfg, _ := config.LoadFrom(configPath)
	if cfg.LocationID != "43" {
		t.Errorf("expected location_id 43, got %s", cfg.LocationID)
	}
}
