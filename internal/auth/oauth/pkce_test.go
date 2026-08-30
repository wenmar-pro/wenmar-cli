package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestGenerateVerifier_Length(t *testing.T) {
	v, err := GenerateVerifier()
	if err != nil {
		t.Fatalf("GenerateVerifier: %v", err)
	}
	// 32 bytes -> 43 chars in base64url without padding
	if len(v) != 43 {
		t.Errorf("verifier length = %d, want 43", len(v))
	}
}

func TestGenerateVerifier_Unique(t *testing.T) {
	v1, err := GenerateVerifier()
	if err != nil {
		t.Fatalf("GenerateVerifier: %v", err)
	}
	v2, _ := GenerateVerifier()
	if v1 == v2 {
		t.Error("two verifiers should not be equal")
	}
}

func TestGenerateVerifier_ValidBase64URL(t *testing.T) {
	v, err := GenerateVerifier()
	if err != nil {
		t.Fatalf("GenerateVerifier: %v", err)
	}
	_, err = base64.RawURLEncoding.DecodeString(v)
	if err != nil {
		t.Errorf("verifier is not valid base64url: %v", err)
	}
}

func TestGenerateChallenge_S256(t *testing.T) {
	verifier := "dBBjJjN._z7fDwO1NfF8j9Qr2s3t4u5v6w7x8y9z0a1b2c"
	challenge := GenerateChallenge(verifier)

	// S256 = base64url(sha256(verifier))
	sum := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge != expected {
		t.Errorf("challenge = %q, want %q", challenge, expected)
	}
}

func TestGenerateChallenge_EmptyVerifier(t *testing.T) {
	challenge := GenerateChallenge("")
	sum := sha256.Sum256([]byte(""))
	expected := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge != expected {
		t.Errorf("challenge for empty = %q, want %q", challenge, expected)
	}
}

func TestGenerateState_Length(t *testing.T) {
	s, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState: %v", err)
	}
	// 16 bytes -> 22 chars in base64url without padding
	if len(s) != 22 {
		t.Errorf("state length = %d, want 22", len(s))
	}
}

func TestGenerateState_Unique(t *testing.T) {
	s1, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState: %v", err)
	}
	s2, _ := GenerateState()
	if s1 == s2 {
		t.Error("two states should not be equal")
	}
}
