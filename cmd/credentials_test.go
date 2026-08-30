package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	authpkg "github.com/wenmar-pro/wenmar-sdk/go/pkg/auth"
)

func TestWenmarConfigHomeRedirectsCredentials(t *testing.T) {
	// Canary: a token at the real path must NOT be visible when
	// WENMAR_CONFIG_HOME points elsewhere. The canary is removed afterward
	// so the test never leaves a stray file in the user's real config dir.
	home, _ := os.UserHomeDir()
	realDir := filepath.Join(home, ".config", "wenmar")
	realFile := filepath.Join(realDir, "credentials.json")
	if err := os.MkdirAll(realDir, 0o700); err == nil {
		_ = os.WriteFile(realFile, []byte(`{"access_token":"REAL-TOKEN-CANARY"}`), 0o600)
	}
	t.Cleanup(func() { _ = os.Remove(realFile) })

	t.Setenv("WENMAR_CONFIG_HOME", t.TempDir())
	store := newCredentialStore()
	tok, err := store.GetToken(context.Background())
	if err == nil && tok != nil && tok.AccessToken == "REAL-TOKEN-CANARY" {
		t.Error("store read the real credentials file — WENMAR_CONFIG_HOME redirect failed")
	}
}

// ensure authpkg stays referenced (FileStore type) for the concrete check.
var _ authpkg.CredentialStore = authpkg.FileStore{}
