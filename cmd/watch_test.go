package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWatch_ExitOnFirst(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []map[string]any{
			{"id": 1, "status": "open"},
		})
	}))
	defer srv.Close()

	_, err := execute(
		"watch", "--base-url", srv.URL, "--token", "secret-token",
		"--interval", "100ms", "--exit-on-first", "--json",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With --exit-on-first, the first poll establishes baseline (no events),
	// then exits. Should not error.
}
