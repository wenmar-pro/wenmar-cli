package agent

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CommandInfo struct {
	Path             string     `json:"path"`
	Description      string     `json:"description"`
	Aliases          []string   `json:"aliases,omitempty"`
	Args             []ArgInfo  `json:"args,omitempty"`
	Flags            []FlagInfo `json:"flags,omitempty"`
	Canonical        bool       `json:"canonical"`
	CompatibilityFor string     `json:"compatibility_for,omitempty"`
	Type             string     `json:"type,omitempty"` // "command" | "topic"
}

type ArgInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

type FlagInfo struct {
	Name        string `json:"name"`
	Short       string `json:"short,omitempty"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

type Catalog struct {
	Commands []CommandInfo `json:"commands"`
}

// AddTopic registers a help topic in the catalog.
func (c *Catalog) AddTopic(name, title string) {
	c.Commands = append(c.Commands, CommandInfo{
		Path:        "help " + name,
		Description: title,
		Canonical:   true,
		Type:        "topic",
	})
}

func BuildCatalog(root *cobra.Command) Catalog {
	var commands []CommandInfo
	buildCatalogRecursive(root, "", &commands)
	return Catalog{Commands: commands}
}

func buildCatalogRecursive(cmd *cobra.Command, parentPath string, commands *[]CommandInfo) {
	for _, sub := range cmd.Commands() {
		// Skip hidden commands and the help command
		if sub.Hidden || sub.Name() == "help" {
			continue
		}

		path := sub.Name()
		if parentPath != "" {
			path = parentPath + " " + path
		}

		info := CommandInfo{
			Path:        path,
			Description: sub.Short,
			Aliases:     sub.Aliases,
			Canonical:   true,
			Type:        "command",
		}

		// Extract flags
		sub.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Hidden {
				return
			}
			info.Flags = append(info.Flags, FlagInfo{
				Name:        f.Name,
				Short:       f.Shorthand,
				Type:        f.Value.Type(),
				Required:    f.Annotations != nil,
				Default:     f.DefValue,
				Description: f.Usage,
			})
		})

		// Extract positional args from the Use string (e.g. "show <id>" → [{Name: "id", Required: true}])
		info.Args = extractArgs(sub)

		// Only add leaf commands (commands with a Run function) to the catalog
		if sub.Run != nil || sub.RunE != nil {
			*commands = append(*commands, info)

			// Emit alias entries with compatibility_for pointing at the canonical name.
			// Aliases may be on this command or on an ancestor (e.g. `wo` for
			// `work_orders`), so combine ancestor aliases with the child path.
			aliasPaths := aliasPathsFor(parentPath, sub)
			for _, aliasPath := range aliasPaths {
				aliasInfo := CommandInfo{
					Path:             aliasPath,
					Description:      sub.Short,
					Canonical:        false,
					CompatibilityFor: path,
					Type:             "command",
				}
				*commands = append(*commands, aliasInfo)
			}
		}

		// Recurse into subcommands
		buildCatalogRecursive(sub, path, commands)
	}
}

// aliasPathsFor returns the alias paths for a leaf command, substituting each
// ancestor's name with its aliases. For example, `work_orders list` with
// `work_orders` aliased to `wo` yields `wo list`. Paths exclude the root
// command name, matching BuildCatalog's convention.
func aliasPathsFor(parentPath string, leaf *cobra.Command) []string {
	// Collect the chain of commands from root to leaf.
	var chain []*cobra.Command
	for c := leaf; c != nil; c = c.Parent() {
		chain = append([]*cobra.Command{c}, chain...)
	}
	// Drop the root command (its name is not part of catalog paths).
	if len(chain) > 0 {
		chain = chain[1:]
	}

	// Build alias paths by substituting each command's name with its aliases.
	paths := []string{""}
	for _, c := range chain {
		names := []string{c.Name()}
		names = append(names, c.Aliases...)
		var next []string
		for _, p := range paths {
			for _, n := range names {
				if p == "" {
					next = append(next, n)
				} else {
					next = append(next, p+" "+n)
				}
			}
		}
		paths = next
	}

	// Drop the canonical path (first entry) and any duplicates.
	canonical := leaf.CommandPath()
	// Strip the root command name from the canonical path.
	if root := leaf.Root(); root != nil && root.Name() != "" {
		canonical = strings.TrimPrefix(canonical, root.Name()+" ")
	}
	seen := map[string]bool{}
	var result []string
	for _, p := range paths {
		if p == canonical || seen[p] {
			continue
		}
		seen[p] = true
		result = append(result, p)
	}
	return result
}

// extractArgs parses positional arg names from a cobra command's Use string.
// Example: "show <id>" → [{Name: "id", Required: true}]
// Example: "list" → []
// Example: "show [filter]" → [{Name: "filter", Required: false}]
func extractArgs(cmd *cobra.Command) []ArgInfo {
	var args []ArgInfo
	fields := strings.Fields(cmd.Use)
	if len(fields) <= 1 {
		return args
	}
	// Skip the command name (first field), parse the rest
	for _, field := range fields[1:] {
		if !strings.HasPrefix(field, "<") && !strings.HasPrefix(field, "[") {
			continue
		}
		name := strings.Trim(field, "<>[]")
		required := strings.HasPrefix(field, "<")
		args = append(args, ArgInfo{
			Name:     name,
			Type:     "string",
			Required: required,
		})
	}
	return args
}
