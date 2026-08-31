package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// docInvocation is one wenmar command line extracted from documentation.
type docInvocation struct {
	Source string // "SKILL.md" | "README.md"
	Line   int
	Raw    string
	Path   []string // command path tokens (canonical or alias)
	Flags  []string // flag names used (without --)
}

var wenmarLineRe = regexp.MustCompile(`(?m)^\s*(?:\$ )?wenmar\s+([^` + "`" + `\n]+)`)

// extractInvocations parses every `wenmar ...` line from doc code fences.
func extractInvocations(t *testing.T, path string) []docInvocation {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	src := string(data)

	// Extract fenced code blocks only — prose mentions of wenmar flags
	// (like this test's own comments) must not be validated.
	var invocations []docInvocation
	inFence := false
	line := 0
	for _, l := range strings.Split(src, "\n") {
		line++
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inFence = !inFence
			continue
		}
		if !inFence {
			continue
		}
		m := wenmarLineRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		inv := docInvocation{Source: path, Line: line, Raw: m[0]}
		// Strip an inline comment so `# ...` never becomes path tokens.
		rest := m[1]
		if i := strings.Index(rest, "#"); i >= 0 {
			rest = rest[:i]
		}
		tokens := strings.Fields(rest)
		seenFlag := false
		for _, tok := range tokens {
			tok = strings.Trim(tok, `"`)
			if strings.HasPrefix(tok, "--") {
				inv.Flags = append(inv.Flags, strings.TrimLeft(tok, "-"))
				seenFlag = true
				continue
			}
			if seenFlag {
				continue // flag value — not a path token
			}
			if strings.HasPrefix(tok, "<") || strings.HasPrefix(tok, "$") {
				break // placeholder or env-var token — stop path building
			}
			if strings.ContainsAny(tok, "<>|") {
				break // placeholder syntax — stop path building
			}
			inv.Path = append(inv.Path, tok)
		}
		if len(inv.Path) > 0 {
			invocations = append(invocations, inv)
		}
	}
	return invocations
}

// resolveCommand walks the cobra tree along path (resolving aliases).
// Returns the deepest command found and any leftover path tokens. Walking
// stops at the first token that is not a subcommand — those tokens are
// positional args, returned as leftover for the caller to validate.
func resolveCommand(root *cobra.Command, path []string) (*cobra.Command, []string, error) {
	// `help` is cobra's built-in command, not a real subcommand in the tree.
	// It accepts any topic/command name as a positional arg, so treat it as
	// a leaf that accepts arbitrary args.
	if len(path) > 0 && path[0] == "help" {
		return root, nil, nil
	}
	cmd := root
	remaining := path
	for len(remaining) > 0 {
		next, _, err := cmd.Find([]string{remaining[0]})
		if err != nil || next == nil || next == cmd {
			return cmd, remaining, nil
		}
		cmd = next
		remaining = remaining[1:]
	}
	return cmd, remaining, nil
}

func TestDocsFreshness(t *testing.T) {
	docs := []string{
		"../skills/wenmar/SKILL.md",
		"../README.md",
	}
	for _, doc := range docs {
		t.Run(doc, func(t *testing.T) {
			invocations := extractInvocations(t, doc)
			if len(invocations) < 10 {
				t.Fatalf("suspiciously few invocations parsed (%d) — parser regression?", len(invocations))
			}
			for _, inv := range invocations {
				inv := inv
				t.Run(fmt.Sprintf("L%d:%s", inv.Line, strings.Join(inv.Path, " ")), func(t *testing.T) {
					cmd, leftover, err := resolveCommand(rootCmd, inv.Path)
					if err != nil {
						t.Errorf("%s:%d: %q — %v (docs drifted from the CLI)", inv.Source, inv.Line, inv.Raw, err)
						return
					}
					// Leftover tokens are positional args — allowed only if
					// the command expects them (a non-nil Args validator).
					if len(leftover) > 0 && cmd.Args == nil {
						t.Errorf("%s:%d: %q passes %d positional args to %q which accepts none",
							inv.Source, inv.Line, inv.Raw, len(leftover), cmd.CommandPath())
					}
					// Every flag must exist on the command (or be a global).
					for _, flag := range inv.Flags {
						if flag == "help" {
							continue // cobra's built-in help flag on every command
						}
						if cmd.Flags().Lookup(flag) != nil {
							continue
						}
						if rootCmd.PersistentFlags().Lookup(flag) != nil {
							continue
						}
						t.Errorf("%s:%d: %q — flag --%s does not exist on %q",
							inv.Source, inv.Line, inv.Raw, flag, cmd.CommandPath())
					}
				})
			}
		})
	}
}
