package cmd

import (
	"strings"
	"testing"
)

func TestWoServicesList_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"wo", "services", "list", "1",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"work_order_services_count"`) {
		t.Errorf("expected work order estimate in output, got: %s", out)
	}
}

func TestWoServicesAdd_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"wo", "services", "add", "1",
		"--name", "Replace brake pads",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"id"`) {
		t.Errorf("expected created service in output, got: %s", out)
	}
}

func TestWoServicesDelete_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"wo", "services", "delete", "1", "10",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Service deleted") {
		t.Errorf("expected delete confirmation, got: %s", out)
	}
}

func TestWoServicesLineItemsAdd_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"wo", "services", "line-items", "add", "1", "10",
		"--description", "Pads", "--item-type", "part",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"id"`) {
		t.Errorf("expected created line item in output, got: %s", out)
	}
}
