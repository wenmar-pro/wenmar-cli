package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
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

type BoardModel struct {
	items       []WorkOrderItem
	cursor      int
	baseURL     string
	token       string
	loading     bool
	err         error
	lastRefresh time.Time
}

func NewBoard(baseURL, token string) BoardModel {
	return BoardModel{
		baseURL: baseURL,
		token:   token,
		loading: true,
	}
}

func (m BoardModel) Init() tea.Cmd {
	return fetchWorkOrders(m.baseURL, m.token)
}

func (m BoardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
		case keyMatches(msg, Keys.Refresh):
			m.loading = true
			return m, fetchWorkOrders(m.baseURL, m.token)
		}
	case fetchResultMsg:
		m.loading = false
		m.err = msg.err
		m.items = msg.items
		m.lastRefresh = time.Now()
		if m.cursor >= len(m.items) {
			m.cursor = len(m.items) - 1
		}
	}

	return m, nil
}

func (m BoardModel) View() string {
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
	s += "\n" + HelpStyle.Render("  ↑↓ navigate • r refresh • q quit • ? help") + "\n"
	return s
}

type fetchResultMsg struct {
	items []WorkOrderItem
	err   error
}

func fetchWorkOrders(baseURL, token string) tea.Cmd {
	return func() tea.Msg {
		url := fmt.Sprintf("%s/work_orders", baseURL)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fetchResultMsg{err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return fetchResultMsg{err: fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))}
		}

		var items []WorkOrderItem
		if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
			return fetchResultMsg{err: err}
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
