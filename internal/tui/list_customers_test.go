package tui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func TestCustomerList_FetchesCustomers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":7,"full_name":"Jane Doe","type":"individual","vehicles_count":2,"outstanding_balance_cents":12345,"updated_at":"2026-01-02T00:00:00Z"}]`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL, "test")
	msg := fetchCustomers(client, "")()

	res, ok := msg.(customerListResultMsg)
	if !ok {
		t.Fatalf("expected customerListResultMsg, got %T", msg)
	}
	if res.err != nil {
		t.Fatalf("unexpected error: %v", res.err)
	}
	if len(res.items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(res.items))
	}
	c := res.items[0]
	if c.FullName != "Jane Doe" {
		t.Errorf("expected 'Jane Doe', got %q", c.FullName)
	}
	if c.Type != "individual" {
		t.Errorf("expected type 'individual', got %q", c.Type)
	}
	if c.VehiclesCount != 2 {
		t.Errorf("expected vehicles count 2, got %d", c.VehiclesCount)
	}
	if c.OutstandingBalanceCents != 12345 {
		t.Errorf("expected balance 12345, got %d", c.OutstandingBalanceCents)
	}
}

func TestCustomerList_FetchWithQuerySendsRequestParam(t *testing.T) {
	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":7,"full_name":"Jane Doe","type":"individual","vehicles_count":0,"outstanding_balance_cents":0,"updated_at":"2026-01-02T00:00:00Z"}]`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL, "test")
	params := wenmar.ListCustomersParams{
		Q: strPtr("jane"),
	}
	msg := fetchCustomersWithParams(client, "", params)()

	res, ok := msg.(customerListResultMsg)
	if !ok {
		t.Fatalf("expected customerListResultMsg, got %T", msg)
	}
	if res.err != nil {
		t.Fatalf("unexpected error: %v", res.err)
	}
	if capturedQuery != "jane" {
		t.Fatalf("expected query param 'jane', got %q", capturedQuery)
	}
	if len(res.items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(res.items))
	}
}

func TestCustomerList_FetchWithTypeSendsTypeParam(t *testing.T) {
	var capturedType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedType = r.URL.Query().Get("type")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":7,"full_name":"Acme Co","type":"business","vehicles_count":0,"outstanding_balance_cents":0,"updated_at":"2026-01-02T00:00:00Z"}]`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL, "test")
	params := wenmar.ListCustomersParams{
		Type: strPtr("business"),
	}
	msg := fetchCustomersWithParams(client, "", params)()

	res, ok := msg.(customerListResultMsg)
	if !ok {
		t.Fatalf("expected customerListResultMsg, got %T", msg)
	}
	if res.err != nil {
		t.Fatalf("unexpected error: %v", res.err)
	}
	if capturedType != "business" {
		t.Fatalf("expected type param 'business', got %q", capturedType)
	}
}

func TestCustomerList_FetchWithHasBalanceSendsBoolParam(t *testing.T) {
	var capturedHasBalance string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHasBalance = r.URL.Query().Get("has_balance")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL, "test")
	params := wenmar.ListCustomersParams{
		HasBalance: boolPtr(true),
	}
	msg := fetchCustomersWithParams(client, "", params)()

	res, ok := msg.(customerListResultMsg)
	if !ok {
		t.Fatalf("expected customerListResultMsg, got %T", msg)
	}
	if res.err != nil {
		t.Fatalf("unexpected error: %v", res.err)
	}
	if capturedHasBalance != "true" {
		t.Fatalf("expected has_balance param 'true', got %q", capturedHasBalance)
	}
}
