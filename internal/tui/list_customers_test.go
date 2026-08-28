package tui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
