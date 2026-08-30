package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// xdgConfigPath returns the config path: $WENMAR_CONFIG_HOME/wenmar/config
// when WENMAR_CONFIG_HOME is set (used by tests to avoid touching real
// credentials), otherwise $XDG_CONFIG_HOME/wenmar/config, or
// ~/.config/wenmar/config as fallback.
func xdgConfigPath() (string, error) {
	if base := os.Getenv("WENMAR_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "wenmar", "config"), nil
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine home directory: %w", err)
		}
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "wenmar", "config"), nil
}

// oldConfigPath returns the legacy ~/.wenmar/config path.
func oldConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".wenmar", "config"), nil
}

// migrateOldConfig moves the legacy ~/.wenmar/config to the XDG path
// if the old file exists and the new one doesn't. Returns true if
// migration occurred.
func migrateOldConfig(oldPath string) (bool, error) {
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return false, nil
	}

	newPath, err := xdgConfigPath()
	if err != nil {
		return false, err
	}

	// Don't overwrite if new config already exists
	if _, err := os.Stat(newPath); err == nil {
		return false, nil
	}

	// Read old, write to new atomically (no empty-config window), then
	// remove the old file.
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return false, fmt.Errorf("could not read old config: %w", err)
	}

	dir := filepath.Dir(newPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, fmt.Errorf("could not create new config dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".migrate-*.tmp")
	if err != nil {
		return false, fmt.Errorf("could not create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return false, fmt.Errorf("could not write migrated config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return false, fmt.Errorf("could not sync migrated config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return false, fmt.Errorf("could not close migrated config: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return false, fmt.Errorf("could not set permissions: %w", err)
	}
	if err := os.Rename(tmpPath, newPath); err != nil {
		return false, fmt.Errorf("could not move migrated config into place: %w", err)
	}

	// Only remove the old file once the new one is durably in place.
	if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
		return true, fmt.Errorf("migrated config but could not remove old file at %s: %w", oldPath, err)
	}

	return true, nil
}
