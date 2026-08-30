package oauth

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestWaitForCallback_Success(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	expectedState := "test-state-abc"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Make the callback request in a goroutine
	go func() {
		time.Sleep(50 * time.Millisecond) // let the server start
		resp, err := http.Get("http://" + listener.Addr().String() + "/callback?code=test-code-123&state=" + expectedState)
		if err != nil {
			t.Errorf("callback request failed: %v", err)
			return
		}
		resp.Body.Close()
	}()

	code, err := WaitForCallback(ctx, expectedState, listener)
	if err != nil {
		t.Fatalf("WaitForCallback failed: %v", err)
	}
	if code != "test-code-123" {
		t.Errorf("code = %q, want test-code-123", code)
	}
}

func TestWaitForCallback_StateMismatch(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		resp, _ := http.Get("http://" + listener.Addr().String() + "/callback?code=test-code&state=wrong-state")
		if resp != nil {
			resp.Body.Close()
		}
	}()

	_, err = WaitForCallback(ctx, "expected-state", listener)
	if err == nil {
		t.Fatal("expected error for state mismatch, got nil")
	}
}

func TestWaitForCallback_MissingCode(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		resp, _ := http.Get("http://" + listener.Addr().String() + "/callback?state=test-state")
		if resp != nil {
			resp.Body.Close()
		}
	}()

	_, err = WaitForCallback(ctx, "test-state", listener)
	if err == nil {
		t.Fatal("expected error for missing code, got nil")
	}
}

func TestWaitForCallback_ContextCancelled(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err = WaitForCallback(ctx, "state", listener)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

func TestWaitForCallback_ErrorParam(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		resp, _ := http.Get("http://" + listener.Addr().String() + "/callback?error=access_denied&state=test-state")
		if resp != nil {
			resp.Body.Close()
		}
	}()

	_, err = WaitForCallback(ctx, "test-state", listener)
	if err == nil {
		t.Fatal("expected error for OAuth error param, got nil")
	}
}
