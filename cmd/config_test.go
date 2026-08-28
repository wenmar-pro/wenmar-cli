package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/config"
)

func TestConfigGet(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "test-token", BaseURL: "https://test.example.com"})

	out, err := executeWithConfig(configPath, "config", "get", "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "test-token" {
		t.Errorf("expected 'test-token', got %q", out)
	}
}

func TestConfigSet(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "old", BaseURL: "https://old.example.com"})

	_, err := executeWithConfig(configPath, "config", "set", "base_url", "https://new.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, _ := config.LoadFrom(configPath)
	if cfg.BaseURL != "https://new.example.com" {
		t.Errorf("expected base_url 'https://new.example.com', got '%s'", cfg.BaseURL)
	}
}

func TestConfigList(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "my-token", BaseURL: "https://test.example.com"})

	out, err := executeWithConfig(configPath, "config", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "token") {
		t.Errorf("expected 'token' in output, got %s", out)
	}
	if !strings.Contains(out, "base_url") {
		t.Errorf("expected 'base_url' in output, got %s", out)
	}
}

func TestConfigPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	out, err := executeWithConfig(configPath, "config", "path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != configPath {
		t.Errorf("expected %s, got %s", configPath, strings.TrimSpace(out))
	}
}

// executeWithConfig runs a command with the config file pointed at configPath.
func executeWithConfig(configPath string, args ...string) (string, error) {
	return execute(append(args, "--config-path", configPath)...)
}
