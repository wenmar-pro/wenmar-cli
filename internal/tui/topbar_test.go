package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTopBar_RenderContainsSearchPrompt(t *testing.T) {
	tb := NewTopBar()
	view := tb.View(120)
	if !strings.Contains(view, "/") {
		t.Error("expected top bar to contain search prompt '/'")
	}
}

func TestTopBar_RenderContainsStubBadges(t *testing.T) {
	tb := NewTopBar()
	view := tb.View(120)
	if !strings.Contains(view, "📨") {
		t.Error("expected top bar to contain messages badge")
	}
	if !strings.Contains(view, "🔔") {
		t.Error("expected top bar to contain notifications badge")
	}
}

func TestTopBar_RenderContainsQuickActions(t *testing.T) {
	tb := NewTopBar()
	view := tb.View(120)
	if !strings.Contains(view, "new") {
		t.Error("expected top bar to contain 'new' quick action")
	}
	if !strings.Contains(view, "refresh") {
		t.Error("expected top bar to contain 'refresh' quick action")
	}
}

func TestTopBar_FocusSearchOnSlashKey(t *testing.T) {
	tb := NewTopBar()
	if tb.searchFocused {
		t.Fatal("expected search not focused initially")
	}
	tb, _ = tb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	if !tb.searchFocused {
		t.Fatal("expected search focused after '/' key")
	}
}

func TestTopBar_UnfocusSearchOnEscape(t *testing.T) {
	tb := NewTopBar()
	tb.searchFocused = true
	tb, _ = tb.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if tb.searchFocused {
		t.Fatal("expected search unfocused after escape")
	}
}

func TestTopBar_SearchValueEmitsFilterMsg(t *testing.T) {
	tb := NewTopBar()
	tb.FocusSearch()
	tb, _ = tb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	tb, _ = tb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	tb, _ = tb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	tb, _ = tb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	if tb.SearchValue() != "jane" {
		t.Fatalf("expected search value 'jane', got %q", tb.SearchValue())
	}
}

func TestTopBar_SearchValueWhenEmpty(t *testing.T) {
	tb := NewTopBar()
	if tb.SearchValue() != "" {
		t.Fatalf("expected empty search value, got %q", tb.SearchValue())
	}
}

func TestTopBar_SetAccountName(t *testing.T) {
	tb := NewTopBar()
	tb.SetAccountName("Acme Auto")
	view := tb.View(120)
	if !strings.Contains(view, "Acme Auto") {
		t.Error("expected top bar to contain account name 'Acme Auto'")
	}
}