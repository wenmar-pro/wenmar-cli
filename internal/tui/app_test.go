package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestKeys_HasSidebarAndSearchBindings(t *testing.T) {
	if len(Keys.SidebarToggle.Keys()) == 0 {
		t.Error("SidebarToggle binding has no keys")
	}
	if len(Keys.FocusSearch.Keys()) == 0 {
		t.Error("FocusSearch binding has no keys")
	}
}

func TestAppModel_TabSwitch(t *testing.T) {
	m := NewApp(nil, "", 0)
	if m.active != 0 {
		t.Fatalf("expected active tab 0, got %d", m.active)
	}

	// Tab cycles forward.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(AppModel)
	if m.active != 1 {
		t.Fatalf("expected active tab 1 after tab, got %d", m.active)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(AppModel)
	if m.active != 2 {
		t.Fatalf("expected active tab 2 after tab, got %d", m.active)
	}
	// Wrap around.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(AppModel)
	if m.active != 0 {
		t.Fatalf("expected active tab 0 after wrap, got %d", m.active)
	}

	// Shift+Tab cycles backward.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(AppModel)
	if m.active != 2 {
		t.Fatalf("expected active tab 2 after shift+tab, got %d", m.active)
	}
}

func TestAppModel_TabJump(t *testing.T) {
	m := NewApp(nil, "", 0)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = updated.(AppModel)
	if m.active != 1 {
		t.Fatalf("expected active tab 1 after '2', got %d", m.active)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m = updated.(AppModel)
	if m.active != 2 {
		t.Fatalf("expected active tab 2 after '3', got %d", m.active)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m = updated.(AppModel)
	if m.active != 0 {
		t.Fatalf("expected active tab 0 after '1', got %d", m.active)
	}
}

func TestAppModel_HelpToggle(t *testing.T) {
	m := NewApp(nil, "", 0)
	if m.showHelp {
		t.Fatal("expected help hidden initially")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(AppModel)
	if !m.showHelp {
		t.Fatal("expected help shown after '?'")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(AppModel)
	if m.showHelp {
		t.Fatal("expected help hidden after second '?'")
	}
}
