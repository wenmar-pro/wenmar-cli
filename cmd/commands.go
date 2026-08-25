package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wenmar-pro/wenmar-cli/internal/agent"
	"github.com/spf13/cobra"
)

var commandsCmd = &cobra.Command{
	Use:   "commands",
	Short: "List all commands as JSON (for agent discovery)",
	Run: func(cmd *cobra.Command, args []string) {
		catalog := agent.BuildCatalog(rootCmd)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(catalog); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(commandsCmd)
}
