package watch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	// MaxConsecutiveFailures is how many consecutive transient errors are
	// tolerated before giving up (0 = default 3).
	MaxConsecutiveFailures int

	// Filters
	EventTypes map[string]bool // e.g. {"new": true, "changed": true}

	previous map[string]map[string]any // id → item
}

func (p *Poller) Run(ctx context.Context, emit func(Event)) error {
	p.previous = nil

	scoped, err := p.scopedClient(ctx)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()

	failures := 0
	maxFailures := p.MaxConsecutiveFailures
	if maxFailures <= 0 {
		maxFailures = 3
	}
	backoff := time.Second

	poll := func() error {
		if err := p.poll(ctx, emit, scoped); err != nil {
			if isAuthError(err) {
				return err // fatal
			}
			failures++
			if failures >= maxFailures {
				return fmt.Errorf("watch: %d consecutive failed polls, giving up: %w", failures, err)
			}
			// Brief backoff, reset on next success.
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			if backoff < 60*time.Second {
				backoff *= 2
			}
			return nil
		}
		failures = 0
		backoff = time.Second
		return nil
	}

	if err := poll(); err != nil {
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
			if err := poll(); err != nil {
				return err
			}
		}
	}
}

func (p *Poller) scopedClient(ctx context.Context) (*wenmar.Client, error) {
	if p.LocationID == "" {
		return p.Client, nil
	}
	return p.Client.ForLocation(p.LocationID), nil
}

func isAuthError(err error) bool {
	var apiErr *wenmar.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 401 || apiErr.StatusCode == 403
	}
	return false
}

func (p *Poller) poll(ctx context.Context, emit func(Event), client *wenmar.Client) error {
	resp, err := p.fetch(ctx, client)
	if err != nil {
		return err
	}

	prev := p.previous
	p.previous = resp

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
func (p *Poller) fetch(ctx context.Context, client *wenmar.Client) (map[string]map[string]any, error) {
	var items []map[string]any
	var err error
	switch p.Resource {
	case "customers":
		resp, ferr := client.ListCustomers(ctx, nil)
		if ferr != nil {
			return nil, ferr
		}
		items, err = decodeList(resp.JSON200)
	case "vehicles":
		resp, ferr := client.ListVehicles(ctx, nil)
		if ferr != nil {
			return nil, ferr
		}
		items, err = decodeList(resp.JSON200)
	default: // work_orders
		resp, ferr := client.ListWorkOrders(ctx, nil)
		if ferr != nil {
			return nil, ferr
		}
		items, err = decodeList(resp.JSON200)
	}
	if err != nil {
		return nil, err
	}

	return indexByID(items), nil
}

// decodeList converts a typed list response into generic maps for diffing.
func decodeList[T any](list *[]T) ([]map[string]any, error) {
	if list == nil {
		return nil, nil
	}
	items := make([]map[string]any, 0, len(*list))
	for _, item := range *list {
		b, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("decode item: %w", err)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("decode item: %w", err)
		}
		items = append(items, m)
	}
	return items, nil
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
