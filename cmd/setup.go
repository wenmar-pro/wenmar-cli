package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/agent"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	"github.com/wenmar-pro/wenmar-cli/internal/errors"
	authpkg "github.com/wenmar-pro/wenmar-sdk/go/pkg/auth"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var (
	setupSkipAgents bool
	setupSilent     bool
)

var setupCmd = &cobra.Command{
	Use:     "setup [claude|codex]",
	Short:   "Configure the Wenmar CLI with your API token",
	Long:    "Interactive setup wizard. Prompts for your API token, tests it, and saves configuration. Optionally installs agent skills.",
	Args:    cobra.MaximumNArgs(1),
	GroupID: "session",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			switch args[0] {
			case "claude":
				return runSetupClaude(cmd.OutOrStdout())
			case "codex":
				return runSetupCodex(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unknown setup target %q (supported: claude, codex)", args[0])
			}
		}
		return runSetupCmd(cmd.InOrStdin(), cmd.OutOrStdout())
	},
}

func init() {
	setupCmd.Flags().BoolVar(&setupSkipAgents, "skip-agents", false, "Skip agent skill installation")
	setupCmd.Flags().BoolVar(&setupSilent, "silent-success", false, "Minimal output on success (for harnesses)")
	rootCmd.AddCommand(setupCmd)
}

func runSetupCmd(in io.Reader, out io.Writer) error {
	configPath, err := config.ConfigPath()
	if err != nil {
		return err
	}
	return runSetup(in, out, configPath, "")
}

func runSetup(in io.Reader, out io.Writer, configPath, baseURLOverride string) error {
	reader := bufio.NewReader(in)

	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "  Welcome to Wenmar CLI")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "  To use the CLI, you need an API token.")
	fmt.Fprintln(out, "  Create one at: https://app.wenmarpro.com/settings/api_tokens")
	fmt.Fprintln(out, "")

	if cfg, err := config.LoadFrom(configPath); err == nil && cfg.Token != "" {
		fmt.Fprintf(out, "  Existing config found (token: %s)\n", errors.MaskToken(cfg.Token))
		fmt.Fprint(out, "  Overwrite? (y/N): ")
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(line)) != "y" {
			fmt.Fprintln(out, "  Setup cancelled.")
			return nil
		}
	}

	fmt.Fprint(out, "  Enter your API token: ")
	tokenLine, _ := reader.ReadString('\n')
	token := strings.TrimSpace(tokenLine)
	if token == "" {
		return fmt.Errorf("token is required")
	}

	baseURL := baseURLOverride
	if baseURL == "" {
		fmt.Fprint(out, "  Base URL? (default: https://app.wenmarpro.com): ")
		baseLine, _ := reader.ReadString('\n')
		baseURL = strings.TrimSpace(baseLine)
		if baseURL == "" {
			baseURL = "https://app.wenmarpro.com"
		}
	}

	fmt.Fprint(out, "  Verifying token...")
	wcfg := wenmar.DefaultConfig()
	wcfg.BaseURL = baseURL
	client, err := wenmar.NewClient(wcfg, wenmar.NewStaticTokenProvider(token))
	if err != nil {
		fmt.Fprintln(out, " ✗")
		return fmt.Errorf("failed to create client: %w", err)
	}

	_, err = client.ListCustomers(context.Background(), nil)
	if err != nil {
		fmt.Fprintln(out, " ✗")
		return fmt.Errorf("token verification failed: %w", err)
	}

	fmt.Fprintln(out, " ✓")
	fmt.Fprintf(out, "  Connected successfully to %s\n", baseURL)

	// Store the token in the keyring (with file fallback), not the config file.
	store := newCredentialStore()
	if err := store.SaveToken(context.Background(), &authpkg.Token{AccessToken: token}); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	// Config file stores base_url and auth_method only.
	cfg := &config.Config{BaseURL: baseURL, AuthMethod: "static"}
	if err := config.SaveTo(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(out, "\n  Config saved to %s\n", configPath)

	if !setupSkipAgents {
		if err := installAgentSkill(out); err != nil {
			fmt.Fprintf(out, "  (skill install skipped: %v)\n", err)
		}
	}

	if setupSilent {
		fmt.Fprintln(out, "  Setup complete.")
		return nil
	}

	fmt.Fprintln(out, "\n  Next steps:")
	fmt.Fprintln(out, "    wenmar customers list --md")
	fmt.Fprintln(out, "    wenmar --help")
	fmt.Fprintln(out, "")

	return nil
}

// installAgentSkill installs the wenmar skill for the given agent.
func installAgentSkill(out io.Writer) error {
	source, err := bundledSkillDir()
	if err != nil {
		return err
	}
	target, err := agent.SkillDir()
	if err != nil {
		return err
	}
	if err := agent.InstallSkill(source, target, false); err != nil {
		return err
	}
	fmt.Fprintf(out, "  Installed agent skill to %s\n", target)
	return nil
}

func runSetupClaude(out io.Writer) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude")); err != nil {
		fmt.Fprintln(out, "  Claude Code not detected (~/.claude not found).")
	}
	if err := installAgentSkill(out); err != nil {
		return err
	}
	fmt.Fprintln(out, "  Claude Code setup complete.")
	return nil
}

func runSetupCodex(out io.Writer) error {
	if err := installAgentSkill(out); err != nil {
		return err
	}
	fmt.Fprintln(out, "  Codex setup complete.")
	return nil
}
