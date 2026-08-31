package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
	authpkg "github.com/wenmar-pro/wenmar-sdk/go/pkg/auth"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run diagnostics: auth, connectivity, config, completion, skill",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

type DoctorResult struct {
	Check  string `json:"check"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func runDoctor(cmd *cobra.Command, args []string) error {
	path, err := resolveConfigPath()
	if err != nil {
		return err
	}

	var results []DoctorResult
	allOK := true

	// Check 1: Config file exists and is readable
	cfg, cfgErr := config.LoadFrom(path)
	if cfgErr != nil {
		results = append(results, DoctorResult{"config", "fail", "No config file found at " + path})
		allOK = false
	} else {
		results = append(results, DoctorResult{"config", "ok", path})
	}

	// Check 2: Token present (keyring or file or config)
	tokenPresent := false
	tokenSource := ""
	store := newCredentialStore()
	if tok, err := store.GetToken(context.Background()); err == nil && tok != nil && tok.AccessToken != "" {
		tokenPresent = true
		tokenSource = "keyring/file"
	} else if cfgErr == nil && cfg.Token != "" {
		tokenPresent = true
		tokenSource = "config file"
	}
	if tokenPresent {
		results = append(results, DoctorResult{"token", "ok", "present (" + tokenSource + ")"})
	} else {
		results = append(results, DoctorResult{"token", "fail", "No token configured. Run 'wenmar setup'."})
		allOK = false
	}

	// Check 3: auth_method
	authMethod := "static"
	if cfgErr == nil && cfg.AuthMethod != "" {
		authMethod = cfg.AuthMethod
	}
	results = append(results, DoctorResult{"auth_method", "ok", authMethod})

	// Check 4: keyring accessible
	if _, err := store.GetToken(context.Background()); err == nil {
		results = append(results, DoctorResult{"keyring", "ok", "accessible"})
	} else {
		results = append(results, DoctorResult{"keyring", "warn", "keyring unavailable; using file fallback"})
	}

	// Check 5: connectivity
	baseURL := "https://app.wenmarpro.com"
	if cfgErr == nil && cfg != nil && cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}
	if tokenPresent {
		wcfg := wenmar.DefaultConfig()
		wcfg.BaseURL = baseURL
		client, err := wenmar.NewClient(wcfg, wenmar.NewStaticTokenProvider(resolveDoctorToken(store, cfg)))
		if err != nil {
			results = append(results, DoctorResult{"connectivity", "fail", fmt.Sprintf("Client error: %v", err)})
			allOK = false
		} else {
			_, err := client.ListAccount(context.Background())
			if err != nil {
				results = append(results, DoctorResult{"connectivity", "fail", fmt.Sprintf("API error: %v", err)})
				allOK = false
			} else {
				results = append(results, DoctorResult{"connectivity", "ok", baseURL})
			}
		}
	} else {
		results = append(results, DoctorResult{"connectivity", "skip", "No token — skipped"})
	}

	// Check 6: Shell completion
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return fmt.Errorf("could not determine home directory: %w", homeErr)
	}
	completionFound := false
	completionPath := ""
	for _, p := range []string{
		home + "/.local/share/bash-completion/completions/wenmar",
		home + "/.config/fish/completions/wenmar.fish",
	} {
		if _, err := os.Stat(p); err == nil {
			completionFound = true
			completionPath = p
			break
		}
	}
	if completionFound {
		results = append(results, DoctorResult{"completion", "ok", completionPath})
	} else {
		results = append(results, DoctorResult{"completion", "warn", "Not found. Run 'wenmar completion <shell>' to install."})
	}

	// Check 7: Skill installed
	skillPath := filepath.Join(home, ".agents", "skills", "wenmar", "SKILL.md")
	if _, err := os.Stat(skillPath); err == nil {
		results = append(results, DoctorResult{"skill", "ok", skillPath})
	} else {
		results = append(results, DoctorResult{"skill", "warn", "Not found. Run 'wenmar skill install'."})
	}

	// Render
	mode, err := resolveMode()
	if err != nil {
		return err
	}
	if mode == output.ModeJSON {
		envelope := map[string]any{
			"ok":     allOK,
			"checks": results,
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(envelope)
	}

	// Human-readable
	for _, r := range results {
		icon := "✓"
		if r.Status == "fail" {
			icon = "✗"
		} else if r.Status == "warn" {
			icon = "⚠"
		} else if r.Status == "skip" {
			icon = "—"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %s  %-14s %s\n", icon, r.Check, r.Detail)
	}
	fmt.Fprintln(cmd.OutOrStdout())
	if allOK {
		fmt.Fprintln(cmd.OutOrStdout(), "  All checks passed.")
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "  Some checks failed. Run 'wenmar setup' to fix.")
	}

	if !allOK {
		return fmt.Errorf("doctor: some checks failed")
	}
	return nil
}

func resolveDoctorToken(store authpkg.CredentialStore, cfg *config.Config) string {
	if tok, err := store.GetToken(context.Background()); err == nil && tok != nil && tok.AccessToken != "" {
		return tok.AccessToken
	}
	if cfg != nil {
		return cfg.Token
	}
	return ""
}
