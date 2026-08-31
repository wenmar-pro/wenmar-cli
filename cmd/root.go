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
	jsonFlag       bool
	agentFlag      bool
	jqFlag         string
	idsOnlyFlag    bool
	styledFlag     bool
	countFlag      bool
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
  --json              Full JSON envelope {ok, data, summary, meta}
  --agent             Raw JSON (no envelope); --agent --help emits JSON
  --jq <expr>         Filter output with a jq expression
  --ids-only          One ID per line; --styled forces tables when piped
  --count             Bare integer count (for monitoring)
  (default: table on a terminal, raw JSON when piped)

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
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output as full JSON envelope {ok, data, summary, meta}")
	rootCmd.PersistentFlags().BoolVar(&agentFlag, "agent", false, "Output raw JSON data, no envelope (with --help: structured JSON)")
	rootCmd.PersistentFlags().StringVar(&jqFlag, "jq", "", "Filter output with a jq expression")
	rootCmd.PersistentFlags().BoolVar(&idsOnlyFlag, "ids-only", false, "Print one ID per line (for shell loops)")
	rootCmd.PersistentFlags().BoolVar(&styledFlag, "styled", false, "Force the human table even when piped")
	rootCmd.PersistentFlags().BoolVar(&countFlag, "count", false, "Print a bare integer count (for monitoring)")
	rootCmd.PersistentFlags().BoolVar(&allowPartial, "allow-partial", false, "Accept truncated responses (adds a notice to the envelope)")
	rootCmd.PersistentFlags().StringVar(&configPathFlag, "config-path", "", "Path to config file")
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Print request debug info (token source, base URL, method/path) to stderr")

	// --config-path stays hidden (it's for testing); the output-mode flags
	// are all visible at every help level.
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
