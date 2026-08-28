package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RepoConfig is the structure of .wenmar.yml files found in project directories.
// Only non-authority keys (currently none beyond base_url) are loaded from
// trusted repo configs. token is NEVER read from repo config.
type RepoConfig struct {
	BaseURL string `yaml:"base_url"`
	Token   string `yaml:"-"` // never populated — token is never read from repo config
}

// LoadRepoConfig loads a .wenmar.yml file from the given path. If the repo
// is not trusted (not listed in the trusted_repos file), authority keys
// (base_url) are stripped. token is never loaded from repo config regardless
// of trust status.
func LoadRepoConfig(path string) (*RepoConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RepoConfig{}, nil
		}
		return nil, err
	}

	var raw struct {
		BaseURL string `yaml:"base_url"`
		Token   string `yaml:"token"` // parsed but NEVER used
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	cfg := &RepoConfig{}

	// Only apply base_url if the repo is trusted
	repoDir := filepath.Dir(path)
	if isRepoTrusted(repoDir) {
		cfg.BaseURL = raw.BaseURL
	}
	// Token is never loaded from repo config — security

	return cfg, nil
}

// isRepoTrusted checks if the given directory is in the trusted_repos file.
func isRepoTrusted(repoDir string) bool {
	trustPath, err := trustedReposPath()
	if err != nil {
		return false
	}

	data, err := os.ReadFile(trustPath)
	if err != nil {
		return false
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == repoDir {
			return true
		}
	}
	return false
}

// TrustRepo adds the given directory to the trusted_repos file.
func TrustRepo(repoDir string) error {
	trustPath, err := trustedReposPath()
	if err != nil {
		return err
	}

	// Read existing entries
	entries := map[string]bool{}
	if data, err := os.ReadFile(trustPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			l := strings.TrimSpace(line)
			if l != "" {
				entries[l] = true
			}
		}
	}

	entries[repoDir] = true

	// Write back
	var sb strings.Builder
	for dir := range entries {
		sb.WriteString(dir + "\n")
	}

	os.MkdirAll(filepath.Dir(trustPath), 0755)
	return os.WriteFile(trustPath, []byte(sb.String()), 0600)
}

func trustedReposPath() (string, error) {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "wenmar", "trusted_repos"), nil
}
