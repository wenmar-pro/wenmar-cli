package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/config"
)

func TestResolveToken_FlagTakesPrecedence(t *testing.T) {
	os.Setenv("WENMAR_TOKEN", "env-token")
	defer os.Unsetenv("WENMAR_TOKEN")

	token, err := ResolveToken("flag-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "flag-token" {
		t.Errorf("expected 'flag-token', got '%s'", token)
	}
}

func TestResolveToken_EnvVarFallback(t *testing.T) {
	os.Setenv("WENMAR_TOKEN", "env-token")
	defer os.Unsetenv("WENMAR_TOKEN")

	token, err := ResolveToken("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "env-token" {
		t.Errorf("expected 'env-token', got '%s'", token)
	}
}

func TestResolveToken_ErrorWhenNoToken(t *testing.T) {
	os.Unsetenv("WENMAR_TOKEN")

	_, err := ResolveToken("")
	if err == nil {
		t.Fatal("expected error when no token provided")
	}
}

func TestResolveBaseURL_FlagTakesPrecedence(t *testing.T) {
	os.Setenv("WENMAR_BASE_URL", "https://env.example.com")
	defer os.Unsetenv("WENMAR_BASE_URL")

	url := ResolveBaseURL("https://flag.example.com")
	if url != "https://flag.example.com" {
		t.Errorf("expected 'https://flag.example.com', got '%s'", url)
	}
}

func TestResolveBaseURL_EnvVarFallback(t *testing.T) {
	os.Setenv("WENMAR_BASE_URL", "https://env.example.com")
	defer os.Unsetenv("WENMAR_BASE_URL")

	url := ResolveBaseURL("")
	if url != "https://env.example.com" {
		t.Errorf("expected 'https://env.example.com', got '%s'", url)
	}
}

func TestResolveBaseURL_Default(t *testing.T) {
	os.Unsetenv("WENMAR_BASE_URL")

	url := ResolveBaseURL("")
	if url != "https://app.wenmarpro.com" {
		t.Errorf("expected default, got '%s'", url)
	}
}

func TestResolveToken_ReadsConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "from-config"})

	token, err := ResolveTokenFrom("", configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "from-config" {
		t.Errorf("expected 'from-config', got '%s'", token)
	}
}

func TestResolveToken_FlagOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "from-config"})

	token, err := ResolveTokenFrom("from-flag", configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "from-flag" {
		t.Errorf("expected 'from-flag', got '%s'", token)
	}
}

func TestResolveToken_EnvOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "from-config"})

	os.Setenv("WENMAR_TOKEN", "from-env")
	defer os.Unsetenv("WENMAR_TOKEN")

	token, err := ResolveTokenFrom("", configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "from-env" {
		t.Errorf("expected 'from-env', got '%s'", token)
	}
}

func TestResolveToken_NoTokenReturnsError(t *testing.T) {
	_, err := ResolveTokenFrom("", "/nonexistent/config")
	if err == nil {
		t.Error("expected error when no token is configured")
	}
}

func TestResolveBaseURL_ReadsConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{BaseURL: "https://from-config.example.com"})

	url := ResolveBaseURLFrom("", configPath)
	if url != "https://from-config.example.com" {
		t.Errorf("expected 'https://from-config.example.com', got '%s'", url)
	}
}
