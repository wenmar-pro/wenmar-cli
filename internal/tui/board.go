package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wenmar-pro/wenmar-sdk/go/pkg/generated"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// WorkOrderItem is a minimal work order representation for the TUI.
type WorkOrderItem struct {
	ID              int `json:"id"`
	WorkOrderNumber int `json:"work_order_number"`
	Status          string `json:"status"`
	Customer        struct {
		FullName string `json:"full_name"`
	} `json:"customer"`
	Vehicle struct {
		Make  string `json:"make"`
		Model string `json:"model"`
	} `json:"vehicle"`
}

// view is the current TUI screen.
type view int

const (
	viewList view = iota
	viewDetail
)

type BoardModel struct {
	client     *wenmar.Client
	locationID string
	interval   time.Duration

	items       []WorkOrderItem
	cursor      int
	loading     bool
	err         error
	lastRefresh time.Time
	online      bool

	view   view
	detail *DetailModel
}

// NewBoard creates a dispatch board backed by the SDK client. If locationID
// is non-empty, requests are scoped to that location.
func NewBoard(client *wenmar.Client, locationID string, interval time.Duration) BoardModel {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return BoardModel{
		client:     client,
		locationID: locationID,
		interval:   interval,
		loading:    true,
		view:       viewList,
	}
}

func (m BoardModel) Init() tea.Cmd {
	return tea.Batch(fetchWorkOrders(m.client, m.locationID), tick(m.interval))
}

func (m BoardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.view == viewDetail && m.detail != nil {
			return m.updateDetail(msg)
		}
		return m.updateList(msg)
	case fetchResultMsg:
		m.loading = false
		m.err = msg.err
		m.online = msg.err == nil
		if msg.err == nil {
			m.items = msg.items
			m.lastRefresh = time.Now()
			if m.cursor >= len(m.items) {
				m.cursor = len(m.items) - 1
			}
		}
		return m, nil
	case tickMsg:
		return m, fetchWorkOrders(m.client, m.locationID)
	}
	return m, nil
}

func (m BoardModel) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, Keys.Quit):
		return m, tea.Quit
	case keyMatches(msg, Keys.Down):
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case keyMatches(msg, Keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case keyMatches(msg, Keys.Enter):
		if len(m.items) > 0 {
			item := m.items[m.cursor]
			m.detail = NewDetailModel(m.client, m.locationID, item.ID)
			m.view = viewDetail
			return m, m.detail.Init()
		}
	case keyMatches(msg, Keys.Refresh) || keyMatches(msg, Keys.RefreshAlt):
		m.loading = true
		return m, fetchWorkOrders(m.client, m.locationID)
	}
	return m, nil
}

func (m BoardModel) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, Keys.Back):
		m.view = viewList
		m.detail = nil
		return m, nil
	case keyMatches(msg, Keys.Quit):
		return m, tea.Quit
	}
	if m.detail != nil {
		return m.detail.Update(msg)
	}
	return m, nil
}

func (m BoardModel) View() string {
	if m.view == viewDetail && m.detail != nil {
		return m.detail.View()
	}
	return m.viewList()
}

func (m BoardModel) viewList() string {
	if m.loading {
		return "  Loading work orders...\n"
	}
	if m.err != nil {
		return fmt.Sprintf("  Error: %v\n", m.err)
	}
	if len(m.items) == 0 {
		return "  No work orders found.\n"
	}

	// Header
	header := fmt.Sprintf("  %-8s %-20s %-20s %-12s %-15s",
		"WO#", "Customer", "Vehicle", "Status", "Updated")
	s := HeaderStyle.Render(header) + "\n"

	// Rows
	for i, item := range m.items {
		line := fmt.Sprintf("  %-8d %-20s %-20s %-12s %-15s",
			item.WorkOrderNumber,
			truncate(item.Customer.FullName, 20),
			truncate(fmt.Sprintf("%s %s", item.Vehicle.Make, item.Vehicle.Model), 20),
			statusColored(item.Status),
			m.lastRefresh.Format("15:04:05"),
		)

		if i == m.cursor {
			line = SelectedStyle.Render(line)
		}
		s += line + "\n"
	}

	// Footer
	status := "● offline"
	if m.online {
		status = "● online"
	}
	statusStyle := StatusOffline
	if m.online {
		statusStyle = StatusOnline
	}
	s += "\n" + HelpStyle.Render(fmt.Sprintf("  %s  last refresh %s  •  ↑↓ navigate • enter detail • r refresh • q quit • ? help",
		statusStyle.Render(status), m.lastRefresh.Format("15:04:05"))) + "\n"
	return s
}

type fetchResultMsg struct {
	items []WorkOrderItem
	err   error
}

type tickMsg struct{}

func tick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

// fetchWorkOrders fetches the work order list through the SDK client.
func fetchWorkOrders(client *wenmar.Client, locationID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var resp *generated.ListWorkOrdersResponse
		var err error
		if locationID != "" {
			lc, lerr := client.ForLocation(ctx, locationID)
			if lerr != nil {
				return fetchResultMsg{err: lerr}
			}
			resp, err = lc.ListWorkOrders(ctx)
		} else {
			resp, err = client.ListWorkOrders(ctx)
		}
		if err != nil {
			return fetchResultMsg{err: err}
		}
		if resp.JSON200 == nil {
			return fetchResultMsg{items: nil}
		}
		items := make([]WorkOrderItem, 0, len(*resp.JSON200))
		for _, wo := range *resp.JSON200 {
			items = append(items, WorkOrderItem{
				ID:              wo.Id,
				WorkOrderNumber: wo.WorkOrderNumber,
				Status:          wo.Status,
			})
		}
		return fetchResultMsg{items: items}
	}
}

func keyMatches(msg tea.KeyMsg, binding key.Binding) bool {
	for _, k := range binding.Keys() {
		if msg.String() == k {
			return true
		}
	}
	return false
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func statusColored(status string) string {
	switch status {
	case "pending":
		return StatusPending.Render(status)
	case "in_progress":
		return StatusInProgress.Render(status)
	case "completed":
		return StatusCompleted.Render(status)
	default:
		return status
	}
}
