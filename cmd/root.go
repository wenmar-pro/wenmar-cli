package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/agent"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	"github.com/wenmar-pro/wenmar-cli/internal/errors"
)

var (
	tokenFlag      string
	baseURLFlag    string
	locationFlag   string
	mdFlag         bool
	jsonFlag       bool
	agentFlag      bool
	quietFlag      bool
	jqFlag         string
	idsOnlyFlag    bool
	countFlag      bool
	htmlFlag       bool
	styledFlag     bool
	allowPartial   bool
	configPathFlag string
	debugFlag      bool
)

// currentDebugInfo is populated by newSDKClient() with the request context
// (token source, base URL) and by setRequest() with the method/path, so the
// error handler can print useful diagnostics on failure.
var currentDebugInfo *errors.DebugInfo

var rootCmd = &cobra.Command{
	Use:           "wenmar",
	Short:         "Wenmar Pro API CLI",
	Long:          "A command-line interface for the Wenmar Pro automotive shop management API.",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
	SuggestionsMinimumDistance: 2,
}

func Execute() {
	// If no config and no env token, and running bare `wenmar`, show welcome
	if len(os.Args) == 1 {
		if envToken := os.Getenv("WENMAR_TOKEN"); envToken == "" {
			if cfg, err := config.Load(); err != nil || cfg.Token == "" {
				fmt.Println("  Welcome to Wenmar CLI")
				fmt.Println()
				fmt.Println("  No API token configured. Run `wenmar setup` to get started.")
				fmt.Println()
				fmt.Println("    wenmar setup")
				fmt.Println()
				os.Exit(0)
			}
		}
	}

	if err := rootCmd.Execute(); err != nil {
		errors.PrintError(os.Stderr, err, currentDebugInfo)
		os.Exit(errors.ExitCode(err))
	}
}

// RootCmd exposes the command tree for tests.
func RootCmd() *cobra.Command { return rootCmd }

func init() {
	rootCmd.SetVersionTemplate(versionString() + "\n")
	rootCmd.PersistentFlags().StringVar(&tokenFlag, "token", "", "API bearer token (or set WENMAR_TOKEN env)")
	rootCmd.PersistentFlags().StringVar(&baseURLFlag, "base-url", "", "API base URL (default: https://app.wenmarpro.com)")
	rootCmd.PersistentFlags().StringVar(&locationFlag, "location", "", "Location ID to scope requests (or set WENMAR_LOCATION_ID)")
	rootCmd.PersistentFlags().BoolVarP(&mdFlag, "md", "m", false, "Output as GFM table")
	rootCmd.PersistentFlags().BoolVar(&mdFlag, "markdown", false, "Output as GFM table (alias for --md)")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output as full JSON envelope {ok, data, summary, meta}")
	rootCmd.PersistentFlags().BoolVar(&agentFlag, "agent", false, "Output raw JSON data (no envelope)")
	rootCmd.PersistentFlags().BoolVar(&quietFlag, "quiet", false, "Raw JSON output, no envelope, no agent discovery")
	rootCmd.PersistentFlags().StringVar(&jqFlag, "jq", "", "jq filter expression (implies --json)")
	rootCmd.PersistentFlags().BoolVar(&idsOnlyFlag, "ids-only", false, "Print one ID per line (for shell loops)")
	rootCmd.PersistentFlags().BoolVar(&countFlag, "count", false, "Print the count of results (bare integer)")
	rootCmd.PersistentFlags().BoolVar(&htmlFlag, "html", false, "Output as an HTML document")
	rootCmd.PersistentFlags().BoolVar(&styledFlag, "styled", false, "Force human tables even when piped")
	rootCmd.PersistentFlags().BoolVar(&allowPartial, "allow-partial", false, "Accept truncated responses (adds a notice to the envelope)")
	rootCmd.PersistentFlags().StringVar(&configPathFlag, "config-path", "", "Path to config file (for testing)")
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Print request debug info (token source, base URL, method/path) to stderr")

	// When --debug is set, surface the request context on success too.
	rootCmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		if debugFlag && currentDebugInfo != nil {
			fmt.Fprintln(os.Stderr, "DEBUG:")
			if currentDebugInfo.TokenMasked != "" {
				fmt.Fprintf(os.Stderr, "  token:    %s  (%s)\n", currentDebugInfo.TokenMasked, currentDebugInfo.TokenSource)
			}
			if currentDebugInfo.BaseURL != "" {
				fmt.Fprintf(os.Stderr, "  base URL: %s\n", currentDebugInfo.BaseURL)
			}
			if currentDebugInfo.Method != "" && currentDebugInfo.Path != "" {
				fmt.Fprintf(os.Stderr, "  request:  %s %s\n", currentDebugInfo.Method, currentDebugInfo.Path)
			}
		}
		return nil
	}

	defaultHelpFunc := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if agentFlag {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			if err := enc.Encode(agent.BuildCommandInfo(rootCmd, cmd)); err != nil {
				fmt.Fprintln(os.Stderr, "wenmar: help:", err)
				os.Exit(1)
			}
			return
		}
		defaultHelpFunc(cmd, args)
	})
}
