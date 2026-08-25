package output

import (
	"bytes"
	"testing"
)

func TestRender_JQ(t *testing.T) {
	var buf bytes.Buffer
	data := []map[string]any{
		{"id": 1, "full_name": "Jane Doe"},
		{"id": 2, "full_name": "John Smith"},
	}
	err := Render(&buf, data, "", nil, Options{Mode: ModeJQ, JQFilter: ".[].full_name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !contains(output, `"Jane Doe"`) {
		t.Errorf("expected filtered names in output, got: %s", output)
	}
	if contains(output, `"id"`) {
		t.Errorf("jq filter should drop id field, got: %s", output)
	}
}

func TestRender_JQInvalidFilter(t *testing.T) {
	var buf bytes.Buffer
	data := []map[string]any{{"id": 1}}
	err := Render(&buf, data, "", nil, Options{Mode: ModeJQ, JQFilter: "["})
	if err == nil {
		t.Fatal("expected error for invalid jq filter")
	}
}
