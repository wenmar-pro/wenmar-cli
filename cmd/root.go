package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wenmar-pro/wenmar-cli/internal/agent"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	"github.com/wenmar-pro/wenmar-cli/internal/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	Use:           "wenmar",
	Short:         "Wenmar Pro API CLI",
	Long:          "A command-line interface for the Wenmar Pro automotive shop management API.",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(errors.ExitCode(err))
	}
}

// RootCmd exposes the command tree for tests.
func RootCmd() *cobra.Command { return rootCmd }

func init() {
	rootCmd.SetVersionTemplate(versionString() + "\n")
	rootCmd.PersistentFlags().StringVar(&tokenFlag, "token", "", "API bearer token (or set WENMAR_TOKEN env)")
	rootCmd.PersistentFlags().StringVar(&baseURLFlag, "base-url", "", "API base URL (default: https://app.wenmarpro.com)")
	rootCmd.PersistentFlags().BoolVarP(&mdFlag, "md", "m", false, "Output as GFM table (default for TTY)")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output as full JSON envelope {ok, data, summary, meta}")
	rootCmd.PersistentFlags().BoolVar(&agentFlag, "agent", false, "Output raw JSON data (no envelope)")
	rootCmd.PersistentFlags().StringVar(&jqFlag, "jq", "", "jq filter expression (implies --json)")

	defaultHelpFunc := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if agentFlag {
			info := agent.CommandInfo{
				Path:        cmd.CommandPath(),
				Description: cmd.Short,
			}
			cmd.Flags().VisitAll(func(f *pflag.Flag) {
				if !f.Hidden {
					info.Flags = append(info.Flags, agent.FlagInfo{
						Name:        f.Name,
						Short:       f.Shorthand,
						Type:        f.Value.Type(),
						Default:     f.DefValue,
						Description: f.Usage,
					})
				}
			})
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(info)
			return
		}
		defaultHelpFunc(cmd, args)
	})
}
