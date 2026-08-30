package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSidebar_HiddenByDefault(t *testing.T) {
	s := NewSidebar()
	if s.visible {
		t.Fatal("expected sidebar hidden by default")
	}
	view := s.View(80)
	if view != "" {
		t.Fatalf("expected empty view when hidden, got %q", view)
	}
}

func TestSidebar_ToggleVisibility(t *testing.T) {
	s := NewSidebar()
	if s.visible {
		t.Fatal("expected hidden initially")
	}
	s.Toggle()
	if !s.visible {
		t.Fatal("expected visible after first toggle")
	}
	s.Toggle()
	if s.visible {
		t.Fatal("expected hidden after second toggle")
	}
}

func TestSidebar_NavDownMovesCursor(t *testing.T) {
	s := NewSidebar()
	s.visible = true
	if s.active != 0 {
		t.Fatalf("expected active 0, got %d", s.active)
	}
	s.Update(tea.KeyMsg{Type: tea.KeyDown})
	if s.active != 1 {
		t.Fatalf("expected active 1, got %d", s.active)
	}
}

func TestSidebar_NavDownWraps(t *testing.T) {
	s := NewSidebar()
	s.visible = true
	s.active = len(s.items) - 1
	s.Update(tea.KeyMsg{Type: tea.KeyDown})
	if s.active != 0 {
		t.Fatalf("expected wrap to 0, got %d", s.active)
	}
}

func TestSidebar_NavUpMovesCursor(t *testing.T) {
	s := NewSidebar()
	s.visible = true
	s.active = 2
	s.Update(tea.KeyMsg{Type: tea.KeyUp})
	if s.active != 1 {
		t.Fatalf("expected active 1, got %d", s.active)
	}
}

func TestSidebar_NavUpWraps(t *testing.T) {
	s := NewSidebar()
	s.visible = true
	s.active = 0
	s.Update(tea.KeyMsg{Type: tea.KeyUp})
	if s.active != len(s.items)-1 {
		t.Fatalf("expected wrap to %d, got %d", len(s.items)-1, s.active)
	}
}

func TestSidebar_SelectReturnsIndex(t *testing.T) {
	s := NewSidebar()
	s.visible = true
	s.active = 1
	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if s.Selected() != 1 {
		t.Fatalf("expected selected 1, got %d", s.Selected())
	}
	_ = cmd
}

func TestSidebar_RenderShowsAllItems(t *testing.T) {
	s := NewSidebar()
	s.visible = true
	view := s.View(80)
	for _, item := range s.items {
		if !strings.Contains(view, item.label) {
			t.Errorf("expected view to contain %q", item.label)
		}
	}
}

func TestSidebar_RenderHighlightsActive(t *testing.T) {
	s := NewSidebar()
	s.visible = true
	s.active = 1
	view := s.View(80)
	if !strings.Contains(view, s.items[1].label) {
		t.Errorf("expected view to contain active item %q", s.items[1].label)
	}
}
