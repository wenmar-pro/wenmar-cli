package cmd

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/auth"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
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

		selectedID, selectedName, err := auth.ResolveAndSaveLocationID(cmd.Context(), client, configPath, locationFlag, !isTerminal(), cmd.OutOrStdout(), cmd.InOrStdin())
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

// isTerminal reports whether both stdin and stdout are attached to a terminal.
// Used to decide whether interactive prompts are safe.
func isTerminal() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
}
