package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/agent"
)

var commandsCmd = &cobra.Command{
	Use:     "commands",
	Short:   "List all commands as JSON (for agent discovery)",
	GroupID: "agents",
	Run: func(cmd *cobra.Command, args []string) {
		catalog := agent.BuildCatalog(rootCmd)
		for _, t := range helpTopics {
			catalog.AddTopic(t.name, t.title)
		}
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
