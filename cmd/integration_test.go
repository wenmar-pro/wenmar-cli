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

	// GET/POST /customers
	mux.HandleFunc("/customers", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, []map[string]any{
				{"id": 1, "full_name": "Jane Doe", "email": "jane@test.com"},
				{"id": 2, "full_name": "John Smith", "email": "john@test.com"},
			})
		case http.MethodPost:
			var body struct {
				Customer *struct {
					FirstName string `json:"first_name"`
					LastName  string `json:"last_name"`
				} `json:"customer,omitempty"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			firstName := ""
			lastName := ""
			if body.Customer != nil {
				firstName = body.Customer.FirstName
				lastName = body.Customer.LastName
			}
			writeJSON(w, http.StatusCreated, map[string]any{
				"id":        3,
				"full_name": firstName + " " + lastName,
			})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// GET/PATCH /customers/:id
	mux.HandleFunc("/customers/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/customers/")
		if id == "999999" {
			writeError(w, http.StatusNotFound, "not_found", "Resource not found")
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"id": 1, "full_name": "Jane Doe"})
		case http.MethodPatch:
			writeJSON(w, http.StatusOK, map[string]any{"id": 1, "full_name": "Jane Doe"})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// GET /vehicles
	mux.HandleFunc("/vehicles", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
			return
		}
		if r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, []map[string]any{
				{"id": 1, "make": "Toyota", "model": "Camry", "year": 2020, "vin": "ABC123"},
			})
			return
		}
		if r.Method == http.MethodPost {
			writeJSON(w, http.StatusCreated, map[string]any{"id": 9, "make": "Honda", "model": "Civic", "year": 2020})
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	// GET /vehicles/vin_decode
	mux.HandleFunc("/vehicles/vin_decode", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"make": "Honda", "model": "Civic", "vin": "1HGCM82633A004352"})
	})

	// GET /vehicles/check_duplicate
	mux.HandleFunc("/vehicles/check_duplicate", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"matches": []map[string]any{}})
	})

	// GET /vehicles/:id, PATCH /vehicles/:id, DELETE /vehicles/:id
	mux.HandleFunc("/vehicles/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/vehicles/")
		if id == "999999" {
			writeError(w, http.StatusNotFound, "not_found", "Resource not found")
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"id": 1, "make": "Toyota", "model": "Camry", "year": 2020})
		case http.MethodPatch:
			writeJSON(w, http.StatusOK, map[string]any{"id": 1, "make": "Toyota", "model": "Camry", "year": 2020})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// GET/POST /work_orders
	mux.HandleFunc("/work_orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, []map[string]any{
				{"id": 1, "work_order_number": 1, "status": "in_progress"},
			})
		case http.MethodPost:
			writeJSON(w, http.StatusCreated, map[string]any{"id": 10, "work_order_number": 10, "status": "pending"})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// GET/PATCH/DELETE /work_orders/:id
	mux.HandleFunc("/work_orders/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/work_orders/")
		if id == "999999" {
			writeError(w, http.StatusNotFound, "not_found", "Resource not found")
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"id": 1, "work_order_number": 1, "status": "in_progress"})
		case http.MethodPatch:
			writeJSON(w, http.StatusOK, map[string]any{"id": 1, "work_order_number": 1, "status": "completed"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// GET /account
	mux.HandleFunc("/account", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": 1, "name": "Main Shop"})
	})

	// GET /locations/:id
	mux.HandleFunc("/locations/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/locations/")
		if id == "999999" {
			writeError(w, http.StatusNotFound, "not_found", "Resource not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": 1, "name": "Bay 1"})
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

func TestCustomersUpdate_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"customers", "update", "1",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"id": 1`) {
		t.Errorf("expected updated customer in output, got: %s", out)
	}
}

func TestVehiclesList_AgentMode(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"vehicles", "list", "--agent",
		"--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var data []map[string]any
	if err := json.Unmarshal([]byte(out), &data); err != nil {
		t.Fatalf("agent output should be raw JSON array: %v\n%s", err, out)
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 vehicle, got %d", len(data))
	}
}

func TestVehiclesCreate_JSON201(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"vehicles", "create", "--make", "Honda", "--model", "Civic", "--year", "2020", "--customer-id", "1",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"make": "Honda"`) {
		t.Errorf("expected created vehicle in output, got: %s", out)
	}
}

func TestVehiclesDecodeVin_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"vehicles", "decode-vin", "1HGCM82633A004352",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"make": "Honda"`) {
		t.Errorf("expected decoded vehicle in output, got: %s", out)
	}
}

func TestVehiclesDuplicates_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"vehicles", "duplicates", "ABC123",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"matches"`) {
		t.Errorf("expected matches in output, got: %s", out)
	}
}

func TestVehiclesDelete_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"vehicles", "delete", "1",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Vehicle 1 deleted") {
		t.Errorf("expected delete confirmation in output, got: %s", out)
	}
}

func TestWorkOrdersCreate_JSON201(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"work_orders", "create", "--customer-id", "1", "--vehicle-id", "1",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"work_order_number": 10`) {
		t.Errorf("expected created work order in output, got: %s", out)
	}
}

func TestWorkOrdersUpdate_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"work_orders", "update", "1", "--intake-method", "drop_off",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"id": 1`) {
		t.Errorf("expected updated work order in output, got: %s", out)
	}
}

func TestWorkOrdersDelete_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"work_orders", "delete", "1",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Work order 1 deleted") {
		t.Errorf("expected delete confirmation in output, got: %s", out)
	}
}

func TestAccountShow_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"account", "show",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"name": "Main Shop"`) {
		t.Errorf("expected account data in output, got: %s", out)
	}
}

func TestLocationsShow_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"locations", "show", "1",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"name": "Bay 1"`) {
		t.Errorf("expected location data in output, got: %s", out)
	}
}

func TestCustomersList_Count(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"customers", "list", "--count",
		"--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	trimmed := strings.TrimSpace(out)
	if trimmed != "2" {
		t.Errorf("expected count '2', got %q", trimmed)
	}
}
