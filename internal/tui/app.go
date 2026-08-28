package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// tab is the interface implemented by each resource tab. Tabs own their list
// and detail views internally; AppModel only routes messages and renders the
// tab bar and shared footer.
type tab interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (tab, tea.Cmd)
	View(width int) string
	Title() string
}

// AppModel is the top-level TUI model. It owns the tab bar and routes
// messages to the active tab.
type AppModel struct {
	client     *wenmar.Client
	locationID string
	interval   time.Duration

	tabs   []tab
	active int

	online      bool
	lastRefresh time.Time
	showHelp    bool

	initialWorkOrder int
}

// NewApp creates the tabbed TUI. If locationID is non-empty, requests are
// scoped to that location.
func NewApp(client *wenmar.Client, locationID string, interval time.Duration) AppModel {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return AppModel{
		client:     client,
		locationID: locationID,
		interval:   interval,
		tabs: []tab{
			NewWorkOrderList(client, locationID),
			NewCustomerList(client, locationID),
			NewVehicleList(client, locationID),
		},
	}
}

// SetInitialWorkOrder configures the app to open a work order detail view
// directly on startup. A value <= 0 disables the behavior.
func (m *AppModel) SetInitialWorkOrder(id int) {
	m.initialWorkOrder = id
}

func (m AppModel) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.tabs)+1)
	for _, t := range m.tabs {
		cmds = append(cmds, t.Init())
	}
	cmds = append(cmds, tick(m.interval))
	if m.initialWorkOrder > 0 {
		m.active = 0
		if wo, ok := m.tabs[0].(*WorkOrderList); ok {
			cmds = append(cmds, wo.OpenDetail(m.initialWorkOrder))
		}
	}
	return tea.Batch(cmds...)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.updateKey(msg)
	case tickMsg:
		// Refresh only the active tab to reduce API load.
		return m, m.tabs[m.active].Init()
	case workOrderListResultMsg, customerListResultMsg, vehicleListResultMsg:
		m.updateOnline(msg)
	}
	// Delegate non-key messages to the active tab.
	updated, cmd := m.tabs[m.active].Update(msg)
	m.tabs[m.active] = updated
	return m, cmd
}

// updateOnline updates the shared footer status based on a fetch result.
func (m *AppModel) updateOnline(msg tea.Msg) {
	switch msg := msg.(type) {
	case workOrderListResultMsg:
		m.online = msg.err == nil
		if msg.err == nil {
			m.lastRefresh = time.Now()
		}
	case customerListResultMsg:
		m.online = msg.err == nil
		if msg.err == nil {
			m.lastRefresh = time.Now()
		}
	case vehicleListResultMsg:
		m.online = msg.err == nil
		if msg.err == nil {
			m.lastRefresh = time.Now()
		}
	}
}

func (m AppModel) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, Keys.Quit):
		return m, tea.Quit
	case keyMatches(msg, Keys.Help):
		m.showHelp = !m.showHelp
		return m, nil
	case keyMatches(msg, Keys.Tab):
		m.active = (m.active + 1) % len(m.tabs)
		return m, nil
	case keyMatches(msg, Keys.ShiftTab):
		m.active = (m.active - 1 + len(m.tabs)) % len(m.tabs)
		return m, nil
	case keyMatches(msg, Keys.Tab1):
		m.active = 0
		return m, nil
	case keyMatches(msg, Keys.Tab2):
		m.active = 1
		return m, nil
	case keyMatches(msg, Keys.Tab3):
		m.active = 2
		return m, nil
	}
	// Delegate to the active tab.
	updated, cmd := m.tabs[m.active].Update(msg)
	m.tabs[m.active] = updated
	return m, cmd
}

func (m AppModel) View() string {
	var s strings.Builder
	s.WriteString(m.renderTabBar())
	s.WriteString(m.tabs[m.active].View(0))
	if m.showHelp {
		s.WriteString("\n" + m.renderHelp())
	}
	s.WriteString("\n" + m.renderFooter())
	return s.String()
}

func (m AppModel) renderTabBar() string {
	var b strings.Builder
	for i, t := range m.tabs {
		label := t.Title()
		if i == m.active {
			b.WriteString(TabActiveStyle.Render(" " + label + " "))
		} else {
			b.WriteString(TabInactiveStyle.Render(" " + label + " "))
		}
	}
	return TabBarStyle.Render(b.String()) + "\n"
}

func (m AppModel) renderFooter() string {
	status := "● offline"
	statusStyle := StatusOffline
	if m.online {
		status = "● online"
		statusStyle = StatusOnline
	}
	loc := ""
	if m.locationID != "" {
		loc = "  " + m.locationID
	}
	refresh := ""
	if !m.lastRefresh.IsZero() {
		refresh = "  last refresh " + m.lastRefresh.Format("15:04:05")
	}
	return HelpStyle.Render(fmt.Sprintf("  %s%s%s  •  ↑↓ move • enter detail • tab switch • r refresh • ? help • q quit",
		statusStyle.Render(status), loc, refresh))
}

func (m AppModel) renderHelp() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("  Key Bindings") + "\n")
	rows := [][2]string{
		{"↑/k ↓/j", "move cursor"},
		{"enter", "open detail"},
		{"esc/backspace", "back to list"},
		{"tab / shift+tab", "cycle tabs"},
		{"1 / 2 / 3", "jump to tab"},
		{"r / ctrl+r", "refresh active tab"},
		{"/", "filter (coming soon)"},
		{"?", "toggle help"},
		{"q / ctrl+c", "quit"},
	}
	for _, r := range rows {
		b.WriteString(fmt.Sprintf("  %-18s %s\n", r[0], r[1]))
	}
	return b.String()
}
