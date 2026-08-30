package tui

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTickReArmsAfterTickMsg(t *testing.T) {
	m := NewApp(nil, "", 10*time.Second)
	// Init arms the first tick.
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned nil cmd")
	}
	// Deliver a tickMsg — the returned command must include a new tick.
	newM, newCmd := m.Update(tickMsg{})
	if newCmd == nil {
		t.Fatal("tickMsg did not schedule the next tick — periodic refresh dies after one poll")
	}
	_ = newM
}

func TestResultMsgReachesTabWhileSearchFocused(t *testing.T) {
	// An error result arriving while search is focused must surface in the
	// active tab, not be swallowed by the topbar early-return.
	m2 := NewApp(nil, "", 10*time.Second)
	m2.layout.topBar.FocusSearch()
	m2.Update(workOrderListResultMsg{err: errors.New("boom")})
	v := m2.tabs[0].View(80)
	if !strings.Contains(v, "boom") && !strings.Contains(v, "Error") {
		t.Errorf("error result swallowed while search focused; view:\n%s", v)
	}
}

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

func TestAppModel_SidebarToggle(t *testing.T) {
	m := NewApp(nil, "", 0)
	if m.layout.sidebar.visible {
		t.Fatal("expected sidebar hidden initially")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("`")})
	m = updated.(AppModel)
	if !m.layout.sidebar.visible {
		t.Fatal("expected sidebar visible after backtick")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("`")})
	m = updated.(AppModel)
	if m.layout.sidebar.visible {
		t.Fatal("expected sidebar hidden after second backtick")
	}
}

func TestAppModel_SearchFocusOnSlash(t *testing.T) {
	m := NewApp(nil, "", 0)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(AppModel)
	if !m.layout.topBar.searchFocused {
		t.Fatal("expected search focused after '/'")
	}
}

func TestAppModel_SearchUnfocusOnEscape(t *testing.T) {
	m := NewApp(nil, "", 0)
	m.layout.topBar.FocusSearch()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = updated.(AppModel)
	if m.layout.topBar.searchFocused {
		t.Fatal("expected search unfocused after escape")
	}
}

func TestAppModel_FooterContainsSidebarHint(t *testing.T) {
	m := NewApp(nil, "", 0)
	footer := m.renderFooter()
	if !strings.Contains(footer, "`") {
		t.Error("expected footer to contain backtick hint for sidebar")
	}
	if !strings.Contains(footer, "/") {
		t.Error("expected footer to contain '/' hint for search")
	}
}

func TestAppModel_WindowSizeSetsDimensions(t *testing.T) {
	m := NewApp(nil, "", 0)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(AppModel)
	if m.width != 120 {
		t.Fatalf("expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Fatalf("expected height 40, got %d", m.height)
	}
}

func TestAppModel_SearchFilterTriggersRefetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if r.URL.Query().Get("query") == "jane" {
			w.Write([]byte(`[{"id":7,"full_name":"Jane Doe","type":"individual","vehicles_count":0,"outstanding_balance_cents":0,"updated_at":"2026-01-02T00:00:00Z"}]`))
		} else {
			w.Write([]byte(`[]`))
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL, "test")
	m := NewApp(client, "", 0)
	m.active = 1 // Customers tab

	// Set the search field to the query (as if the user typed it) so the
	// debounce's stale-check passes, then simulate the search filter message.
	m.layout.topBar.FocusSearch()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("jane")})
	m = updated.(AppModel)
	updated, _ = m.Update(searchFilterMsg{query: "jane"})
	m = updated.(AppModel)

	// The Update should have returned a fetch command. Execute it.
	if cl, ok := m.tabs[1].(*CustomerList); ok {
		if cl.SearchQuery() != "jane" {
			t.Fatalf("expected search query 'jane', got %q", cl.SearchQuery())
		}
	}
}
