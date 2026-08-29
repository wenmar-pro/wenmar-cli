package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/config"
	authpkg "github.com/wenmar-pro/wenmar-sdk/go/pkg/auth"
)

func TestRunAuthLogin_StaticTokenStillWorks(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	tokenFlag = "static-token-123"
	defer func() { tokenFlag = "" }()

	if err := runAuthLogin(&bytes.Buffer{}, configPath); err != nil {
		t.Fatalf("runAuthLogin with static token failed: %v", err)
	}

	// Verify the config recorded the static auth method. The token itself
	// goes to the real keyring (not testable in CI); config is the contract.
	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if cfg.AuthMethod != "static" {
		t.Errorf("AuthMethod = %q, want static", cfg.AuthMethod)
	}
}

func TestAuthLogin_StoresToken(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")

	oldToken := tokenFlag
	tokenFlag = "sk-test-1234"
	defer func() { tokenFlag = oldToken }()

	var output bytes.Buffer
	if err := runAuthLogin(&output, configPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store := authpkg.NewCredentialStore()
	tok, err := store.GetToken(context.Background())
	if err != nil {
		t.Fatalf("token not stored: %v", err)
	}
	if tok.AccessToken != "sk-test-1234" {
		t.Errorf("expected stored token 'sk-test-1234', got %q", tok.AccessToken)
	}
	_ = configPath
}

func TestAuthToken_PrintsToken(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")

	oldToken := tokenFlag
	tokenFlag = "sk-print-me"
	defer func() { tokenFlag = oldToken }()

	var output bytes.Buffer
	if err := runAuthToken(&output, configPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(output.String()) != "sk-print-me" {
		t.Errorf("expected 'sk-print-me', got %q", output.String())
	}
}

func TestAuthRefresh_NotImplemented(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")

	var output bytes.Buffer
	if err := runAuthRefresh(&output, configPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output.String(), "not yet implemented") {
		t.Errorf("expected guidance about OAuth not implemented, got: %s", output.String())
	}
}

func TestAuthLogout_ClearsCredentials(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{BaseURL: "https://app.wenmarpro.com"})

	store := authpkg.NewCredentialStore()
	_ = store.SaveToken(context.Background(), &authpkg.Token{AccessToken: "sk-temp"})

	if err := runAuthLogout(configPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("config file should be deleted")
	}
}
