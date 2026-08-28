package tui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

func newTestClient(t *testing.T, url, token string) *wenmar.Client {
	t.Helper()
	cfg := wenmar.DefaultConfig()
	cfg.BaseURL = url
	cfg.MaxRetries = 0
	cfg.CacheEnabled = false
	c, err := wenmar.NewClient(cfg, wenmar.NewStaticTokenProvider(token))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return c
}

func TestDetailModel_FetchesWorkOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"work_order_number":42,"status":"in_progress","customer":{"id":1,"full_name":"Jane Doe"},"vehicle":{"id":1,"make":"Honda","model":"Civic","year":2020,"vin":"ABC123"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL, "test")
	msg := fetchWorkOrderDetail(client, "", 1)()

	res, ok := msg.(detailResultMsg)
	if !ok {
		t.Fatalf("expected detailResultMsg, got %T", msg)
	}
	if res.err != nil {
		t.Fatalf("unexpected error: %v", res.err)
	}
	if res.wo == nil {
		t.Fatal("expected work order")
	}
	if res.wo.WorkOrderNumber != 42 {
		t.Errorf("expected work order number 42, got %d", res.wo.WorkOrderNumber)
	}
	if res.wo.Customer.FullName != "Jane Doe" {
		t.Errorf("expected customer 'Jane Doe', got %q", res.wo.Customer.FullName)
	}
}
