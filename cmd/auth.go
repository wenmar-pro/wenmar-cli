package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/auth"
	oauthflow "github.com/wenmar-pro/wenmar-cli/internal/auth/oauth"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	authpkg "github.com/wenmar-pro/wenmar-sdk/go/pkg/auth"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in via OAuth browser flow (or --token for static token)",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, err := config.ConfigPath()
		if err != nil {
			return err
		}
		return runAuthLogin(cmd.OutOrStdout(), configPath)
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

var authTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Print the bearer token to stdout (for scripts)",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, err := config.ConfigPath()
		if err != nil {
			return err
		}
		return runAuthToken(cmd.OutOrStdout(), configPath)
	},
}

var authRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh the stored token (OAuth only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, err := config.ConfigPath()
		if err != nil {
			return err
		}
		return runAuthRefresh(cmd.OutOrStdout(), configPath)
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

func init() {
	authCmd.AddCommand(authLoginCmd, authStatusCmd, authTokenCmd, authRefreshCmd, authLogoutCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthLogin(out io.Writer, configPath string) error {
	// Static token path: --token flag provided (backward compat for CI/agents)
	if tokenFlag != "" {
		return storeStaticToken(tokenFlag, configPath, out)
	}

	// OAuth flow: no --token flag
	baseURL := auth.ResolveBaseURLFrom(baseURLFlag, configPath)
	fmt.Fprintf(out, "  Opening browser to log in to %s...\n", baseURL)

	token, err := oauthflow.Login(context.Background(), baseURL)
	if err != nil {
		return fmt.Errorf("OAuth login failed: %w", err)
	}

	// Store token (keyring with file fallback)
	store := authpkg.NewCredentialStore()
	if err := store.SaveToken(context.Background(), token); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	// Save config
	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		cfg = &config.Config{}
	}
	cfg.BaseURL = baseURL
	cfg.AuthMethod = "oauth"
	if err := config.SaveTo(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintln(out, "  Logged in successfully.")
	return nil
}

func storeStaticToken(token, configPath string, out io.Writer) error {
	store := authpkg.NewCredentialStore()
	if err := store.SaveToken(context.Background(), &authpkg.Token{AccessToken: token}); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	baseURL := auth.ResolveBaseURLFrom(baseURLFlag, configPath)
	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		cfg = &config.Config{}
	}
	cfg.BaseURL = baseURL
	cfg.AuthMethod = "static"
	if err := config.SaveTo(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintln(out, "  Token stored.")
	return nil
}

func runAuthLogout(configPath string) error {
	store := authpkg.NewCredentialStore()
	_ = store.DeleteToken(context.Background())
	if err := config.DeleteFrom(configPath); err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}
	fmt.Println("  Logged out.")
	return nil
}

func runAuthStatus(out io.Writer, configPath string) error {
	rt, err := auth.ResolveTokenWithSource(tokenFlag, configPath)
	if err != nil {
		fmt.Fprintln(out, "  Not logged in. Run `wenmar auth login` to configure.")
		os.Exit(2)
	}

	baseURL := auth.ResolveBaseURLFrom(baseURLFlag, configPath)
	fmt.Fprintf(out, "  Token:      %s  (from: %s)\n", maskToken(rt.Token), rt.Source)
	fmt.Fprintf(out, "  Base URL:   %s\n", baseURL)
	fmt.Fprint(out, "  Testing connection...")

	wcfg := wenmar.DefaultConfig()
	wcfg.BaseURL = baseURL
	client, err := wenmar.NewClient(wcfg, wenmar.NewStaticTokenProvider(rt.Token))
	if err != nil {
		fmt.Fprintln(out, " ✗")
		return err
	}

	_, err = client.ListAccount(context.Background())
	if err != nil {
		fmt.Fprintln(out, " ✗")
		fmt.Fprintf(out, "  Connection failed: %v\n", err)
		return err
	}

	fmt.Fprintln(out, " ✓")
	fmt.Fprintln(out, "  Connected.")
	return nil
}

func runAuthToken(out io.Writer, configPath string) error {
	rt, err := auth.ResolveTokenWithSource(tokenFlag, configPath)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, rt.Token)
	return nil
}

func runAuthRefresh(out io.Writer, configPath string) error {
	store := authpkg.NewCredentialStore()
	manager, err := auth.ResolveAuthManager(tokenFlag, configPath)
	if err != nil {
		fmt.Fprintln(out, "  Not logged in. Run `wenmar auth login` to authenticate.")
		return nil
	}
	_ = manager // ResolveAuthManager already wired the refresh function

	tok, err := store.GetToken(context.Background())
	if err != nil || tok == nil || tok.RefreshToken == "" {
		fmt.Fprintln(out, "  No OAuth token to refresh. Using static token — re-run `wenmar auth login` to get a new token.")
		return nil
	}

	if err := manager.Refresh(context.Background()); err != nil {
		if errors.Is(err, authpkg.ErrOAuthNotImplemented) {
			fmt.Fprintln(out, "  OAuth token refresh is not yet implemented. Re-run `wenmar auth login` to get a new token.")
			return nil
		}
		return fmt.Errorf("token refresh failed: %w", err)
	}

	fmt.Fprintln(out, "  Token refreshed.")
	return nil
}
