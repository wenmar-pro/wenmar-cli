package cmd

import (
	"strings"
	"testing"
)

func TestExamplesOnKeyCommands(t *testing.T) {
	for _, args := range [][]string{
		{"customers", "list"},
		{"customers", "show"},
		{"workorders", "list"},
		{"workorders", "show"},
		{"vehicles", "list"},
		{"tags", "list"},
		{"servicecategories", "list"},
		{"account", "show"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			out, err := execute(append(args, "--help")...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(out, "Examples:") {
				t.Errorf("no Examples section in help for %v", args)
			}
		})
	}
}
