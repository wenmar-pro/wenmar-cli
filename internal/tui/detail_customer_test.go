package tui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCustomerDetail_Show(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":7,"full_name":"Jane Doe","type":"individual","company_name":"Acme Auto","emails_count":2,"phones_count":1,"addresses":[],"location":{"id":1,"name":"Main Shop","url":"https://x"},"outstanding_balance_cents":12345,"store_credit_cents":500,"total_revenue_cents":99999,"vehicles_count":2,"work_orders_url":"https://wo","marketing_opt_in":true,"tax_exempt":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL, "test")
	msg := fetchCustomerDetail(client, "", 7)()

	res, ok := msg.(customerDetailResultMsg)
	if !ok {
		t.Fatalf("expected customerDetailResultMsg, got %T", msg)
	}
	if res.err != nil {
		t.Fatalf("unexpected error: %v", res.err)
	}
	if res.customer == nil {
		t.Fatal("expected customer")
	}
	c := res.customer
	if c.FullName != "Jane Doe" {
		t.Errorf("expected 'Jane Doe', got %q", c.FullName)
	}
	if c.OutstandingBalanceCents != 12345 {
		t.Errorf("expected balance 12345, got %d", c.OutstandingBalanceCents)
	}
	if c.VehiclesCount != 2 {
		t.Errorf("expected vehicles 2, got %d", c.VehiclesCount)
	}
}
