package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderIDsOnly_Array(t *testing.T) {
	data := []map[string]any{
		{"id": float64(1), "name": "Jane"},
		{"id": float64(2), "name": "John"},
	}
	var buf bytes.Buffer
	err := renderIDsOnly(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), out)
	}
	if lines[0] != "1" {
		t.Errorf("expected first line '1', got %q", lines[0])
	}
	if lines[1] != "2" {
		t.Errorf("expected second line '2', got %q", lines[1])
	}
}

func TestRenderIDsOnly_SingleObject(t *testing.T) {
	data := map[string]any{"id": float64(42), "name": "Jane"}
	var buf bytes.Buffer
	err := renderIDsOnly(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if out != "42" {
		t.Errorf("expected '42', got %q", out)
	}
}

func TestRenderIDsOnly_Nil(t *testing.T) {
	var buf bytes.Buffer
	err := renderIDsOnly(&buf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if out != "" {
		t.Errorf("expected empty output for nil, got %q", out)
	}
}
