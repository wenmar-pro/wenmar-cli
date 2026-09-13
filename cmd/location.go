package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/auth"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
)

var locationUseCmd = &cobra.Command{
	Use:   "use",
	Short: "Set the default location for API requests",
	Long:  "Interactively selects or directly sets the default X-Wenmar-Location header value used by other commands.",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := configPathFlag
		if configPath == "" {
			if p, err := config.ConfigPath(); err == nil {
				configPath = p
			}
		}

		client, err := newClient()
		if err != nil {
			return err
		}

		selectedID, selectedName, err := auth.ResolveAndSaveLocationID(cmd.Context(), client, configPath, locationFlag, !output.IsInteractive(), cmd.OutOrStdout(), cmd.InOrStdin())
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Set default location to %s (%s).\n", selectedName, selectedID)
		return nil
	},
}

var locationCmd = &cobra.Command{
	Use:   "location",
	Short: "Manage CLI location defaults",
}

func init() {
	locationCmd.AddCommand(locationUseCmd)
	rootCmd.AddCommand(locationCmd)
}
