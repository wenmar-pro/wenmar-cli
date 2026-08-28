package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure the Wenmar CLI with your API token",
	Long:  "Interactive setup wizard. Prompts for your API token, tests it, and saves configuration to ~/.wenmar/config.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetupCmd(cmd.InOrStdin(), cmd.OutOrStdout())
	},
}

func init() {
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
		fmt.Fprintf(out, "  Existing config found (token: %s)\n", maskToken(cfg.Token))
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
	client, err := wenmar.NewClient(baseURL, token)
	if err != nil {
		fmt.Fprintln(out, " ✗")
		return fmt.Errorf("failed to create client: %w", err)
	}

	_, err = client.ListCustomers(context.Background())
	if err != nil {
		fmt.Fprintln(out, " ✗")
		return fmt.Errorf("token verification failed: %w", err)
	}

	fmt.Fprintln(out, " ✓")
	fmt.Fprintf(out, "  Connected successfully to %s\n", baseURL)

	cfg := &config.Config{Token: token, BaseURL: baseURL}
	if err := config.SaveTo(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(out, "\n  Config saved to %s\n", configPath)
	fmt.Fprintln(out, "\n  Next steps:")
	fmt.Fprintln(out, "    wenmar customers list --md")
	fmt.Fprintln(out, "    wenmar --help")
	fmt.Fprintln(out, "")

	return nil
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return strings.Repeat("*", len(token))
	}
	return token[:4] + "..." + token[len(token)-4:]
}
