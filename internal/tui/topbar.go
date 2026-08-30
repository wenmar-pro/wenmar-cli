package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// searchFilterMsg is emitted when the search field value changes. The active
// list refetches with the new query string.
type searchFilterMsg struct {
	query string
}

// TopBarModel renders the top bar: search field, quick actions, messages
// badge (stub), notifications badge (stub), and account name.
type TopBarModel struct {
	search        textinput.Model
	searchFocused bool
	accountName   string
}

// NewTopBar creates the top bar with an unfocused search field.
func NewTopBar() TopBarModel {
	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.CharLimit = 80
	ti.Width = 30
	return TopBarModel{
		search: ti,
	}
}

// SetAccountName sets the account name shown on the right side of the bar.
func (m *TopBarModel) SetAccountName(name string) {
	m.accountName = name
}

// SearchValue returns the current search field text.
func (m TopBarModel) SearchValue() string {
	return m.search.Value()
}

// FocusSearch focuses the search field.
func (m *TopBarModel) FocusSearch() {
	m.searchFocused = true
	m.search.Focus()
}

// UnfocusSearch blurs the search field.
func (m *TopBarModel) UnfocusSearch() {
	m.searchFocused = false
	m.search.Blur()
}

// Update handles key messages. When search is focused, all keys go to the
// textinput. Otherwise, "/" focuses search and Escape unfocuses it.
func (m TopBarModel) Update(msg tea.Msg) (TopBarModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searchFocused {
			switch msg.Type {
			case tea.KeyEscape:
				m.UnfocusSearch()
				return m, nil
			case tea.KeyEnter:
				m.UnfocusSearch()
				return m, m.emitFilter()
			default:
				var cmd tea.Cmd
				m.search, cmd = m.search.Update(msg)
				return m, tea.Batch(cmd, m.emitFilter())
			}
		}
		// Not focused: "/" focuses search.
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 && msg.Runes[0] == '/' {
			m.FocusSearch()
			return m, nil
		}
	}
	return m, nil
}

// emitFilter returns a command that emits a searchFilterMsg after a short
// debounce. This prevents refetching on every keystroke.
func (m TopBarModel) emitFilter() tea.Cmd {
	query := m.search.Value()
	return tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
		return searchFilterMsg{query: query}
	})
}

// View renders the top bar as a single line. The width parameter is the
// total available width for the bar.
func (m TopBarModel) View(width int) string {
	var b strings.Builder

	// Search field segment.
	searchPart := m.search.View()
	if !m.searchFocused {
		searchPart = SearchBarStyle.Render("/ " + searchPart)
	} else {
		searchPart = SearchBarFocusStyle.Render("/ " + searchPart)
	}
	b.WriteString(searchPart)

	// Quick actions segment.
	quickActions := QuickActionStyle.Render("  [n new] [r refresh]")
	b.WriteString(quickActions)

	// Badges (stubs).
	badges := BadgeStyle.Render("  📨  🔔")
	b.WriteString(badges)

	// Account name on the right.
	userPart := UserStyle.Render("  " + m.accountName)
	b.WriteString(userPart)

	return TopBarStyle.Render(b.String())
}
