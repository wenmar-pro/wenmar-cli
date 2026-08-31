package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderCount_Array(t *testing.T) {
	data := []any{1, 2, 3}
	var buf bytes.Buffer
	err := renderCount(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if out != "3" {
		t.Errorf("expected '3', got %q", out)
	}
}

func TestRenderCount_SingleObject(t *testing.T) {
	data := map[string]any{"id": 1}
	var buf bytes.Buffer
	err := renderCount(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if out != "1" {
		t.Errorf("expected '1', got %q", out)
	}
}

func TestRenderCount_Nil(t *testing.T) {
	var buf bytes.Buffer
	err := renderCount(&buf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if out != "0" {
		t.Errorf("expected '0', got %q", out)
	}
}

func TestRenderCount_EmptyArray(t *testing.T) {
	data := []any{}
	var buf bytes.Buffer
	err := renderCount(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if out != "0" {
		t.Errorf("expected '0', got %q", out)
	}
}
