package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	defaultBaseURL = "https://app.wenmarpro.com"
)

type Config struct {
	Token   string `yaml:"token"`
	BaseURL string `yaml:"base_url"`
}

func ConfigPath() (string, error) {
	// Try XDG path first
	newPath, err := xdgConfigPath()
	if err != nil {
		return "", err
	}

	// Check if new config exists
	if _, err := os.Stat(newPath); err == nil {
		return newPath, nil
	}

	// Try migrating from old path
	oldPath, err := oldConfigPath()
	if err != nil {
		return newPath, nil
	}

	migrated, err := migrateOldConfig(oldPath)
	if err != nil {
		// Log but don't fail — fall back to new path
		return newPath, nil
	}

	if migrated {
		fmt.Fprintf(os.Stderr, "  Migrated config from ~/.wenmar/ to ~/.config/wenmar/\n")
	}

	return newPath, nil
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return SaveTo(path, cfg)
}

func SaveTo(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("could not write config file: %w", err)
	}

	return nil
}

func Delete() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return DeleteFrom(path)
}

func DeleteFrom(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not delete config file: %w", err)
	}
	return nil
}
