package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/agent"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	"github.com/wenmar-pro/wenmar-cli/internal/errors"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
)

var (
	tokenFlag      string
	baseURLFlag    string
	locationFlag   string
	outputFlag     string
	jsonFlag       bool
	agentFlag      bool
	quietFlag      bool
	jqFlag         string
	allowPartial   bool
	configPathFlag string
	debugFlag      bool
)

// currentDebugInfo is populated by newSDKClient() with the request context
// (token source, base URL) and by setRequest() with the method/path, so the
// error handler can print useful diagnostics on failure.
var currentDebugInfo *errors.DebugInfo

var (
	groupResources = &cobra.Group{ID: "resources", Title: "Resources"}
	groupSession   = &cobra.Group{ID: "session", Title: "Session & Config"}
	groupAgents    = &cobra.Group{ID: "agents", Title: "Agents & Discovery"}
	groupPlatform  = &cobra.Group{ID: "platform", Title: "Platform"}
)

var rootCmd = &cobra.Command{
	Use:   "wenmar",
	Short: "Wenmar Pro API CLI",
	Long: `A command-line interface for the Wenmar Pro automotive shop
management API.

Getting started:
  wenmar setup        Configure your API token (or export WENMAR_TOKEN)

Output:
  --output <mode>    table | md | json | agent | quiet | ids-only |
                      count | html | styled  (default: table on a
                      terminal, quiet when piped)
  Quick flags: --json --agent --quiet --jq   (see 'wenmar help output')

Topics:
  wenmar help output      All output modes and the envelope format
  wenmar help exit-codes  The stable 0-10 exit-code contract
  wenmar help auth        Token sources and auth methods
  wenmar help location    Location scoping
  wenmar help environment Environment variables
  wenmar help watch       The watch command
  wenmar help agent-help  Structured --agent --help for AI agents`,
	Version:                    version,
	SilenceUsage:               true,
	SilenceErrors:              true,
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
	rootCmd.AddGroup(groupResources, groupSession, groupAgents, groupPlatform)
	rootCmd.SetVersionTemplate(versionString() + "\n")
	rootCmd.PersistentFlags().StringVar(&tokenFlag, "token", "", "API bearer token (or set WENMAR_TOKEN env)")
	rootCmd.PersistentFlags().StringVar(&baseURLFlag, "base-url", "", "API base URL (default: https://app.wenmarpro.com)")
	rootCmd.PersistentFlags().StringVar(&locationFlag, "location", "", "Location ID to scope requests (or set WENMAR_LOCATION_ID)")
	rootCmd.PersistentFlags().StringVar(&outputFlag, "output", "", "Output mode: table, md, json, agent, quiet, ids-only, count, html, styled (see 'wenmar help output')")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Shorthand for --output json")
	rootCmd.PersistentFlags().BoolVar(&agentFlag, "agent", false, "Shorthand for --output agent (also makes --help emit JSON)")
	rootCmd.PersistentFlags().BoolVar(&quietFlag, "quiet", false, "Shorthand for --output quiet")
	rootCmd.PersistentFlags().StringVar(&jqFlag, "jq", "", "jq filter expression (shorthand for --output json plus a filter)")
	rootCmd.PersistentFlags().BoolVar(&allowPartial, "allow-partial", false, "Accept truncated responses (adds a notice to the envelope)")
	rootCmd.PersistentFlags().StringVar(&configPathFlag, "config-path", "", "Path to config file")
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Print request debug info (token source, base URL, method/path) to stderr")

	// Sugar flags work everywhere but stay out of leaf help; the root
	// Long and `wenmar help output` are their discovery surfaces.
	rootCmd.PersistentFlags().MarkHidden("json")
	rootCmd.PersistentFlags().MarkHidden("agent")
	rootCmd.PersistentFlags().MarkHidden("quiet")
	rootCmd.PersistentFlags().MarkHidden("jq")
	rootCmd.PersistentFlags().MarkHidden("config-path")

	// Fail fast on mode conflicts/typos BEFORE any command runs (and
	// before any API call inside it).
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		_, err := output.ParseMode(modeSpec())
		return err
	}

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
