package cmd

import (
	"strings"
	"testing"
)

func TestWoPaymentsList_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"wo", "payments", "list", "1",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "app_url") {
		t.Errorf("expected work order payments in output, got: %s", out)
	}
}

func TestWoPaymentsAdd_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"wo", "payments", "add", "1",
		"--amount-cents", "10000", "--method", "credit_card",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"id"`) {
		t.Errorf("expected created payment in output, got: %s", out)
	}
}

func TestWoPaymentsReverseAr_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"wo", "payments", "reverse-ar", "1",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "AR reversal processed") {
		t.Errorf("expected confirmation, got: %s", out)
	}
}

func TestWoPaymentsSendToAr_JSON(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"wo", "payments", "send-to-ar", "1",
		"--json", "--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Sent to AR") {
		t.Errorf("expected confirmation, got: %s", out)
	}
}