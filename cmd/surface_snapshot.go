package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var surfaceSnapshotCmd = &cobra.Command{
	Use:   "surface-snapshot",
	Short: "Dump the command tree as JSON (for CI diffing)",
	Run: func(cmd *cobra.Command, args []string) {
		snapshot := buildSurfaceSnapshot(rootCmd, "")
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(snapshot); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(surfaceSnapshotCmd)
}

// SurfaceCommand is a serializable representation of a cobra command.
type SurfaceCommand struct {
	Path     string           `json:"path"`
	Use      string           `json:"use"`
	Short    string           `json:"short"`
	Flags    []SurfaceFlag    `json:"flags"`
	Children []SurfaceCommand `json:"children,omitempty"`
}

// SurfaceFlag is a serializable representation of a cobra flag.
type SurfaceFlag struct {
	Name     string `json:"name"`
	Required bool   `json:"required,omitempty"`
	Default  string `json:"default,omitempty"`
}

func buildSurfaceSnapshot(cmd *cobra.Command, parentPath string) SurfaceCommand {
	path := cmd.Use
	if parentPath != "" {
		path = parentPath + " " + cmd.Use
	}

	surf := SurfaceCommand{
		Path:  path,
		Use:   cmd.Use,
		Short: cmd.Short,
	}

	// Collect flags.
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		surf.Flags = append(surf.Flags, SurfaceFlag{
			Name:     f.Name,
			Required: false,
			Default:  f.DefValue,
		})
	})
	// Mark required flags.
	for _, f := range surf.Flags {
		if cmd.Flags().Lookup(f.Name) != nil {
			f.Required = cmd.Flags().Lookup(f.Name).Annotations["cobra_annotation_bash_completion_one_required_flag"] != nil
		}
	}

	// Sort flags for stable output.
	sort.Slice(surf.Flags, func(i, j int) bool {
		return surf.Flags[i].Name < surf.Flags[j].Name
	})

	// Collect children.
	for _, child := range cmd.Commands() {
		if child.IsAvailableCommand() && !child.IsAdditionalHelpTopicCommand() {
			surf.Children = append(surf.Children, buildSurfaceSnapshot(child, path))
		}
	}
	sort.Slice(surf.Children, func(i, j int) bool {
		return surf.Children[i].Path < surf.Children[j].Path
	})

	return surf
}
