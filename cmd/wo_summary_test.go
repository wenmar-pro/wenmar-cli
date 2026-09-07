package cmd

import (
	"strings"
	"testing"
)

func TestWoActivity_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute("wo", "activity", "1", "--json", "--base-url", srv.URL, "--token", "secret-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"action"`) {
		t.Errorf("expected activity in output, got: %s", out)
	}
}

func TestWoVehicleHistory_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute("wo", "vehicle-history", "1", "--json", "--base-url", srv.URL, "--token", "secret-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"work_order_number"`) {
		t.Errorf("expected history in output, got: %s", out)
	}
}

func TestWoAppointments_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute("wo", "appointments", "1", "--json", "--base-url", srv.URL, "--token", "secret-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"starts_at"`) {
		t.Errorf("expected appointments in output, got: %s", out)
	}
}

func TestWoAuthLogs_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute("wo", "auth-logs", "1", "--json", "--base-url", srv.URL, "--token", "secret-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"event_type"`) {
		t.Errorf("expected authorization logs in output, got: %s", out)
	}
}

func TestWoNotesAdd_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute("wo", "notes", "add", "1", "--body", "Customer approved estimate", "--json", "--base-url", srv.URL, "--token", "secret-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Note added") {
		t.Errorf("expected confirmation, got: %s", out)
	}
}
