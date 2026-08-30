package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/agent"
)

func TestAgentCatalog_RequiredFlags(t *testing.T) {
	cat := agent.BuildCatalog(rootCmd)
	var woCreate *agent.CommandInfo
	for i := range cat.Commands {
		if cat.Commands[i].Path == "work_orders create" {
			woCreate = &cat.Commands[i]
		}
	}
	if woCreate == nil {
		t.Fatal("work_orders create not in catalog")
	}
	for _, f := range woCreate.Flags {
		if f.Name == "customer-id" && !f.Required {
			t.Errorf("work_orders create --customer-id must be required:true in catalog")
		}
	}
}

func TestAgentSurfacesAgree(t *testing.T) {
	cat := agent.BuildCatalog(rootCmd)
	var catalogInfo *agent.CommandInfo
	for i := range cat.Commands {
		if cat.Commands[i].Path == "customers list" {
			catalogInfo = &cat.Commands[i]
		}
	}
	if catalogInfo == nil {
		t.Fatal("customers list not in catalog")
	}

	agentFlag = true
	defer func() { agentFlag = false }()

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"customers", "list", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var helpInfo agent.CommandInfo
	if err := json.Unmarshal(buf.Bytes(), &helpInfo); err != nil {
		t.Fatalf("unmarshal help JSON: %v\noutput: %s", err, buf.String())
	}
	if helpInfo.Path != "wenmar customers list" {
		t.Errorf("help path = %q, want %q", helpInfo.Path, "wenmar customers list")
	}

	have := map[string]bool{}
	for _, f := range catalogInfo.Flags {
		have[f.Name] = true
	}
	for _, f := range helpInfo.Flags {
		if !have[f.Name] {
			t.Errorf("help surface lists flag %q missing from catalog", f.Name)
		}
	}
}
