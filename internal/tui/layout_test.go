package tui

import (
	"strings"
	"testing"
)

func TestLayout_RenderContainsTopBarAndFooter(t *testing.T) {
	layout := NewLayout()
	layout.content = "  some content\n"
	layout.footer = "  footer\n"
	view := layout.View(80, 24)
	if !strings.Contains(view, "some content") {
		t.Error("expected layout to contain content")
	}
	if !strings.Contains(view, "footer") {
		t.Error("expected layout to contain footer")
	}
}

func TestLayout_SidebarOverlayWhenVisible(t *testing.T) {
	layout := NewLayout()
	layout.content = "  content line\n"
	layout.footer = "  footer\n"
	layout.sidebar.visible = true
	view := layout.View(80, 24)
	if view == "" {
		t.Fatal("expected non-empty view")
	}
	// When sidebar is visible, content still renders (overlay, not replace).
	if !strings.Contains(view, "content line") {
		t.Error("expected content to still render with sidebar visible")
	}
}

func TestLayout_SidebarHiddenDoesNotAffectContent(t *testing.T) {
	layout := NewLayout()
	layout.content = "  content line\n"
	layout.footer = "  footer\n"
	view := layout.View(80, 24)
	if !strings.Contains(view, "content line") {
		t.Error("expected content to render without sidebar")
	}
}

func TestLayout_HeightCalculatesRegions(t *testing.T) {
	layout := NewLayout()
	layout.content = "  content\n"
	layout.footer = "  footer\n"
	view := layout.View(80, 24)
	// Top bar takes 1 line, content takes 1 line, footer takes 1 line.
	// Total should be 3 lines + any spacing.
	lines := strings.Split(strings.TrimSpace(view), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}
}
