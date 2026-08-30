package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// sidebarItem is a single navigation entry in the sidebar.
type sidebarItem struct {
	label  string
	hotkey string
}

// SidebarModel renders the overlay navigation sidebar. It is hidden by
// default and toggled via a hotkey. When visible, it overlays the content
// area without resizing it.
type SidebarModel struct {
	items   []sidebarItem
	active  int
	visible bool
}

// NewSidebar creates a sidebar with the standard nav items, hidden by default.
func NewSidebar() SidebarModel {
	return SidebarModel{
		items: []sidebarItem{
			{label: "Work Orders", hotkey: "1"},
			{label: "Customers", hotkey: "2"},
			{label: "Vehicles", hotkey: "3"},
			{label: "Tags", hotkey: "4"},
			{label: "Vendors", hotkey: "5"},
			{label: "Statements", hotkey: "6"},
			{label: "Settings", hotkey: "7"},
		},
	}
}

// Toggle flips sidebar visibility.
func (m *SidebarModel) Toggle() {
	m.visible = !m.visible
}

// Update handles key messages when the sidebar is visible. Returns a command
// (always nil for now — selection is read by the caller via Selected()).
func (m *SidebarModel) Update(msg tea.Msg) (SidebarModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyDown:
			m.active = (m.active + 1) % len(m.items)
		case tea.KeyUp:
			m.active = (m.active - 1 + len(m.items)) % len(m.items)
		case tea.KeyEnter:
			return *m, nil
		}
	}
	return *m, nil
}

// Selected returns the index of the highlighted nav item.
func (m SidebarModel) Selected() int {
	return m.active
}

// View renders the sidebar. Returns empty string when hidden.
// The width parameter is the sidebar's allocated width (typically ~20 cols).
func (m SidebarModel) View(width int) string {
	if !m.visible {
		return ""
	}
	var b strings.Builder
	for i, item := range m.items {
		line := fmt.Sprintf(" %s  %s ", item.hotkey, item.label)
		if i == m.active {
			line = SidebarActiveItemStyle.Render(line)
		} else {
			line = SidebarInactiveItemStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	return SidebarStyle.Render(b.String())
}

// Width returns the ideal sidebar width in columns.
func (m SidebarModel) Width() int {
	return 22
}
