package tui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVehicleList_FetchesVehicles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":13,"year":2020,"make":"Honda","model":"Civic","vin":"1HGCN2345ABC","customer":{"id":7,"full_name":"Jane Doe"},"open_work_orders_count":3}]`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL, "test")
	msg := fetchVehicles(client, "")()

	res, ok := msg.(vehicleListResultMsg)
	if !ok {
		t.Fatalf("expected vehicleListResultMsg, got %T", msg)
	}
	if res.err != nil {
		t.Fatalf("unexpected error: %v", res.err)
	}
	if len(res.items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(res.items))
	}
	v := res.items[0]
	if v.Year != 2020 {
		t.Errorf("expected year 2020, got %d", v.Year)
	}
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
