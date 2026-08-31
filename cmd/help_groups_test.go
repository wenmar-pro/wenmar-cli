package cmd

import (
	"strings"
	"testing"
)

func TestHelpGroups(t *testing.T) {
	out, err := execute("--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, section := range []string{"Resources", "Session & Config", "Agents & Discovery", "Platform"} {
		if !strings.Contains(out, section) {
			t.Errorf("root help missing group %q; got:\n%s", section, out)
		}
	}
	// Resources section lists the resource commands.
	if !strings.Contains(out, "workorders") {
		t.Error("workorders should appear under Resources")
	}
	// No ungrouped orphans section with our commands in it.
	for _, orphan := range []string{"setup", "doctor", "completion", "tui", "watch"} {
		if orphanSectionContains(out, orphan) {
			t.Errorf("%s should be grouped, not in Additional Commands", orphan)
		}
	}
}

func orphanSectionContains(out, name string) bool {
	// cobra renders ungrouped commands under "Available Commands:" and
	// grouped ones under their group titles. Simplest robust check: the
	// command must appear BELOW a group title line, not under a bare
	// "Available Commands:" header. (Implementation: split output at the
	// first group title; anything before it must not contain the name.)
	idx := strings.Index(out, "Resources:")
	if idx < 0 {
		return false // no groups at all — the outer test already failed
	}
	return strings.Contains(out[:idx], "  "+name+" ")
}
