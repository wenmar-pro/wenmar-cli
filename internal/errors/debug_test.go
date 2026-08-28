package errors

import (
	"bytes"
	"strings"
	"testing"

	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

func TestPrintError_APIError_IncludesDebugContext(t *testing.T) {
	apiErr := &wenmar.APIError{
		Code:       "unauthorized",
		Message:    "Invalid or missing API token",
		StatusCode: 401,
		Method:     "GET",
		Path:       "/customers",
	}
	info := &DebugInfo{
		TokenSource: "WENMAR_TOKEN env",
		TokenMasked: "abcd...wxyz",
		BaseURL:     "http://localhost:3000",
	}

	var buf bytes.Buffer
	PrintError(&buf, apiErr, info)
	out := buf.String()

	for _, want := range []string{
		"ERROR: GET /customers -> unauthorized: Invalid or missing API token (HTTP 401)",
		"token:    abcd...wxyz  (WENMAR_TOKEN env)",
		"base URL: http://localhost:3000",
		"request:  GET /customers",
		"status:   401",
		"Hint: the token may be invalid",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestPrintError_ValidationDetails(t *testing.T) {
	apiErr := &wenmar.APIError{
		Code:       "validation_failed",
		Message:    "Full name can't be blank",
		StatusCode: 422,
		Method:     "POST",
		Path:       "/customers",
		FieldErrorsMap: map[string]any{"full_name": []any{"can't be blank"}},
	}
	info := &DebugInfo{TokenSource: "--token flag", TokenMasked: "abcd...wxyz", BaseURL: "http://localhost:3000"}

	var buf bytes.Buffer
	PrintError(&buf, apiErr, info)
	out := buf.String()

	if !strings.Contains(out, "field errors:") {
		t.Errorf("expected field errors section, got:\n%s", out)
	}
	if !strings.Contains(out, "full_name: [can't be blank]") {
		t.Errorf("expected full_name field error, got:\n%s", out)
	}
	if !strings.Contains(out, "Hint: fix the field errors above") {
		t.Errorf("expected validation hint, got:\n%s", out)
	}
}

func TestPrintError_PlainError_ShowsDebugInfo(t *testing.T) {
	info := &DebugInfo{TokenSource: "config file", TokenMasked: "abcd...wxyz", BaseURL: "http://localhost:3000"}

	var buf bytes.Buffer
	PrintError(&buf, &plainError{}, info)
	out := buf.String()

	if !strings.Contains(out, "ERROR: boom") {
		t.Errorf("expected error message, got:\n%s", out)
	}
	if !strings.Contains(out, "token:    abcd...wxyz  (config file)") {
		t.Errorf("expected token line, got:\n%s", out)
	}
	if !strings.Contains(out, "base URL: http://localhost:3000") {
		t.Errorf("expected base URL line, got:\n%s", out)
	}
}

func TestPrintError_NoInfo_StillPrintsError(t *testing.T) {
	apiErr := &wenmar.APIError{Code: "not_found", Message: "Customer not found", StatusCode: 404}

	var buf bytes.Buffer
	PrintError(&buf, apiErr, nil)
	out := buf.String()

	if !strings.Contains(out, "ERROR: not_found: Customer not found (HTTP 404)") {
		t.Errorf("expected error message, got:\n%s", out)
	}
}

func TestMaskToken(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"abc", "***"},
		{"abcdefgh", "********"},
		{"abcdefghijkl", "abcd...ijkl"},
	}
	for _, c := range cases {
		if got := MaskToken(c.in); got != c.want {
			t.Errorf("MaskToken(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

type plainError struct{}

func (e *plainError) Error() string { return "boom" }
