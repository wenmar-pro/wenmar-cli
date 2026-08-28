package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// xdgConfigPath returns the XDG-compliant config path:
// $XDG_CONFIG_HOME/wenmar/config, or ~/.config/wenmar/config as fallback.
func xdgConfigPath() (string, error) {
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

	// Read old, write to new
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return false, fmt.Errorf("could not read old config: %w", err)
	}

	if err := SaveTo(newPath, &Config{}); err != nil {
		return false, fmt.Errorf("could not create new config dir: %w", err)
	}

	// Write the actual content (SaveTo creates the dir; now write the data)
	if err := os.WriteFile(newPath, data, 0600); err != nil {
		return false, fmt.Errorf("could not write migrated config: %w", err)
	}

	// Remove old file
	os.Remove(oldPath)

	return true, nil
}
