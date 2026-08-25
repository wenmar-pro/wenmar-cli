package output

import (
	"bytes"
	"testing"
)

func TestRender_MD(t *testing.T) {
	var buf bytes.Buffer
	data := []map[string]any{
		{"id": 1, "name": "Jane"},
		{"id": 2, "name": "John"},
	}
	err := Render(&buf, data, "2 of 2 customers", nil, Options{Mode: ModeMD})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !contains(output, "|") {
		t.Error("expected GFM table with pipes")
	}
	if !contains(output, "Jane") {
		t.Error("expected data in output")
	}
}

func TestRender_JSON(t *testing.T) {
	var buf bytes.Buffer
	data := []map[string]any{
		{"id": 1, "name": "Jane"},
	}
	meta := &Meta{Page: 1}
	err := Render(&buf, data, "1 customer", meta, Options{Mode: ModeJSON})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !contains(output, `"ok": true`) {
		t.Error("expected ok:true in JSON envelope")
	}
	if !contains(output, `"summary"`) {
		t.Error("expected summary in JSON envelope")
	}
}

func TestRender_Agent(t *testing.T) {
	var buf bytes.Buffer
	data := []map[string]any{
		{"id": 1, "name": "Jane"},
	}
	err := Render(&buf, data, "", nil, Options{Mode: ModeAgent})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !contains(output, `"name": "Jane"`) {
		t.Error("expected raw data without envelope")
	}
	if contains(output, `"ok"`) {
		t.Error("expected no envelope in agent mode")
	}
}

func TestRender_DefaultIsMD(t *testing.T) {
	var buf bytes.Buffer
	data := []map[string]any{{"id": 1}}
	err := Render(&buf, data, "", nil, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !contains(buf.String(), "|") {
		t.Error("expected GFM table as default output")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
