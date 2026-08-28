package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Event represents a change detected by the poller.
type Event struct {
	Type          string         `json:"event"` // "new", "changed", "removed"
	Resource      string         `json:"resource"`
	ID            any            `json:"id"`
	ChangedFields map[string]any `json:"changed_fields,omitempty"`
	At            time.Time      `json:"at"`
}

// Poller polls a list endpoint on an interval, diffs the results against
// the previous state, and emits events for new, changed, and removed items.
type Poller struct {
	URL         string
	Token       string
	Interval    time.Duration
	ExitOnFirst bool

	// Filters
	EventTypes map[string]bool // e.g. {"new": true, "changed": true}

	mu       sync.Mutex
	previous map[string]map[string]any // id → item
}

func (p *Poller) Run(ctx context.Context, emit func(Event)) error {
	p.mu.Lock()
	p.previous = nil
	p.mu.Unlock()

	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()

	// Do an immediate first poll
	if err := p.poll(ctx, emit); err != nil {
		return err
	}

	if p.ExitOnFirst {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := p.poll(ctx, emit); err != nil {
				return err
			}
		}
	}
}

func (p *Poller) poll(ctx context.Context, emit func(Event)) error {
	resp, err := p.fetch(ctx)
	if err != nil {
		return err
	}

	p.mu.Lock()
	prev := p.previous
	p.previous = resp
	p.mu.Unlock()

	if prev == nil {
		// First poll — establish baseline, no events
		return nil
	}

	now := time.Now()

	// Diff: find new and changed items
	prevByID := prev
	respByID := resp

	for id, item := range respByID {
		oldItem, exists := prevByID[id]
		if !exists {
			p.maybeEmit(emit, Event{Type: "new", ID: item["id"], Resource: detectResource(p.URL), At: now})
			continue
		}

		// Check for changed fields
		changed := diffFields(oldItem, item)
		if len(changed) > 0 {
			p.maybeEmit(emit, Event{Type: "changed", ID: item["id"], Resource: detectResource(p.URL), ChangedFields: changed, At: now})
		}
	}

	// Find removed items
	for id, item := range prevByID {
		if _, exists := respByID[id]; !exists {
			p.maybeEmit(emit, Event{Type: "removed", ID: item["id"], Resource: detectResource(p.URL), At: now})
		}
	}

	return nil
}

func (p *Poller) maybeEmit(emit func(Event), e Event) {
	if p.EventTypes != nil && !p.EventTypes[e.Type] {
		return
	}
	emit(e)
}

func (p *Poller) fetch(ctx context.Context) (map[string]map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var items []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("could not decode response: %w", err)
	}

	return indexByID(items), nil
}

func indexByID(items []map[string]any) map[string]map[string]any {
	result := make(map[string]map[string]any)
	for _, item := range items {
		id := fmt.Sprintf("%v", item["id"])
		result[id] = item
	}
	return result
}

func diffFields(old, new map[string]any) map[string]any {
	changes := make(map[string]any)
	for k, newVal := range new {
		oldVal, exists := old[k]
		if !exists || !valuesEqual(oldVal, newVal) {
			changes[k] = newVal
		}
	}
	return changes
}

func valuesEqual(a, b any) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func detectResource(url string) string {
	// Simple heuristic: /work_orders → "work_order", /customers → "customer"
	if strings.Contains(url, "/work_orders") {
		return "work_order"
	}
	if strings.Contains(url, "/customers") {
		return "customer"
	}
	if strings.Contains(url, "/vehicles") {
		return "vehicle"
	}
	return "unknown"
}
