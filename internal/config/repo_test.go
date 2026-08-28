package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRepoConfig_UntrustedByDefault(t *testing.T) {
	dir := t.TempDir()
	repoConfig := filepath.Join(dir, ".wenmar.yml")
	os.WriteFile(repoConfig, []byte("base_url: https://untrusted.example.com\n"), 0644)

	// No trusted repos file → repo config is untrusted
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	cfg, err := LoadRepoConfig(repoConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// base_url from untrusted repo config should NOT be applied
	if cfg.BaseURL != "" {
		t.Errorf("expected empty base_url from untrusted repo, got '%s'", cfg.BaseURL)
	}
}

func TestLoadRepoConfig_TrustedAppliesBaseURL(t *testing.T) {
	dir := t.TempDir()
	repoConfig := filepath.Join(dir, ".wenmar.yml")
	os.WriteFile(repoConfig, []byte("base_url: https://trusted.example.com\n"), 0644)

	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	// Mark this repo as trusted
	trustPath := filepath.Join(xdg, "wenmar", "trusted_repos")
	os.MkdirAll(filepath.Dir(trustPath), 0755)
	os.WriteFile(trustPath, []byte(dir+"\n"), 0600)

	cfg, err := LoadRepoConfig(repoConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BaseURL != "https://trusted.example.com" {
		t.Errorf("expected trusted base_url, got '%s'", cfg.BaseURL)
	}
}

func TestLoadRepoConfig_TokenNeverReadFromRepo(t *testing.T) {
	dir := t.TempDir()
	repoConfig := filepath.Join(dir, ".wenmar.yml")
	os.WriteFile(repoConfig, []byte("token: secret-from-repo\nbase_url: https://trusted.example.com\n"), 0644)

	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	// Mark as trusted
	trustPath := filepath.Join(xdg, "wenmar", "trusted_repos")
	os.MkdirAll(filepath.Dir(trustPath), 0755)
	os.WriteFile(trustPath, []byte(dir+"\n"), 0600)

	cfg, err := LoadRepoConfig(repoConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// base_url should be applied (trusted)
	if cfg.BaseURL != "https://trusted.example.com" {
		t.Errorf("expected trusted base_url, got '%s'", cfg.BaseURL)
	}
	// Token must NEVER be read from repo config
	if cfg.Token != "" {
		t.Errorf("token should never be read from repo config, got '%s'", cfg.Token)
	}
}

func TestTrustRepo(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	repoDir := "/home/user/my-project"
	err := TrustRepo(repoDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	trustPath := filepath.Join(xdg, "wenmar", "trusted_repos")
	data, err := os.ReadFile(trustPath)
	if err != nil {
		t.Fatalf("could not read trust file: %v", err)
	}
	if !strings.Contains(string(data), repoDir) {
		t.Errorf("expected %s in trust file, got %s", repoDir, string(data))
	}
}
