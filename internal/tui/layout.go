package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// LayoutModel composes the top bar, content area, and footer into a single
// rendered string. The sidebar overlays the content when visible.
type LayoutModel struct {
	topBar  TopBarModel
	sidebar SidebarModel
	content string
	footer  string
}

// NewLayout creates an empty layout with a top bar and sidebar.
func NewLayout() LayoutModel {
	return LayoutModel{
		topBar:  NewTopBar(),
		sidebar: NewSidebar(),
	}
}

// SetContent sets the main content string (already rendered by the active tab).
func (m *LayoutModel) SetContent(s string) {
	m.content = s
}

// SetFooter sets the footer string (already rendered).
func (m *LayoutModel) SetFooter(s string) {
	m.footer = s
}

// TopBar returns a pointer to the top bar model for updates.
func (m *LayoutModel) TopBar() *TopBarModel {
	return &m.topBar
}

// Sidebar returns a pointer to the sidebar model for updates.
func (m *LayoutModel) Sidebar() *SidebarModel {
	return &m.sidebar
}

// View renders the full layout. width and height are the terminal dimensions.
func (m LayoutModel) View(width, height int) string {
	topBarView := m.topBar.View(width)

	// Content area: when sidebar is visible, overlay it on the left of
	// the content by joining horizontally. The sidebar takes its Width()
	// and the content gets the rest.
	var body string
	if m.sidebar.visible {
		sidebarView := m.sidebar.View(m.sidebar.Width())
		body = lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, m.content)
	} else {
		body = m.content
	}

	return lipgloss.JoinVertical(lipgloss.Left, topBarView, body, m.footer)
}
