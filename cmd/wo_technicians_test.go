package cmd

import (
	"strings"
	"testing"
)

func TestWoTechniciansAdd_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"wo", "technicians", "add", "1",
		"--technician-id", "7",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"assigned_technician_id"`) {
		t.Errorf("expected tech assignment in output, got: %s", out)
	}
}
