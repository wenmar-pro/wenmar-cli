package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_ReturnsConfigWhenFileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".wenmar", "config")
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte("token: abc123\nbase_url: https://custom.example.com\n"), 0600)

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Token != "abc123" {
		t.Errorf("expected token 'abc123', got '%s'", cfg.Token)
	}
	if cfg.BaseURL != "https://custom.example.com" {
		t.Errorf("expected base_url 'https://custom.example.com', got '%s'", cfg.BaseURL)
	}
}

func TestLoad_ReturnsErrorWhenFileDoesNotExist(t *testing.T) {
	_, err := LoadFrom("/nonexistent/path/config")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoad_DefaultsBaseURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte("token: abc123\n"), 0600)

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != "https://app.wenmarpro.com" {
		t.Errorf("expected default base_url, got '%s'", cfg.BaseURL)
	}
}

func TestSave_WritesFileWithCorrectFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".wenmar", "config")

	cfg := &Config{Token: "secret123", BaseURL: "https://api.example.com"}
	err := SaveTo(path, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 permissions, got %v", info.Mode().Perm())
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "token: secret123") {
		t.Error("expected token in file")
	}
	if !strings.Contains(string(data), "base_url: https://api.example.com") {
		t.Error("expected base_url in file")
	}
}

func TestSave_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "config")

	cfg := &Config{Token: "test"}
	err := SaveTo(path, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected file to be created")
	}
}

func TestDelete_RemovesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte("token: abc"), 0600)

	err := DeleteFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

func TestDelete_NoErrorWhenFileMissing(t *testing.T) {
	err := DeleteFrom("/nonexistent/path/config")
	if err != nil {
		t.Errorf("expected no error for missing file, got %v", err)
	}
}
