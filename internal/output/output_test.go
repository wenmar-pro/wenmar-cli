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
	err := Render(&buf, data, "2 of 2 customers", nil, Options{Mode: ModeDefault})
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

func TestRender_MD_LargeIntegerIDs(t *testing.T) {
	var buf bytes.Buffer
	// JSON unmarshalling produces float64 for numbers; large IDs must not
	// render in scientific notation.
	data := []map[string]any{
		{"id": float64(1043910119), "name": "Jane"},
	}
	err := Render(&buf, data, "", nil, Options{Mode: ModeDefault})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if contains(output, "1.043910119e+09") {
		t.Errorf("expected no scientific notation, got: %s", output)
	}
	if !contains(output, "1043910119") {
		t.Errorf("expected full integer id, got: %s", output)
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

func TestRenderJSON_Breadcrumbs(t *testing.T) {
	var buf bytes.Buffer
	data := []map[string]any{{"id": 1, "name": "Jane"}}
	opts := Options{Mode: ModeJSON, Breadcrumbs: []Breadcrumb{{Cmd: "wenmar vehicles show 13"}}}
	err := Render(&buf, data, "", nil, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !contains(output, `"breadcrumbs"`) {
		t.Error("expected breadcrumbs in JSON envelope")
	}
	if !contains(output, `"cmd": "wenmar vehicles show 13"`) {
		t.Error("expected cmd entry in breadcrumbs")
	}
}

func TestRenderJSON_NoBreadcrumbsWhenEmpty(t *testing.T) {
	var buf bytes.Buffer
	data := []map[string]any{{"id": 1}}
	err := Render(&buf, data, "", nil, Options{Mode: ModeJSON})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contains(buf.String(), "breadcrumbs") {
		t.Error("expected no breadcrumbs when none provided")
	}
}

func TestRender_AgentRawJSON(t *testing.T) {
	var buf bytes.Buffer
	data := []map[string]any{{"id": 1, "name": "Jane"}}
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

func TestParseMode(t *testing.T) {
	cases := []struct {
		name    string
		spec    ModeSpec
		want    Mode
		wantErr bool
	}{
		{name: "json", spec: ModeSpec{JSON: true}, want: ModeJSON},
		{name: "agent", spec: ModeSpec{Agent: true}, want: ModeAgent},
		{name: "jq", spec: ModeSpec{JQ: ".[]"}, want: ModeJQ},
		{name: "ids-only", spec: ModeSpec{IDsOnly: true}, want: ModeIDsOnly},
		{name: "count", spec: ModeSpec{Count: true}, want: ModeCount},
		{name: "styled forces default", spec: ModeSpec{Styled: true}, want: ModeDefault},
		{name: "json plus agent conflicts", spec: ModeSpec{JSON: true, Agent: true}, wantErr: true},
		{name: "json plus jq conflicts", spec: ModeSpec{JSON: true, JQ: ".[]"}, wantErr: true},
		{name: "agent plus ids-only conflicts", spec: ModeSpec{Agent: true, IDsOnly: true}, wantErr: true},
		{name: "styled plus json conflicts", spec: ModeSpec{Styled: true, JSON: true}, wantErr: true},
		{name: "styled plus ids-only conflicts", spec: ModeSpec{Styled: true, IDsOnly: true}, wantErr: true},
		{name: "jq plus ids-only conflicts", spec: ModeSpec{JQ: ".[]", IDsOnly: true}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseMode(tc.spec)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseMode(%+v) expected error, got mode %v", tc.spec, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMode(%+v): %v", tc.spec, err)
			}
			if got != tc.want {
				t.Errorf("ParseMode(%+v) = %v, want %v", tc.spec, got, tc.want)
			}
		})
	}
}

func TestParseModeAutoSwitch(t *testing.T) {
	// ParseMode must not auto-switch here: the pipe check happens inside
	// ParseMode only when nothing explicit is set. This test pins that the
	// explicit table mode survives even when piped (test stdout is not a
	// TTY in CI).
	got, err := ParseMode(ModeSpec{Styled: true})
	if err != nil {
		t.Fatal(err)
	}
	if got != ModeDefault {
		t.Errorf("explicit styled mode should win over pipe auto-switch, got %v", got)
	}
	// With nothing set and piped stdout, raw JSON is chosen.
	got, err = ParseMode(ModeSpec{})
	if err != nil {
		t.Fatal(err)
	}
	if got != ModeAgent {
		t.Errorf("piped stdout with no explicit mode should auto-switch to raw JSON (ModeAgent), got %v", got)
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
