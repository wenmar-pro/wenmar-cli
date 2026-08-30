package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
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
// It fetches through the SDK client so it inherits auth, retry, and caching.
type Poller struct {
	Client      *wenmar.Client
	Resource    string // "work_orders", "customers", "vehicles"
	LocationID  string
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
	resource := p.Resource

	for id, item := range resp {
		oldItem, exists := prev[id]
		if !exists {
			p.maybeEmit(emit, Event{Type: "new", ID: item["id"], Resource: resource, At: now})
			continue
		}

		// Check for changed fields
		changed := diffFields(oldItem, item)
		if len(changed) > 0 {
			p.maybeEmit(emit, Event{Type: "changed", ID: item["id"], Resource: resource, ChangedFields: changed, At: now})
		}
	}

	// Find removed items
	for id, item := range prev {
		if _, exists := resp[id]; !exists {
			p.maybeEmit(emit, Event{Type: "removed", ID: item["id"], Resource: resource, At: now})
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

// fetch retrieves the current list of items through the SDK client.
func (p *Poller) fetch(ctx context.Context) (map[string]map[string]any, error) {
	client := p.Client
	if p.LocationID != "" {
		lc, err := client.ForLocation(ctx, p.LocationID)
		if err != nil {
			return nil, err
		}
		client = lc.Client
	}

	var items []map[string]any
	switch p.Resource {
	case "customers":
		resp, err := client.ListCustomers(ctx)
		if err != nil {
			return nil, err
		}
		items = decodeList(resp.JSON200)
	case "vehicles":
		resp, err := client.ListVehicles(ctx)
		if err != nil {
			return nil, err
		}
		items = decodeList(resp.JSON200)
	default: // work_orders
		resp, err := client.ListWorkOrders(ctx)
		if err != nil {
			return nil, err
		}
		items = decodeList(resp.JSON200)
	}

	return indexByID(items), nil
}

// decodeList converts a typed list response into generic maps for diffing.
func decodeList[T any](list *[]T) []map[string]any {
	if list == nil {
		return nil
	}
	items := make([]map[string]any, 0, len(*list))
	for _, item := range *list {
		b, err := json.Marshal(item)
		if err != nil {
			continue
		}
		var m map[string]any
		if json.Unmarshal(b, &m) == nil {
			items = append(items, m)
		}
	}
	return items
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
