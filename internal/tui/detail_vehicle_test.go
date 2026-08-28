package tui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVehicleDetail_Show(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":13,"year":2020,"make":"Honda","model":"Civic","vin":"1HGCN2345ABC","customer":{"id":7,"full_name":"Jane Doe","url":"https://c"},"location":{"id":1,"name":"Main Shop","url":"https://l"},"lifetime_revenue_cents":50000,"open_work_orders_count":3,"appointments_count":1,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL, "test")
	msg := fetchVehicleDetail(client, "", 13)()

	res, ok := msg.(vehicleDetailResultMsg)
	if !ok {
		t.Fatalf("expected vehicleDetailResultMsg, got %T", msg)
	}
	if res.err != nil {
		t.Fatalf("unexpected error: %v", res.err)
	}
	if res.vehicle == nil {
		t.Fatal("expected vehicle")
	}
	v := res.vehicle
	if v.Make != "Honda" || v.Model != "Civic" {
		t.Errorf("expected 'Honda Civic', got %q %q", v.Make, v.Model)
	}
	if v.Customer.FullName != "Jane Doe" {
		t.Errorf("expected customer 'Jane Doe', got %q", v.Customer.FullName)
	}
	if v.OpenWorkOrdersCount != 3 {
		t.Errorf("expected open WOs 3, got %d", v.OpenWorkOrdersCount)
	}
}
