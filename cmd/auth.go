package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Configure your API token (same as `wenmar setup`)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetupCmd(cmd.InOrStdin(), cmd.OutOrStdout())
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Delete your saved API token",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, err := config.ConfigPath()
		if err != nil {
			return err
		}
		return runAuthLogout(configPath)
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, err := config.ConfigPath()
		if err != nil {
			return err
		}
		return runAuthStatus(cmd.OutOrStdout(), configPath)
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd, authLogoutCmd, authStatusCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthLogout(configPath string) error {
	if err := config.DeleteFrom(configPath); err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}
	fmt.Println("  Logged out.")
	return nil
}

func runAuthStatus(out io.Writer, configPath string) error {
	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		fmt.Fprintln(out, "  Not logged in. Run `wenmar setup` to configure.")
		os.Exit(2)
	}

	fmt.Fprintf(out, "  Token:      %s\n", maskToken(cfg.Token))
	fmt.Fprintf(out, "  Base URL:   %s\n", cfg.BaseURL)
	fmt.Fprint(out, "  Testing connection...")

	client, err := wenmar.NewClient(cfg.BaseURL, cfg.Token)
	if err != nil {
		fmt.Fprintln(out, " ✗")
		return err
	}

	_, err = client.ListCustomers(context.Background(), nil)
	if err != nil {
		fmt.Fprintln(out, " ✗")
		fmt.Fprintf(out, "  Connection failed: %v\n", err)
		return err
	}

	fmt.Fprintln(out, " ✓")
	fmt.Fprintln(out, "  Connected.")
	return nil
}
