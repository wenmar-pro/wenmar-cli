package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderHTML_Table(t *testing.T) {
	var buf bytes.Buffer
	data := []map[string]any{
		{"id": 1, "full_name": "Jane Doe"},
		{"id": 2, "full_name": "John Smith"},
	}
	if err := renderHTML(&buf, data, "Customers"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"<!DOCTYPE html>", "<title>Customers</title>", "<table>", "Jane Doe", "John Smith"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected HTML to contain %q, got:\n%s", want, out)
		}
	}
}

func TestRenderHTML_WorkOrder(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"work_order_number": 42,
		"status":            "in_progress",
		"customer":          map[string]any{"full_name": "Jane Doe"},
		"vehicle":           map[string]any{"make": "Honda", "model": "Civic", "year": 2020, "vin": "ABC123"},
	}
	if err := renderHTML(&buf, data, "Work Order"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Work Order 42", "Jane Doe", "Honda", "Civic", "VIN: ABC123"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected HTML to contain %q, got:\n%s", want, out)
		}
	}
}

func TestRenderHTML_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := renderHTML(&buf, []map[string]any{}, "Empty"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No results") {
		t.Errorf("expected 'No results', got:\n%s", buf.String())
	}
}
