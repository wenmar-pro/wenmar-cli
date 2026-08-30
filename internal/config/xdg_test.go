package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestXDGConfigPath_UsesXDG(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path, err := xdgConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(dir, "wenmar", "config")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestXDGConfigPath_FallsBackToHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := xdgConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(home, ".config", "wenmar", "config")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestMigrateOldConfig_MovesFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	oldPath := filepath.Join(home, ".wenmar", "config")
	os.MkdirAll(filepath.Dir(oldPath), 0755)
	os.WriteFile(oldPath, []byte("token: old-token\nbase_url: https://old.example.com\n"), 0600)

	migrated, err := migrateOldConfig(oldPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !migrated {
		t.Error("expected migrated=true")
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("old config file should be removed after migration")
	}

	newPath, _ := xdgConfigPath()
	cfg, err := LoadFrom(newPath)
	if err != nil {
		t.Fatalf("failed to load migrated config: %v", err)
	}
	if cfg.Token != "old-token" {
		t.Errorf("expected token 'old-token', got '%s'", cfg.Token)
	}
}

func TestMigrateOldConfig_FailureKeepsOldData(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("WENMAR_CONFIG_HOME", cfgHome)

	// Legacy config with a token.
	home := t.TempDir()
	t.Setenv("HOME", home) // oldConfigPath uses os.UserHomeDir
	legacyDir := filepath.Join(home, ".wenmar")
	os.MkdirAll(legacyDir, 0o700)
	oldPath := filepath.Join(legacyDir, "config")
	oldData := []byte("token: legacy-secret\n")
	os.WriteFile(oldPath, oldData, 0o600)

	// Make the NEW location unwritable to force the write failure.
	// (WENMAR_CONFIG_HOME exists; corrupt it by making it a file.)
	blocker := filepath.Join(cfgHome, "wenmar")
	os.WriteFile(blocker, []byte("not a dir"), 0o600)

	_, err := migrateOldConfig(oldPath)
	if err == nil {
		t.Fatal("expected migration failure")
	}
	// The old file must be intact.
	got, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatalf("old config deleted during failed migration: %v", err)
	}
	if string(got) != string(oldData) {
		t.Errorf("old config corrupted: %q", got)
	}
}

func TestMigrateOldConfig_NoOldFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	oldPath := filepath.Join(home, ".wenmar", "config")
	migrated, err := migrateOldConfig(oldPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if migrated {
		t.Error("expected migrated=false when no old config exists")
	}
}
