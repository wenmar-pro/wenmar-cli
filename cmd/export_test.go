package cmd

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/errors"
)

// This test file reuses writeJSON/writeError helpers defined in
// cmd/integration_test.go and the execute() harness from the same file.

func TestExport_List(t *testing.T) {
	srv := startExportFakeAPI(t, "tok-list")
	out, err := execute("export", "--list", "--base-url", srv.URL, "--token", "tok-list")
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "customers") {
		t.Errorf("expected customers in list output, got:\n%s", out)
	}
}

func TestExport_SyncInline(t *testing.T) {
	srv := startExportFakeAPI(t, "tok-inline")
	out, err := execute("export", "customers", "--inline", "--base-url", srv.URL, "--token", "tok-inline")
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "id,full_name") {
		t.Errorf("expected CSV content in output, got:\n%s", out)
	}
}

func TestExport_AsyncPollAndWrite(t *testing.T) {
	srv := startExportFakeAPI(t, "tok-poll")
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "customers.csv")

	_, err := execute("export", "customers", "--force-async", "-o", outPath, "--base-url", srv.URL, "--token", "tok-poll")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !strings.Contains(string(data), "id,full_name") {
		t.Errorf("expected CSV content in file, got:\n%s", string(data))
	}
}

func TestExport_FilterValidationError(t *testing.T) {
	srv := startExportFakeAPI(t, "tok-val")
	_, err := execute("export", "customers", "--filter", "staus=active", "--base-url", srv.URL, "--token", "tok-val")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if code := errors.ExitCode(err); code != errors.ExitValidation {
		t.Errorf("expected exit code %d for validation_failed, got %d", errors.ExitValidation, code)
	}
}

// startExportFakeAPI returns a fake API that implements the /exports endpoints.
func startExportFakeAPI(t *testing.T, token string) *httptest.Server {
	t.Helper()
	var status atomic.Value
	status.Store("pending")

	mux := http.NewServeMux()
	mux.HandleFunc("/exports/schema.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "bad token")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"resources": []map[string]any{
				{"name": "customers", "formats": []string{"csv"}, "filters": []map[string]any{{"key": "q", "type": "string", "optional": true}, {"key": "status", "type": "string", "optional": true}}},
				{"name": "inspections", "formats": []string{"json"}, "filters": []map[string]any{{"key": "q", "type": "string", "optional": true}}},
			},
		})
	})

	mux.HandleFunc("/exports.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "bad token")
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Resource string         `json:"resource"`
			Format   string         `json:"format"`
			Filters  map[string]any `json:"filters"`
			Inline   bool           `json:"inline"`
		}
		json.Unmarshal(body, &req)
		if req.Filters != nil {
			if _, ok := req.Filters["staus"]; ok {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
					"error": map[string]any{"code": "invalid_filters", "message": "Unknown filter keys: staus", "details": map[string]any{"valid_filters": []string{"q", "status"}}},
				})
				return
			}
		}
		if req.Inline {
			csv := "id,full_name\n1,Jane Doe\n"
			writeJSON(w, http.StatusOK, map[string]any{
				"status": "complete", "export_log_id": 42, "row_count": 1,
				"download_url": "/exports/42/download", "format": req.Format, "data": base64.StdEncoding.EncodeToString([]byte(csv)),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "pending", "export_log_id": 42, "row_count": 0,
			"download_url": "/exports/42/download", "format": req.Format,
		})
		status.Store("pending")
	})

	mux.HandleFunc("/exports/42/download", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "bad token")
			return
		}
		if status.Load() == "ready" {
			w.Header().Set("Content-Disposition", "attachment; filename=\"customers.csv\"")
			w.Header().Set("Content-Type", "text/csv")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("id,full_name\n1,Jane Doe\n"))
			return
		}
		status.Store("ready")
		writeJSON(w, http.StatusAccepted, map[string]any{"status": "processing", "retry_after": 1})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}
