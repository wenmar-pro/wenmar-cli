package agent

import (
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
)

func TestBuildCatalog_HasCommands(t *testing.T) {
	root := &cobra.Command{Use: "wenmar"}
	customers := &cobra.Command{Use: "customers", Short: "Manage customers"}
	list := &cobra.Command{Use: "list", Short: "List customers", Aliases: []string{"ls"}, Run: func(cmd *cobra.Command, args []string) {}}
	customers.AddCommand(list)
	root.AddCommand(customers)

	catalog := BuildCatalog(root)
	if len(catalog.Commands) == 0 {
		t.Fatal("expected commands in catalog")
	}

	var found bool
	for _, c := range catalog.Commands {
		if c.Path == "customers list" {
			found = true
			if c.Description != "List customers" {
				t.Errorf("expected description 'List customers', got '%s'", c.Description)
			}
			if len(c.Aliases) != 1 || c.Aliases[0] != "ls" {
				t.Errorf("expected alias 'ls', got %v", c.Aliases)
			}
		}
	}
	if !found {
		t.Error("expected 'customers list' in catalog")
	}
}

func TestBuildCatalog_JSONRoundTrip(t *testing.T) {
	root := &cobra.Command{Use: "wenmar"}
	root.AddCommand(&cobra.Command{Use: "test", Short: "Test cmd", Run: func(cmd *cobra.Command, args []string) {}})

	catalog := BuildCatalog(root)
	b, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("failed to marshal catalog: %v", err)
	}

	var parsed Catalog
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("failed to unmarshal catalog: %v", err)
	}
	if len(parsed.Commands) != len(catalog.Commands) {
		t.Error("round-trip mismatch")
	}
}

func TestBuildCatalog_PopulatesArgs(t *testing.T) {
	root := &cobra.Command{Use: "wenmar"}
	show := &cobra.Command{
		Use:    "show <id>",
		Short:  "Show a resource",
		Args:   cobra.ExactArgs(1),
		RunE:   func(cmd *cobra.Command, args []string) error { return nil },
	}
	res := &cobra.Command{Use: "customers", Short: "Manage customers"}
	res.AddCommand(show)
	root.AddCommand(res)

	catalog := BuildCatalog(root)

	// Find the "customers show" command
	var found *CommandInfo
	for _, c := range catalog.Commands {
		if c.Path == "customers show" {
			found = &c
			break
		}
	}
	if found == nil {
		t.Fatal("expected to find 'customers show' in catalog")
	}
	if len(found.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(found.Args))
	}
	if found.Args[0].Name != "id" {
		t.Errorf("expected arg name 'id', got %q", found.Args[0].Name)
	}
	if !found.Args[0].Required {
		t.Error("expected arg to be required")
	}
}
