package auth

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wenmar-pro/wenmar-cli/internal/config"
)

const defaultBaseURL = "https://app.wenmarpro.com"

// TokenSource describes where a resolved API token came from.
type TokenSource string

const (
	SourceFlag   TokenSource = "--token flag"
	SourceEnv    TokenSource = "WENMAR_TOKEN env"
	SourceConfig TokenSource = "config file"
	SourceNone   TokenSource = "none"
)

// ResolvedToken pairs a token with the source it was resolved from.
type ResolvedToken struct {
	Token  string
	Source TokenSource
}

func ResolveToken(flagToken string) (string, error) {
	cfg, _ := config.Load()
	configPath := ""
	if cfg != nil {
		configPath, _ = config.ConfigPath()
	}
	return ResolveTokenFrom(flagToken, configPath)
}

func ResolveTokenFrom(flagToken, configPath string) (string, error) {
	rt, err := ResolveTokenWithSource(flagToken, configPath)
	if err != nil {
		return "", err
	}
	return rt.Token, nil
}

// ResolveTokenWithSource resolves a token and reports where it came from,
// so callers can surface useful diagnostics on auth failures.
func ResolveTokenWithSource(flagToken, configPath string) (ResolvedToken, error) {
	if flagToken != "" {
		return ResolvedToken{Token: flagToken, Source: SourceFlag}, nil
	}

	if envToken := os.Getenv("WENMAR_TOKEN"); envToken != "" {
		return ResolvedToken{Token: envToken, Source: SourceEnv}, nil
	}

	if configPath != "" {
		if cfg, err := config.LoadFrom(configPath); err == nil && cfg.Token != "" {
			return ResolvedToken{Token: cfg.Token, Source: SourceConfig}, nil
		}
	}

	return ResolvedToken{}, fmt.Errorf("API token required. Run `wenmar setup` to configure, or set --token / WENMAR_TOKEN env var")
}

func ResolveBaseURL(flagURL string) string {
	configPath, _ := config.ConfigPath()
	return ResolveBaseURLFrom(flagURL, configPath)
}

func ResolveBaseURLFrom(flagURL, configPath string) string {
	if flagURL != "" {
		return flagURL
	}

	if envURL := os.Getenv("WENMAR_URL"); envURL != "" {
		return envURL
	}

	if configPath != "" {
		if cfg, err := config.LoadFrom(configPath); err == nil && cfg.BaseURL != "" {
			return cfg.BaseURL
		}
	}

	// Check for per-repo .wenmar.yml (only base_url, only if trusted)
	if cwd, err := os.Getwd(); err == nil {
		repoPath := filepath.Join(cwd, ".wenmar.yml")
		if repoCfg, err := config.LoadRepoConfig(repoPath); err == nil && repoCfg.BaseURL != "" {
			return repoCfg.BaseURL
		}
	}

	return defaultBaseURL
}
