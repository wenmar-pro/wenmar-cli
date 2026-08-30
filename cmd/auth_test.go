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
	defer clearStoredCredentials(t)

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
	t.Setenv("WENMAR_CONFIG_HOME", t.TempDir())

	oldToken := tokenFlag
	tokenFlag = "sk-test-1234"
	defer func() { tokenFlag = oldToken }()

	var output bytes.Buffer
	if err := runAuthLogin(&output, configPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store := newCredentialStore()
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

func TestAuthRefresh_StaticTokenGuidance(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	t.Setenv("WENMAR_CONFIG_HOME", t.TempDir())

	// Seed a static (non-OAuth) token so the codepath is deterministic —
	// other tests may have left tokens in the shared real credential store.
	credsPath := credentialsJSONPath(t)
	seedCredsFile(t, `{"access_token":"static-only"}`)

	var output bytes.Buffer
	if err := runAuthRefresh(&output, configPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output.String(), "No OAuth token to refresh") {
		t.Errorf("expected static-token guidance, got: %s", output.String())
	}
	_ = credsPath
}

func TestAuthRefresh_NotLoggedIn(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	t.Setenv("WENMAR_CONFIG_HOME", t.TempDir())

	clearStoredCredentials(t)

	var output bytes.Buffer
	if err := runAuthRefresh(&output, configPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output.String(), "Not logged in") {
		t.Errorf("expected not-logged-in guidance, got: %s", output.String())
	}
}

func TestAuthLogout_ClearsCredentials(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	t.Setenv("WENMAR_CONFIG_HOME", t.TempDir())
	config.SaveTo(configPath, &config.Config{BaseURL: "https://app.wenmarpro.com"})

	store := newCredentialStore()
	_ = store.SaveToken(context.Background(), &authpkg.Token{AccessToken: "sk-temp"})

	if err := runAuthLogout(configPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("config file should be deleted")
	}
}

// credentialsJSONPath returns the file-fallback credentials path used by the
// SDK credential store. When WENMAR_CONFIG_HOME is set it points under that
// dir (the test-isolated path); otherwise it's the real user config path.
func credentialsJSONPath(t *testing.T) string {
	t.Helper()
	if base := os.Getenv("WENMAR_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "wenmar", "credentials.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "wenmar", "credentials.json")
}

// seedCredsFile writes (or removes, when empty) the file-fallback credentials
// used by the keyring-less test environment.
func seedCredsFile(t *testing.T, contents string) {
	t.Helper()
	path := credentialsJSONPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}
}

// clearStoredCredentials removes credentials from both the real keyring and
// the file fallback. runAuthLogin / runAuthRefresh write to the real OS
// keyring via the SDK, so tests that touch credentials must clean up after
// themselves or they leak into other tests.
func clearStoredCredentials(t *testing.T) {
	t.Helper()
	store := newCredentialStore()
	if err := store.DeleteToken(context.Background()); err != nil {
		t.Logf("cleanup: delete credentials: %v", err)
	}
}
