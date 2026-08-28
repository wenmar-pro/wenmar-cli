package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wenmar-pro/wenmar-cli/internal/config"
	authpkg "github.com/wenmar-pro/wenmar-sdk/go/pkg/auth"
)

const defaultBaseURL = "https://app.wenmarpro.com"

// TokenSource describes where a resolved API token came from.
type TokenSource string

const (
	SourceFlag    TokenSource = "--token flag"
	SourceEnv     TokenSource = "WENMAR_TOKEN env"
	SourceKeyring TokenSource = "keyring"
	SourceConfig  TokenSource = "config file"
	SourceNone    TokenSource = "none"
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
	return ResolveTokenWithSourceFrom(flagToken, configPath, authpkg.NewCredentialStore())
}

// ResolveTokenWithSourceFrom is ResolveTokenWithSource with an injectable
// credential store (for tests).
func ResolveTokenWithSourceFrom(flagToken, configPath string, store authpkg.CredentialStore) (ResolvedToken, error) {
	if flagToken != "" {
		return ResolvedToken{Token: flagToken, Source: SourceFlag}, nil
	}

	if envToken := os.Getenv("WENMAR_TOKEN"); envToken != "" {
		return ResolvedToken{Token: envToken, Source: SourceEnv}, nil
	}

	// Keyring (via the SDK CredentialStore) before the config file.
	if tok, err := keyringTokenFrom(store); err == nil && tok != "" {
		return ResolvedToken{Token: tok, Source: SourceKeyring}, nil
	}

	if configPath != "" {
		if cfg, err := config.LoadFrom(configPath); err == nil && cfg.Token != "" {
			return ResolvedToken{Token: cfg.Token, Source: SourceConfig}, nil
		}
	}

	return ResolvedToken{}, fmt.Errorf("API token required. Run `wenmar setup` to configure, or set --token / WENMAR_TOKEN env var")
}

// keyringToken reads the token from the SDK credential store (keyring with
// file fallback). Returns an error if no token is stored.
func keyringToken() (string, error) {
	return keyringTokenFrom(authpkg.NewCredentialStore())
}

func keyringTokenFrom(store authpkg.CredentialStore) (string, error) {
	tok, err := store.GetToken(context.Background())
	if err != nil {
		return "", err
	}
	if tok == nil || tok.AccessToken == "" {
		return "", fmt.Errorf("no token stored")
	}
	return tok.AccessToken, nil
}

// ResolveAuthManager builds the full auth stack: keyring → file → env → flag.
// It returns an AuthManager whose provider resolves the token with the
// correct precedence. flagToken is the --token flag value.
func ResolveAuthManager(flagToken, configPath string) (*authpkg.AuthManager, error) {
	return ResolveAuthManagerWithStore(flagToken, configPath, authpkg.NewCredentialStore())
}

// ResolveAuthManagerWithStore is ResolveAuthManager with an injectable
// credential store (for tests).
func ResolveAuthManagerWithStore(flagToken, configPath string, store authpkg.CredentialStore) (*authpkg.AuthManager, error) {
	// Highest precedence: --token flag.
	if flagToken != "" {
		return authpkg.NewAuthManager(store, authpkg.NewStaticTokenProvider(flagToken)), nil
	}

	// WENMAR_TOKEN env.
	if envToken := os.Getenv("WENMAR_TOKEN"); envToken != "" {
		return authpkg.NewAuthManager(store, authpkg.NewStaticTokenProvider(envToken)), nil
	}

	// Keyring / file credential store with auto-refresh.
	if tok, err := store.GetToken(context.Background()); err == nil && tok != nil && tok.AccessToken != "" {
		manager := authpkg.NewAuthManager(store, nil)
		provider := &authpkg.CredentialStoreProvider{Store: store, Manager: manager}
		manager.Provider = provider
		return manager, nil
	}

	// Config file token (legacy).
	if configPath != "" {
		if cfg, err := config.LoadFrom(configPath); err == nil && cfg.Token != "" {
			return authpkg.NewAuthManager(store, authpkg.NewStaticTokenProvider(cfg.Token)), nil
		}
	}

	return nil, fmt.Errorf("API token required. Run `wenmar setup` to configure, or set --token / WENMAR_TOKEN env var")
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

// ResolveLocationID resolves the location scope: --location flag →
// WENMAR_LOCATION_ID env → config location_id.
func ResolveLocationID(flagLocation, configPath string) string {
	if flagLocation != "" {
		return flagLocation
	}
	if envLoc := os.Getenv("WENMAR_LOCATION_ID"); envLoc != "" {
		return envLoc
	}
	if configPath != "" {
		if cfg, err := config.LoadFrom(configPath); err == nil && cfg.LocationID != "" {
			return cfg.LocationID
		}
	}
	return ""
}
