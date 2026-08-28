package tui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWorkOrderList_FetchPopulatesCustomer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1,"work_order_number":42,"status":"in_progress","customer":{"id":7,"full_name":"Jane Doe"},"vehicle":{"id":13,"make":"Honda","model":"Civic","year":2020,"vin":"ABC123"}}]`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL, "test")
	msg := fetchWorkOrders(client, "")()

	res, ok := msg.(workOrderListResultMsg)
	if !ok {
		t.Fatalf("expected workOrderListResultMsg, got %T", msg)
	}
	if res.err != nil {
		t.Fatalf("unexpected error: %v", res.err)
	}
	if len(res.items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(res.items))
	}
	wo := res.items[0]
	if wo.Customer.FullName != "Jane Doe" {
		t.Errorf("expected customer 'Jane Doe', got %q", wo.Customer.FullName)
	}
	if wo.Vehicle.Make != "Honda" || wo.Vehicle.Model != "Civic" {
		t.Errorf("expected vehicle 'Honda Civic', got %q %q", wo.Vehicle.Make, wo.Vehicle.Model)
	}
}
