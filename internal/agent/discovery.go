package agent

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CommandInfo struct {
	Path        string     `json:"path"`
	Description string     `json:"description"`
	Aliases     []string   `json:"aliases,omitempty"`
	Args        []ArgInfo  `json:"args,omitempty"`
	Flags       []FlagInfo `json:"flags,omitempty"`
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

		// Only add leaf commands (commands with a Run function) to the catalog
		if sub.Run != nil || sub.RunE != nil {
			*commands = append(*commands, info)
		}

		// Recurse into subcommands
		buildCatalogRecursive(sub, path, commands)
	}
}
