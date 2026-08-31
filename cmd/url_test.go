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

func TestParseWenmarURL_CanonicalAndLegacyResources(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want string // resource_type ("" = unknown)
	}{
		{"workorders canonical", "https://app.wenmarpro.com/workorders/42.json", "workorders"},
		{"work_orders legacy still parses", "https://app.wenmarpro.com/work_orders/42.json", "work_orders"},
		{"servicecategories", "https://app.wenmarpro.com/servicecategories/7.json", "servicecategories"},
		{"service-categories legacy", "https://app.wenmarpro.com/service-categories/7.json", "service-categories"},
		{"vendors", "https://app.wenmarpro.com/vendors/3.json", "vendors"},
		{"drivers", "https://app.wenmarpro.com/drivers/9.json", "drivers"},
		{"statements", "https://app.wenmarpro.com/statements/9001.json", "statements"},
		{"tags", "https://app.wenmarpro.com/customer_tags/5.json", "customer_tags"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseWenmarURL(tc.url)
			if got.ResourceType != tc.want {
				t.Errorf("parseWenmarURL(%q).ResourceType = %q, want %q", tc.url, got.ResourceType, tc.want)
			}
		})
	}
}
