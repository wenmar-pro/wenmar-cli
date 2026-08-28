package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// tickMsg is sent on each polling interval.
type tickMsg struct{}

func tick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func keyMatches(msg tea.KeyMsg, binding key.Binding) bool {
	for _, k := range binding.Keys() {
		if msg.String() == k {
			return true
		}
	}
	return false
}

// formatCents renders an integer cent amount as a dollar string.
func formatCents(cents int) string {
	return fmt.Sprintf("$%.2f", float64(cents)/100)
}

// stringify converts an interface{} value to its string form, returning an
// empty string for nil.
func stringify(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func statusColored(status string) string {
	switch status {
	case "pending":
		return StatusPending.Render(status)
	case "in_progress":
		return StatusInProgress.Render(status)
	case "completed":
		return StatusCompleted.Render(status)
	default:
		return status
	}
}

// ListModel holds the shared state and behavior for a resource list tab.
// Resource-specific list types embed this and provide fetch, column, and
// detail-navigation behavior.
type ListModel[T any] struct {
	client     *wenmar.Client
	locationID string
	items      []T
	cursor     int
	loading    bool
	err        error
	refreshed  time.Time
}

func (m *ListModel[T]) moveDown() {
	if m.cursor < len(m.items)-1 {
		m.cursor++
	}
}

func (m *ListModel[T]) moveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *ListModel[T]) setItems(items []T) {
	m.items = items
	m.loading = false
	m.err = nil
	m.refreshed = time.Now()
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *ListModel[T]) setError(err error) {
	m.loading = false
	m.err = err
}

func (m *ListModel[T]) startLoading() {
	m.loading = true
	m.err = nil
}

func (m *ListModel[T]) selected() (T, bool) {
	var zero T
	if len(m.items) == 0 || m.cursor < 0 || m.cursor >= len(m.items) {
		return zero, false
	}
	return m.items[m.cursor], true
}

// renderList renders the list with the given title, headers, and row renderer.
// The row renderer returns pre-padded columns for each item.
func (m *ListModel[T]) renderList(title string, headers []string, row func(T) []string) string {
	if m.loading {
		return "  Loading " + title + "...\n"
	}
	if m.err != nil {
		return fmt.Sprintf("  Error: %v\n", m.err)
	}
	if len(m.items) == 0 {
		return "  No " + title + " found.\n"
	}
	s := HeaderStyle.Render("  " + strings.Join(headers, " ")) + "\n"
	for i, item := range m.items {
		line := "  " + strings.Join(row(item), " ")
		if i == m.cursor {
			line = SelectedStyle.Render(line)
		}
		s += line + "\n"
	}
	return s
}
