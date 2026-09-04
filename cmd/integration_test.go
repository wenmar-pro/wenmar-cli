package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/errors"
)

// lastPatchBody records the most recent PATCH body seen by the fake API,
// keyed by path. Tests use this to assert request wiring.
var lastPatchBody sync.Map // path -> []byte

// lastDupQuery records the most recent check_duplicate query string.
var lastDupQuery atomic.Value // url.Values

func TestMain(m *testing.M) {
	// Ensure tests never inherit a real token or base URL.
	os.Unsetenv("WENMAR_TOKEN")
	os.Unsetenv("WENMAR_URL")
	// ...and never touch the developer's real credentials: point the
	// config home at a temp dir for the whole test binary.
	cfgHome, err := os.MkdirTemp("", "wenmar-test-config-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "mkdir temp:", err)
		os.Exit(1)
	}
	os.Setenv("WENMAR_CONFIG_HOME", cfgHome)
	defer os.RemoveAll(cfgHome)

	code := m.Run()
	os.Exit(code)
}

// execute runs the root command with the given args and returns stdout/err.
func execute(args ...string) (string, error) {
	// Reset global output flags so prior tests don't leak state.
	jsonFlag, agentFlag, jqFlag, idsOnlyFlag, styledFlag, countFlag = false, false, "", false, false, false
	// Cobra lazily adds a --help flag whose "true" value and Changed bit
	// persist across Execute calls once a test invokes --help; clear it on
	// every command so a later --agent run isn't hijacked into printing help.
	resetHelpFlag(rootCmd)
	// Reset auth flag globals so env-based resolution (WENMAR_URL/TOKEN)
	// isn't shadowed by a prior test's --base-url/--token.
	baseURLFlag, tokenFlag = "", ""
	// Reset repeatable customer flags so prior tests don't leak list state.
	customerEmails, customerPhones = nil, nil
	customerAddresses, customerTagIDs = nil, nil
	customerRemovePhoneIDs = nil
	currentDebugInfo = nil
	rootCmd.SetArgs(args)
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	err := rootCmd.Execute()
	return buf.String(), err
}

// resetHelpFlag clears the cobra-generated --help flag value and Changed bit
// on a command and all of its descendants.
func resetHelpFlag(c *cobra.Command) {
	if hf := c.Flags().Lookup("help"); hf != nil {
		_ = hf.Value.Set("false")
		hf.Changed = false
	}
	for _, sub := range c.Commands() {
		resetHelpFlag(sub)
	}
}

// startFakeAPI returns an httptest server that mimics the wenmar-pro API
// envelope for the endpoints the CLI calls.
func startFakeAPI(t *testing.T, token string) *httptest.Server {
	t.Helper()

	fakeAPIRequests.Store(0)

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
			body, _ := io.ReadAll(r.Body)
			lastPatchBody.Store(r.URL.Path, body)
			writeJSON(w, http.StatusOK, map[string]any{"id": 1, "full_name": "Jane Doe"})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// GET /customers/check_duplicate
	mux.HandleFunc("/customers/check_duplicate", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
			return
		}
		lastDupQuery.Store(r.URL.Query())
		writeJSON(w, http.StatusOK, map[string]any{"matches": []any{}})
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fakeAPIRequests.Add(1)
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// fakeAPIRequests counts requests seen by the fake API, so tests can assert
// that validation fails before any API call is made.
var fakeAPIRequests atomic.Int32

func srvRequestCount() int {
	return int(fakeAPIRequests.Load())
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

func TestCustomersDuplicates_PhoneWiresThrough(t *testing.T) {
	srv := startFakeAPI(t, "tok-dup")
	t.Setenv("WENMAR_URL", srv.URL)
	t.Setenv("WENMAR_TOKEN", "tok-dup")

	if _, err := execute("customers", "duplicates", "--first-name", "Jane", "--phone", "5550100"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q, ok := lastDupQuery.Load().(url.Values)
	if !ok {
		t.Fatal("no check_duplicate query captured")
	}
	if q.Get("phone") != "5550100" {
		t.Errorf("phone param not sent, got %q", q.Get("phone"))
	}
	if q.Get("first_name") != "Jane" {
		t.Errorf("first_name param not sent, got %q", q.Get("first_name"))
	}
}

func TestUnknownSubcommandFails(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"nonexistent customers subcommand", []string{"customers", "delete"}},
		{"typo'd work_orders subcommand", []string{"work_orders", "delet"}},
		{"cross-resource concept", []string{"vehicles", "estimate"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := execute(tc.args...)
			if err == nil {
				t.Errorf("%v: expected error, got exit 0", tc.args)
			}
		})
	}
}

func TestHelpCommandFallback(t *testing.T) {
	// help <command> prints that command's help, not root help.
	out, err := execute("help", "customers")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Manage customers") || !strings.Contains(out, "Available Commands") {
		t.Errorf("help customers printed:\n%s", out)
	}

	out, err = execute("help", "customers", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "List all customers") {
		t.Errorf("help customers list printed:\n%s", out)
	}

	// Topics still win over command names.
	out, err = execute("help", "output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Output Modes") {
		t.Errorf("help output printed:\n%s", out)
	}

	// Unknown help target errors instead of printing root help.
	_, err = execute("help", "nosuchthing")
	if err == nil {
		t.Error("help nosuchthing should error")
	}
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

func TestCustomersList_Styled(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	defer srv.Close()
	out, err := execute(
		"customers", "list", "--styled",
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

func TestCustomersUpdate_PhonesAndEmailsWireThrough(t *testing.T) {
	srv := startFakeAPI(t, "tok-update")
	t.Setenv("WENMAR_URL", srv.URL)
	t.Setenv("WENMAR_TOKEN", "tok-update")

	if _, err := execute("customers", "update", "42",
		"--email", "work|jane@corp.com",
		"--phone", "cell|555-0100",
		"--remove-phone", "7"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, ok := lastPatchBody.Load("/customers/42")
	if !ok {
		t.Fatal("no PATCH body captured at /customers/42")
	}
	var body struct {
		Customer struct {
			EmailsAttributes []struct {
				Email string  `json:"email"`
				Label *string `json:"label"`
			} `json:"emails_attributes"`
			PhonesAttributes []struct {
				UnderscoreDestroy *bool   `json:"_destroy"`
				Id                *int    `json:"id"`
				Label             *string `json:"label"`
				Number            *string `json:"number"`
			} `json:"phones_attributes"`
		} `json:"customer"`
	}
	if err := json.Unmarshal(raw.([]byte), &body); err != nil {
		t.Fatalf("unmarshal captured body: %v\nbody: %s", err, raw.([]byte))
	}
	if n := len(body.Customer.EmailsAttributes); n != 1 || body.Customer.EmailsAttributes[0].Email != "jane@corp.com" {
		t.Errorf("emails_attributes not wired: %+v", body.Customer.EmailsAttributes)
	}
	if n := len(body.Customer.PhonesAttributes); n != 2 {
		t.Fatalf("want 2 phone attrs (1 add + 1 destroy), got %d: %+v", n, body.Customer.PhonesAttributes)
	}
	sawDestroy := false
	for _, p := range body.Customer.PhonesAttributes {
		if p.UnderscoreDestroy != nil && p.Id != nil && *p.Id == 7 {
			sawDestroy = true
		}
	}
	if !sawDestroy {
		t.Errorf("phones_attributes missing {_destroy:true, id:7}: %+v", body.Customer.PhonesAttributes)
	}
}

func TestCustomersUpdate_UnsupportableFlagsRemoved(t *testing.T) {
	cases := map[string]string{
		"remove-email":   "3",
		"remove-address": "5",
		"tag-id":         "11",
		"address":        "1 Main St|Springfield|IL|62704|USA",
	}
	for flag, value := range cases {
		t.Run(flag, func(t *testing.T) {
			srv := startFakeAPI(t, "tok-rm")
			t.Setenv("WENMAR_URL", srv.URL)
			t.Setenv("WENMAR_TOKEN", "tok-rm")

			_, err := execute("customers", "update", "42", "--"+flag, value)
			if err == nil {
				t.Errorf("--%s must error: unsupported by the update API (flag removed)", flag)
			}
		})
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

func TestVehiclesTrash_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"vehicles", "trash", "1",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Vehicle trash") {
		t.Errorf("expected trash confirmation in output, got: %s", out)
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

func TestDroppedOutputFlagsRemoved(t *testing.T) {
	dropped := []string{"--output", "--md", "-m", "--markdown", "--quiet", "--html"}
	for _, flag := range dropped {
		t.Run(flag, func(t *testing.T) {
			args := []string{"customers", "list", flag}
			// --output takes a value; give it one so the error is the
			// unknown-flag error, not a missing-argument error.
			if flag == "--output" {
				args = append(args, "md")
			}
			_, err := execute(args...)
			if err == nil {
				t.Errorf("%s was dropped; it must error", flag)
			}
		})
	}
}

func TestOutputFlags(t *testing.T) {
	srv := startFakeAPI(t, "tok-out")
	defer srv.Close()
	t.Setenv("WENMAR_URL", srv.URL)
	t.Setenv("WENMAR_TOKEN", "tok-out")

	out, err := execute("customers", "list", "--styled")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "| id |") {
		t.Errorf("--styled should render the human table, got:\n%s", out)
	}

	out, err = execute("customers", "list", "--ids-only")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.FieldsFunc(out, func(r rune) bool { return r == '\n' })
	if len(lines) != 2 || lines[0] != "1" || lines[1] != "2" {
		t.Errorf("--ids-only should print one ID per line, got %q", out)
	}

	out, err = execute("customers", "list", "--count")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "2" {
		t.Errorf("--count should print a bare integer, got %q", out)
	}
}

func TestOutputFlagConflictFailsFast(t *testing.T) {
	srv := startFakeAPI(t, "tok-conflict")
	defer srv.Close()
	t.Setenv("WENMAR_URL", srv.URL)

	_, err := execute("customers", "list", "--json", "--agent")
	if err == nil {
		t.Fatal("conflicting output flags must error")
	}
	if n := srvRequestCount(); n != 0 {
		t.Errorf("conflict validation should run before any API call; saw %d requests", n)
	}
}

func TestWorkOrdersDelete_DryRun(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"work_orders", "delete", "42", "--dry-run",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"dry_run": true`) {
		t.Errorf("expected dry_run:true in output, got: %s", out)
	}
}

func TestCompletion_Bash(t *testing.T) {
	out, err := execute("completion", "bash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "bash completion") && !strings.Contains(out, "_wenmar") {
		t.Errorf("expected bash completion script, got: %s", out[:200])
	}
}

func TestCompletion_Zsh(t *testing.T) {
	out, err := execute("completion", "zsh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "_wenmar") {
		t.Errorf("expected zsh completion script, got: %s", out[:200])
	}
}

func TestCompletion_Fish(t *testing.T) {
	out, err := execute("completion", "fish")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "wenmar") {
		t.Errorf("expected fish completion script, got: %s", out[:200])
	}
}

func TestCustomersList_WrongToken_DebugOutput(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	_, err := execute(
		"customers", "list",
		"--base-url", srv.URL, "--token", "wrong-token",
	)
	if err == nil {
		t.Fatal("expected error for wrong token")
	}

	var buf bytes.Buffer
	errors.PrintError(&buf, err, currentDebugInfo)
	out := buf.String()

	for _, want := range []string{
		"ERROR:",
		"token:",
		"base URL:",
		"request:  GET /customers",
		"status:   401",
		"Hint: the token may be invalid",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected debug output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestCustomersShow_NotFound_DebugOutput(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	_, err := execute(
		"customers", "show", "999999",
		"--base-url", srv.URL, "--token", "secret-token",
	)
	if err == nil {
		t.Fatal("expected error for 404")
	}

	var buf bytes.Buffer
	errors.PrintError(&buf, err, currentDebugInfo)
	out := buf.String()

	if !strings.Contains(out, "request:  GET /customers/999999") {
		t.Errorf("expected request line with path, got:\n%s", out)
	}
	if !strings.Contains(out, "status:   404") {
		t.Errorf("expected status 404, got:\n%s", out)
	}
}

func TestCustomersList_DebugFlag_Success(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"customers", "list", "--json", "--debug",
		"--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// stdout must still be clean JSON (debug goes to stderr, which execute
	// does not capture).
	var env struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("stdout should be clean JSON envelope: %v\n%s", err, out)
	}
	if !env.OK {
		t.Error("expected ok:true in envelope")
	}
}
