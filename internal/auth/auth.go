package auth

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wenmar-pro/wenmar-cli/internal/config"
)

const defaultBaseURL = "https://app.wenmarpro.com"

func ResolveToken(flagToken string) (string, error) {
	cfg, _ := config.Load()
	configPath := ""
	if cfg != nil {
		configPath, _ = config.ConfigPath()
	}
	return ResolveTokenFrom(flagToken, configPath)
}

func ResolveTokenFrom(flagToken, configPath string) (string, error) {
	if flagToken != "" {
		return flagToken, nil
	}

	if envToken := os.Getenv("WENMAR_TOKEN"); envToken != "" {
		return envToken, nil
	}

	if configPath != "" {
		if cfg, err := config.LoadFrom(configPath); err == nil && cfg.Token != "" {
			return cfg.Token, nil
		}
	}

	return "", fmt.Errorf("API token required. Run `wenmar setup` to configure, or set --token / WENMAR_TOKEN env var")
}

func ResolveBaseURL(flagURL string) string {
	configPath, _ := config.ConfigPath()
	return ResolveBaseURLFrom(flagURL, configPath)
}

func ResolveBaseURLFrom(flagURL, configPath string) string {
	if flagURL != "" {
		return flagURL
	}

	if envURL := os.Getenv("WENMAR_BASE_URL"); envURL != "" {
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
