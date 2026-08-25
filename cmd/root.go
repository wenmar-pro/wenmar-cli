package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	tokenFlag   string
	baseURLFlag string
	mdFlag      bool
	jsonFlag    bool
	agentFlag   bool
	jqFlag      string
)

var rootCmd = &cobra.Command{
	Use:   "wenmar",
	Short: "Wenmar Pro API CLI",
	Long:  "A command-line interface for the Wenmar Pro automotive shop management API.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&tokenFlag, "token", "", "API bearer token (or set WENMAR_TOKEN env)")
	rootCmd.PersistentFlags().StringVar(&baseURLFlag, "base-url", "", "API base URL (default: https://app.wenmarpro.com)")
	rootCmd.PersistentFlags().BoolVarP(&mdFlag, "md", "m", false, "Output as GFM table (default for TTY)")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output as full JSON envelope {ok, data, summary, meta}")
	rootCmd.PersistentFlags().BoolVar(&agentFlag, "agent", false, "Output raw JSON data (no envelope)")
	rootCmd.PersistentFlags().StringVar(&jqFlag, "jq", "", "jq filter expression (implies --json)")
}
