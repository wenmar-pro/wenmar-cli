package cmd

import "testing"

func TestParseURL_Show(t *testing.T) {
	r := parseWenmarURL("https://app.wenmarpro.com/vehicles/13.json")
	if r.ResourceType != "vehicles" {
		t.Errorf("expected 'vehicles', got '%s'", r.ResourceType)
	}
	if r.ID != "13" {
		t.Errorf("expected id '13', got '%s'", r.ID)
	}
	if r.Format != "json" {
		t.Errorf("expected format 'json', got '%s'", r.Format)
	}
	if r.Host != "app.wenmarpro.com" {
		t.Errorf("expected host 'app.wenmarpro.com', got '%s'", r.Host)
	}
}

func TestParseURL_CustomersShow(t *testing.T) {
	r := parseWenmarURL("https://app.wenmarpro.com/customers/7.json")
	if r.ResourceType != "customers" || r.ID != "7" {
		t.Errorf("expected customers/7, got %s/%s", r.ResourceType, r.ID)
	}
}

func TestParseURL_WorkOrdersShow(t *testing.T) {
	r := parseWenmarURL("https://app.wenmarpro.com/work_orders/42.json")
	if r.ResourceType != "work_orders" || r.ID != "42" {
		t.Errorf("expected work_orders/42, got %s/%s", r.ResourceType, r.ID)
	}
}

func TestParseURL_CollectionWithQuery(t *testing.T) {
	r := parseWenmarURL("https://app.wenmarpro.com/vehicles.json?customer_id=7")
	if r.ResourceType != "vehicles" {
		t.Errorf("expected 'vehicles', got '%s'", r.ResourceType)
	}
	if r.ID != "" {
		t.Errorf("expected no id, got '%s'", r.ID)
	}
	if r.QueryParams["customer_id"] != "7" {
		t.Errorf("expected query customer_id=7, got '%v'", r.QueryParams)
	}
}

func TestParseURL_UnknownPath(t *testing.T) {
	r := parseWenmarURL("https://app.wenmarpro.com/reports/summary")
	if r.ResourceType != "" {
		t.Errorf("expected no resource type for unknown path, got '%s'", r.ResourceType)
	}
	if r.Path != "/reports/summary" {
		t.Errorf("expected path preserved, got '%s'", r.Path)
	}
}
