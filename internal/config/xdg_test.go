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
