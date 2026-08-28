package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run diagnostics: auth, connectivity, config, completion",
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

	// Check 2: Token present
	tokenPresent := cfgErr == nil && cfg.Token != ""
	if tokenPresent {
		results = append(results, DoctorResult{"token", "ok", maskToken(cfg.Token)})
	} else {
		results = append(results, DoctorResult{"token", "fail", "No token configured. Run 'wenmar setup'."})
		allOK = false
	}

	// Check 3: Base URL reachable + token valid
	if tokenPresent && cfg != nil {
		client, err := wenmar.NewClient(cfg.BaseURL, cfg.Token)
		if err != nil {
			results = append(results, DoctorResult{"connectivity", "fail", fmt.Sprintf("Client error: %v", err)})
			allOK = false
		} else {
			_, err := client.ListAccount(context.Background())
			if err != nil {
				results = append(results, DoctorResult{"connectivity", "fail", fmt.Sprintf("API error: %v", err)})
				allOK = false
			} else {
				results = append(results, DoctorResult{"connectivity", "ok", cfg.BaseURL})
			}
		}
	} else {
		results = append(results, DoctorResult{"connectivity", "skip", "No token — skipped"})
	}

	// Check 4: Shell completion (check if a known completion file exists)
	home, _ := os.UserHomeDir()
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

	// Render
	if jsonFlag {
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
