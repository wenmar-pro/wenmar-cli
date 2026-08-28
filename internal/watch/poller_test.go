package watch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoller_EmitsNewItems(t *testing.T) {
	var requestCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			// First poll: one work order
			json.NewEncoder(w).Encode([]map[string]any{
				{"id": float64(1), "status": "open"},
			})
		} else {
			// Second poll: two work orders (one new, one changed)
			json.NewEncoder(w).Encode([]map[string]any{
				{"id": float64(1), "status": "in_progress"}, // changed
				{"id": float64(2), "status": "open"},         // new
			})
		}
	}))
	defer srv.Close()

	poller := &Poller{
		URL:      srv.URL,
		Token:    "test",
		Interval: 100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var events []Event
	var mu sync.Mutex
	seenNew := false
	seenChanged := false

	err := poller.Run(ctx, func(e Event) {
		mu.Lock()
		events = append(events, e)
		if e.Type == "new" {
			seenNew = true
		}
		if e.Type == "changed" {
			seenChanged = true
		}
		done := seenNew && seenChanged
		mu.Unlock()
		if done {
			cancel()
		}
	})
	if err != nil && err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Should have at least 2 events: one for id=1 status change, one for id=2 new
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d: %+v", len(events), events)
	}

	// Verify event types
	hasNew := false
	hasChanged := false
	for _, e := range events {
		if e.Type == "new" {
			hasNew = true
		}
		if e.Type == "changed" {
			hasChanged = true
		}
	}
	if !hasNew {
		t.Error("expected at least one 'new' event")
	}
	if !hasChanged {
		t.Error("expected at least one 'changed' event")
	}
}

func TestPoller_ExitOnFirst(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": float64(1), "status": "open"},
		})
	}))
	defer srv.Close()

	poller := &Poller{
		URL:         srv.URL,
		Token:       "test",
		Interval:    100 * time.Millisecond,
		ExitOnFirst: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	var eventCount int
	err := poller.Run(ctx, func(e Event) {
		eventCount++
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With ExitOnFirst, the first poll establishes baseline (no events) and
	// the poller returns immediately — well before the 2s timeout.
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("expected ExitOnFirst to return quickly, took %v", elapsed)
	}
	if eventCount != 0 {
		t.Errorf("expected 0 events on first poll (baseline), got %d", eventCount)
	}
}
