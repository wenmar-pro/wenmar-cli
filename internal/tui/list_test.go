package tui

import (
	"testing"
)

type testItem struct {
	name string
}

func TestTruncateMultibyteSafe(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"José", 10, "José"},  // fits
		{"Zoë", 2, "Z…"},      // cut on rune boundary
		{"日本語テキスト", 3, "日本…"}, // CJK cut
		{"abc", 3, "abc"},     // exact fit unchanged
		{"ab", 0, ""},         // max<=0 must not panic
	}
	for _, tc := range cases {
		if got := truncate(tc.in, tc.max); got != tc.want {
			t.Errorf("truncate(%q,%d) = %q, want %q", tc.in, tc.max, got, tc.want)
		}
	}
}

func TestListModel_CursorNavigation(t *testing.T) {
	m := &ListModel[testItem]{}
	m.setItems([]testItem{{"a"}, {"b"}, {"c"}})

	if m.cursor != 0 {
		t.Fatalf("expected cursor 0, got %d", m.cursor)
	}

	m.moveDown()
	if m.cursor != 1 {
		t.Fatalf("expected cursor 1, got %d", m.cursor)
	}
	m.moveDown()
	if m.cursor != 2 {
		t.Fatalf("expected cursor 2, got %d", m.cursor)
	}
	// Clamp at bottom.
	m.moveDown()
	if m.cursor != 2 {
		t.Fatalf("expected cursor clamped at 2, got %d", m.cursor)
	}

	m.moveUp()
	if m.cursor != 1 {
		t.Fatalf("expected cursor 1, got %d", m.cursor)
	}
	m.moveUp()
	if m.cursor != 0 {
		t.Fatalf("expected cursor 0, got %d", m.cursor)
	}
	// Clamp at top.
	m.moveUp()
	if m.cursor != 0 {
		t.Fatalf("expected cursor clamped at 0, got %d", m.cursor)
	}
}

func TestListModel_ClampsCursorOnShrink(t *testing.T) {
	m := &ListModel[testItem]{}
	m.setItems([]testItem{{"a"}, {"b"}, {"c"}})
	m.moveDown()
	m.moveDown()
	if m.cursor != 2 {
		t.Fatalf("expected cursor 2, got %d", m.cursor)
	}
	m.setItems([]testItem{{"a"}})
	if m.cursor != 0 {
		t.Fatalf("expected cursor clamped to 0, got %d", m.cursor)
	}
}

func TestListModel_Selected(t *testing.T) {
	m := &ListModel[testItem]{}
	if _, ok := m.selected(); ok {
		t.Fatal("expected no selection on empty list")
	}
	m.setItems([]testItem{{"a"}, {"b"}})
	m.moveDown()
	item, ok := m.selected()
	if !ok {
		t.Fatal("expected a selection")
	}
	if item.name != "b" {
		t.Fatalf("expected 'b', got %q", item.name)
	}
}

func TestListModel_RefreshKey(t *testing.T) {
	m := &ListModel[testItem]{}
	m.setItems([]testItem{{"a"}})
	m.startLoading()
	if !m.loading {
		t.Fatal("expected loading state after startLoading")
	}
	if m.err != nil {
		t.Fatalf("expected nil error, got %v", m.err)
	}
}
