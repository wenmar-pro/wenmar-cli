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
	Token      string `yaml:"token"`
	BaseURL    string `yaml:"base_url"`
	AuthMethod string `yaml:"auth_method"`
	LocationID string `yaml:"location_id"`
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

// ValueWithSource pairs a resolved value with where it came from.
type ValueWithSource struct {
	Value  string
	Source string
}

// ResolveWithProvenance resolves each config value and reports its source.
// Sources are: "flag", "env", "keyring", "config file", "default".
func ResolveWithProvenance(flagToken, flagBaseURL, flagLocation, configPath string) map[string]ValueWithSource {
	result := map[string]ValueWithSource{}

	// Token
	switch {
	case flagToken != "":
		result["token"] = ValueWithSource{Value: flagToken, Source: "flag"}
	case os.Getenv("WENMAR_TOKEN") != "":
		result["token"] = ValueWithSource{Value: os.Getenv("WENMAR_TOKEN"), Source: "env WENMAR_TOKEN"}
	default:
		result["token"] = ValueWithSource{Value: "", Source: "none"}
	}

	// Base URL
	switch {
	case flagBaseURL != "":
		result["base_url"] = ValueWithSource{Value: flagBaseURL, Source: "flag"}
	case os.Getenv("WENMAR_URL") != "":
		result["base_url"] = ValueWithSource{Value: os.Getenv("WENMAR_URL"), Source: "env WENMAR_URL"}
	case configPath != "":
		if cfg, err := LoadFrom(configPath); err == nil && cfg.BaseURL != "" {
			result["base_url"] = ValueWithSource{Value: cfg.BaseURL, Source: "config file"}
		} else {
			result["base_url"] = ValueWithSource{Value: defaultBaseURL, Source: "default"}
		}
	default:
		result["base_url"] = ValueWithSource{Value: defaultBaseURL, Source: "default"}
	}

	// Location ID
	switch {
	case flagLocation != "":
		result["location_id"] = ValueWithSource{Value: flagLocation, Source: "flag"}
	case os.Getenv("WENMAR_LOCATION_ID") != "":
		result["location_id"] = ValueWithSource{Value: os.Getenv("WENMAR_LOCATION_ID"), Source: "env WENMAR_LOCATION_ID"}
	case configPath != "":
		if cfg, err := LoadFrom(configPath); err == nil && cfg.LocationID != "" {
			result["location_id"] = ValueWithSource{Value: cfg.LocationID, Source: "config file"}
		} else {
			result["location_id"] = ValueWithSource{Value: "", Source: "none"}
		}
	default:
		result["location_id"] = ValueWithSource{Value: "", Source: "none"}
	}

	// Auth method
	if configPath != "" {
		if cfg, err := LoadFrom(configPath); err == nil && cfg.AuthMethod != "" {
			result["auth_method"] = ValueWithSource{Value: cfg.AuthMethod, Source: "config file"}
		} else {
			result["auth_method"] = ValueWithSource{Value: "static", Source: "default"}
		}
	} else {
		result["auth_method"] = ValueWithSource{Value: "static", Source: "default"}
	}

	return result
}
