package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/errors"
)

func TestMain(m *testing.M) {
	// Ensure tests never inherit a real token or base URL.
	os.Unsetenv("WENMAR_TOKEN")
	os.Unsetenv("WENMAR_BASE_URL")
	code := m.Run()
	os.Exit(code)
}

// execute runs the root command with the given args and returns stdout/err.
func execute(args ...string) (string, error) {
	// Reset global output flags so prior tests don't leak state.
	mdFlag, jsonFlag, agentFlag, jqFlag = false, false, false, ""
	rootCmd.SetArgs(args)
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	err := rootCmd.Execute()
	return buf.String(), err
}

// startFakeAPI returns an httptest server that mimics the wenmar-pro API
// envelope for the endpoints the CLI calls.
func startFakeAPI(t *testing.T, token string) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// GET/POST /api/customers
	mux.HandleFunc("/api/customers", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{
				"data": []map[string]any{
					{"id": 1, "full_name": "Jane Doe", "email": "jane@test.com"},
					{"id": 2, "full_name": "John Smith", "email": "john@test.com"},
				},
			})
		case http.MethodPost:
			var body struct {
				Customer *struct {
					FullName string  `json:"full_name"`
					Email    *string `json:"email,omitempty"`
				} `json:"customer,omitempty"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			email := ""
			if body.Customer != nil && body.Customer.Email != nil {
				email = *body.Customer.Email
			}
			fullName := ""
			if body.Customer != nil {
				fullName = body.Customer.FullName
			}
			writeJSON(w, http.StatusCreated, map[string]any{
				"data": map[string]any{
					"id":        3,
					"full_name": fullName,
					"email":     email,
				},
			})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// GET /api/customers/:id
	mux.HandleFunc("/api/customers/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/customers/")
		if id == "999999" {
			writeError(w, http.StatusNotFound, "not_found", "Resource not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{"id": 1, "full_name": "Jane Doe"},
		})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{"code": code, "message": msg, "details": map[string]any{}},
	})
}

func TestCustomersList_AgentMode(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"customers", "list", "--agent",
		"--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var data []map[string]any
	if err := json.Unmarshal([]byte(out), &data); err != nil {
		t.Fatalf("agent output should be raw JSON array: %v\n%s", err, out)
	}
	if len(data) != 2 {
		t.Fatalf("expected 2 customers, got %d", len(data))
	}
}

func TestCustomersList_JSONEnvelope(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"customers", "list", "--json",
		"--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var env struct {
		OK      bool             `json:"ok"`
		Summary string           `json:"summary"`
		Data    []map[string]any `json:"data"`
		Meta    struct {
			HasNext bool `json:"has_next"`
		} `json:"meta"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("json envelope should parse: %v\n%s", err, out)
	}
	if !env.OK {
		t.Error("expected ok:true in envelope")
	}
	if len(env.Data) != 2 {
		t.Fatalf("expected 2 customers in data, got %d", len(env.Data))
	}
}

func TestCustomersShow_NotFoundExitCode(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	_, err := execute(
		"customers", "show", "999999",
		"--base-url", srv.URL, "--token", "secret-token",
	)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if code := errors.ExitCode(err); code != 3 {
		t.Errorf("expected exit code 3 for not_found, got %d", code)
	}
}

func TestCustomersCreate_JSON201(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"customers", "create", "--full-name", "New Person", "--email", "new@test.com",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"full_name": "New Person"`) {
		t.Errorf("expected created customer in output, got: %s", out)
	}
}

func TestCustomersList_Markdown(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"customers", "list", "--md",
		"--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "| id |") {
		t.Errorf("expected GFM table header, got: %s", out)
	}
	if !strings.Contains(out, "Jane Doe") {
		t.Errorf("expected customer row, got: %s", out)
	}
}
