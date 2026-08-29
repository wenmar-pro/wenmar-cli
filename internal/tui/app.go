package tui

import (
	"context"
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

	layout         LayoutModel
	accountName    string
	accountFetched bool

	width  int
	height int

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
		layout: NewLayout(),
		width:  80,
		height: 24,
	}
}

// SetInitialWorkOrder configures the app to open a work order detail view
// directly on startup. A value <= 0 disables the behavior.
func (m *AppModel) SetInitialWorkOrder(id int) {
	m.initialWorkOrder = id
}

func (m AppModel) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.tabs)+2)
	for _, t := range m.tabs {
		cmds = append(cmds, t.Init())
	}
	cmds = append(cmds, tick(m.interval))
	if m.client != nil {
		cmds = append(cmds, fetchAccountName(m.client))
	}
	if m.initialWorkOrder > 0 {
		m.active = 0
		if wo, ok := m.tabs[0].(*WorkOrderList); ok {
			cmds = append(cmds, wo.OpenDetail(m.initialWorkOrder))
		}
	}
	return tea.Batch(cmds...)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle account fetch result.
	if r, ok := msg.(accountResultMsg); ok {
		if r.err == nil {
			m.accountName = r.name
			m.accountFetched = true
			m.layout.topBar.SetAccountName(r.name)
		}
		return m, nil
	}

	// Handle search filter message — delegate to active tab if it's
	// the customer list (the only tab with server-side filtering).
	if f, ok := msg.(searchFilterMsg); ok {
		if cl, ok := m.tabs[m.active].(*CustomerList); ok {
			cl.SetSearchQuery(f.query)
			cl.startLoading()
			return m, cl.Init()
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	case tickMsg:
		// Refresh only the active tab to reduce API load.
		return m, m.tabs[m.active].Init()
	case workOrderListResultMsg, customerListResultMsg, vehicleListResultMsg:
		m.updateOnline(msg)
	}

	// If search is focused, route to top bar.
	if m.layout.topBar.searchFocused {
		var cmd tea.Cmd
		m.layout.topBar, cmd = m.layout.topBar.Update(msg)
		return m, cmd
	}

	// If sidebar is visible and not in search, route to sidebar.
	if m.layout.sidebar.visible && !m.layout.topBar.searchFocused {
		var cmd tea.Cmd
		m.layout.sidebar, cmd = m.layout.sidebar.Update(msg)
		return m, cmd
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
	// If search is focused, route all keys to the top bar.
	if m.layout.topBar.searchFocused {
		var cmd tea.Cmd
		m.layout.topBar, cmd = m.layout.topBar.Update(msg)
		return m, cmd
	}

	// If sidebar is visible, route keys to sidebar (except toggle and quit).
	if m.layout.sidebar.visible {
		switch {
		case keyMatches(msg, Keys.SidebarToggle):
			m.layout.sidebar.Toggle()
			return m, nil
		case keyMatches(msg, Keys.Quit):
			return m, tea.Quit
		case keyMatches(msg, Keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		}
		var cmd tea.Cmd
		m.layout.sidebar, cmd = m.layout.sidebar.Update(msg)
		return m, cmd
	}

	switch {
	case keyMatches(msg, Keys.Quit):
		return m, tea.Quit
	case keyMatches(msg, Keys.Help):
		m.showHelp = !m.showHelp
		return m, nil
	case keyMatches(msg, Keys.SidebarToggle):
		m.layout.sidebar.Toggle()
		return m, nil
	case keyMatches(msg, Keys.FocusSearch):
		m.layout.topBar.FocusSearch()
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
	m.layout.SetContent(m.tabs[m.active].View(m.width))
	m.layout.SetFooter(m.renderFooter())
	if m.showHelp {
		m.layout.SetContent(m.tabs[m.active].View(m.width) + "\n" + m.renderHelp())
	}
	return m.layout.View(m.width, m.height)
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
	return HelpStyle.Render(fmt.Sprintf("  %s%s%s  •  ↑↓ move • enter detail • tab switch • ` sidebar • / search • ? help • q quit",
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

type accountResultMsg struct {
	name string
	err  error
}

func fetchAccountName(client *wenmar.Client) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.ListAccount(context.Background())
		if err != nil {
			return accountResultMsg{err: err}
		}
		if resp.JSON200 != nil {
			return accountResultMsg{name: resp.JSON200.Name}
		}
		return accountResultMsg{}
	}
}
