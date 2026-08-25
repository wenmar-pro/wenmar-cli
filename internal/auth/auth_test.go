package auth

import (
	"os"
	"testing"
)

func TestResolveToken_FlagTakesPrecedence(t *testing.T) {
	os.Setenv("WENMAR_TOKEN", "env-token")
	defer os.Unsetenv("WENMAR_TOKEN")

	token, err := ResolveToken("flag-token", "")
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

	token, err := ResolveToken("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "env-token" {
		t.Errorf("expected 'env-token', got '%s'", token)
	}
}

func TestResolveToken_ErrorWhenNoToken(t *testing.T) {
	os.Unsetenv("WENMAR_TOKEN")

	_, err := ResolveToken("", "")
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
