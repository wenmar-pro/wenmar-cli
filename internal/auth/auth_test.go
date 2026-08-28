package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/config"
	authpkg "github.com/wenmar-pro/wenmar-sdk/go/pkg/auth"
)

// emptyStore is an in-memory CredentialStore with no token, so tests are
// deterministic regardless of the real keyring/file state.
type emptyStore struct{}

func (emptyStore) GetToken(context.Context) (*authpkg.Token, error) {
	return nil, errors.New("no token stored")
}
func (emptyStore) SaveToken(context.Context, *authpkg.Token) error { return nil }
func (emptyStore) DeleteToken(context.Context) error               { return nil }

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
	os.Setenv("WENMAR_URL", "https://env.example.com")
	defer os.Unsetenv("WENMAR_URL")

	url := ResolveBaseURL("https://flag.example.com")
	if url != "https://flag.example.com" {
		t.Errorf("expected 'https://flag.example.com', got '%s'", url)
	}
}

func TestResolveBaseURL_EnvVarFallback(t *testing.T) {
	os.Setenv("WENMAR_URL", "https://env.example.com")
	defer os.Unsetenv("WENMAR_URL")

	url := ResolveBaseURL("")
	if url != "https://env.example.com" {
		t.Errorf("expected 'https://env.example.com', got '%s'", url)
	}
}

func TestResolveBaseURL_Default(t *testing.T) {
	os.Unsetenv("WENMAR_URL")

	url := ResolveBaseURL("")
	if url != "https://app.wenmarpro.com" {
		t.Errorf("expected default, got '%s'", url)
	}
}

func TestResolveToken_ReadsConfigFile(t *testing.T) {
		dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "from-config"})

	rt, err := ResolveTokenWithSourceFrom("", configPath, emptyStore{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Token != "from-config" {
		t.Errorf("expected 'from-config', got '%s'", rt.Token)
	}
}

func TestResolveToken_FlagOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "from-config"})

	rt, err := ResolveTokenWithSourceFrom("from-flag", configPath, emptyStore{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Token != "from-flag" {
		t.Errorf("expected 'from-flag', got '%s'", rt.Token)
	}
}

func TestResolveToken_EnvOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "from-config"})

	os.Setenv("WENMAR_TOKEN", "from-env")
	defer os.Unsetenv("WENMAR_TOKEN")

	rt, err := ResolveTokenWithSourceFrom("", configPath, emptyStore{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Token != "from-env" {
		t.Errorf("expected 'from-env', got '%s'", rt.Token)
	}
}

func TestResolveToken_NoTokenReturnsError(t *testing.T) {
		_, err := ResolveTokenWithSourceFrom("", "/nonexistent/config", emptyStore{})
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

func TestResolveAuthManager_FlagPrecedence(t *testing.T) {
	os.Unsetenv("WENMAR_TOKEN")
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "from-config"})

	m, err := ResolveAuthManagerWithStore("from-flag", configPath, emptyStore{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tok, err := m.Token(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "from-flag" {
		t.Errorf("expected 'from-flag', got '%s'", tok)
	}
}

func TestResolveAuthManager_EnvPrecedence(t *testing.T) {
	os.Setenv("WENMAR_TOKEN", "from-env")
	defer os.Unsetenv("WENMAR_TOKEN")
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "from-config"})

	m, err := ResolveAuthManagerWithStore("", configPath, emptyStore{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tok, err := m.Token(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "from-env" {
		t.Errorf("expected 'from-env', got '%s'", tok)
	}
}

func TestResolveAuthManager_ConfigFallback(t *testing.T) {
		os.Unsetenv("WENMAR_TOKEN")
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{Token: "from-config"})

	m, err := ResolveAuthManagerWithStore("", configPath, emptyStore{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tok, err := m.Token(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "from-config" {
		t.Errorf("expected 'from-config', got '%s'", tok)
	}
}

func TestResolveAuthManager_NoToken(t *testing.T) {
		os.Unsetenv("WENMAR_TOKEN")
	_, err := ResolveAuthManagerWithStore("", "/nonexistent/config", emptyStore{})
	if err == nil {
		t.Error("expected error when no token configured")
	}
}

func TestResolveLocationID_FlagPrecedence(t *testing.T) {
	os.Setenv("WENMAR_LOCATION_ID", "env-loc")
	defer os.Unsetenv("WENMAR_LOCATION_ID")

	if got := ResolveLocationID("flag-loc", ""); got != "flag-loc" {
		t.Errorf("expected 'flag-loc', got '%s'", got)
	}
}

func TestResolveLocationID_EnvFallback(t *testing.T) {
	os.Setenv("WENMAR_LOCATION_ID", "env-loc")
	defer os.Unsetenv("WENMAR_LOCATION_ID")

	if got := ResolveLocationID("", ""); got != "env-loc" {
		t.Errorf("expected 'env-loc', got '%s'", got)
	}
}

func TestResolveLocationID_ConfigFallback(t *testing.T) {
	os.Unsetenv("WENMAR_LOCATION_ID")
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config.SaveTo(configPath, &config.Config{LocationID: "cfg-loc"})

	if got := ResolveLocationID("", configPath); got != "cfg-loc" {
		t.Errorf("expected 'cfg-loc', got '%s'", got)
	}
}
