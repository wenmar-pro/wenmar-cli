package cmd

import (
	"os"
	"path/filepath"

	authpkg "github.com/wenmar-pro/wenmar-sdk/go/pkg/auth"
)

// newCredentialStore returns the SDK credential store with the file fallback
// redirected under $WENMAR_CONFIG_HOME when set. Tests set WENMAR_CONFIG_HOME
// so they never touch the developer's real keyring or credentials file.
func newCredentialStore() authpkg.CredentialStore {
	if base := os.Getenv("WENMAR_CONFIG_HOME"); base != "" {
		return authpkg.FileStore{
			Path: filepath.Join(base, "wenmar", "credentials.json"),
		}
	}
	return authpkg.NewCredentialStore()
}
