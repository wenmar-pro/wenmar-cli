package cmd

import (
	"strings"
	"testing"
)

func TestReportsStatements_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"reports", "statements", "--status", "sent",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"statement_number"`) {
		t.Errorf("expected statement in output, got: %s", out)
	}
}

func TestReportsTaxPeriods_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"reports", "taxperiods",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"period_start"`) {
		t.Errorf("expected tax period in output, got: %s", out)
	}
}

func TestReportsCreate_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"reports", "create",
		"--period-start", "2026-01-01", "--period-end", "2026-03-31",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Tax period created") {
		t.Errorf("expected create summary, got: %s", out)
	}
}

func TestReportsUpdate_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"reports", "update", "5",
		"--marked-remitted", "--remitted-date", "2026-04-15",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Tax period updated") {
		t.Errorf("expected update summary, got: %s", out)
	}
}

func TestInventoryExtract_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"inventory", "extract",
		"--extraction-id", "abc123", "--text", "2x oil filter",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "stream-1") {
		t.Errorf("expected stream id in output, got: %s", out)
	}
}
