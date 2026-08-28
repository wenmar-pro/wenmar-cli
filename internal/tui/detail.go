package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wenmar-pro/wenmar-sdk/go/pkg/generated"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// DetailModel shows a single work order with customer, vehicle, status, and
// timestamps, plus actions (start time tracking, mark complete, refresh).
type DetailModel struct {
	client     *wenmar.Client
	locationID string
	id         int

	wo      *generated.WorkOrder
	loading bool
	err     error
	notice  string
}

// NewDetailModel creates a detail view for the given work order ID.
func NewDetailModel(client *wenmar.Client, locationID string, id int) *DetailModel {
	return &DetailModel{
		client:     client,
		locationID: locationID,
		id:         id,
		loading:    true,
	}
}

func (m *DetailModel) Init() tea.Cmd {
	return fetchWorkOrderDetail(m.client, m.locationID, m.id)
}

func (m *DetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case detailResultMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.wo = msg.wo
		}
	case tea.KeyMsg:
		switch {
		case keyMatches(msg, Keys.Refresh) || keyMatches(msg, Keys.RefreshAlt):
			m.loading = true
			return m, fetchWorkOrderDetail(m.client, m.locationID, m.id)
		case keyMatches(msg, Keys.Complete):
			return m, markWorkOrderComplete(m.client, m.locationID, m.id)
		case keyMatches(msg, Keys.Start):
			m.notice = "Time tracking requires an API endpoint (POST /work_orders/{id}/time_entries) not yet available."
		}
	case completeResultMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.notice = "Work order marked complete."
			return m, fetchWorkOrderDetail(m.client, m.locationID, m.id)
		}
	}
	return m, nil
}

func (m *DetailModel) View() string {
	if m.loading {
		return "  Loading work order...\n"
	}
	if m.err != nil {
		return fmt.Sprintf("  Error: %v\n", m.err)
	}
	if m.wo == nil {
		return "  Work order not found.\n"
	}

	wo := m.wo
	s := TitleStyle.Render(fmt.Sprintf(" Work Order #%d ", wo.WorkOrderNumber)) + "\n\n"

	s += DetailLabelStyle.Render("Status: ") + statusColored(wo.Status) + "\n"
	s += DetailLabelStyle.Render("Intake: ") + DetailValueStyle.Render(wo.IntakeMethod) + "\n\n"

	s += DetailLabelStyle.Render("Customer\n")
	s += fmt.Sprintf("  %s\n", wo.Customer.FullName)
	s += fmt.Sprintf("  id: %d\n\n", wo.Customer.Id)

	s += DetailLabelStyle.Render("Vehicle\n")
	s += fmt.Sprintf("  %d %s %s\n", wo.Vehicle.Year, wo.Vehicle.Make, wo.Vehicle.Model)
	if wo.Vehicle.Vin != "" {
		s += fmt.Sprintf("  VIN: %s\n", wo.Vehicle.Vin)
	}
	s += "\n"

	if wo.Totals.TotalCents > 0 {
		s += DetailLabelStyle.Render("Totals\n")
		s += fmt.Sprintf("  Total: $%.2f\n", float64(wo.Totals.TotalCents)/100)
		s += fmt.Sprintf("  Paid:  $%.2f\n", float64(wo.Totals.PaidCents)/100)
		s += fmt.Sprintf("  Remaining: $%.2f\n\n", float64(wo.Totals.RemainingCents)/100)
	}

	s += DetailLabelStyle.Render("Timestamps\n")
	s += fmt.Sprintf("  Created: %s\n", wo.CreatedAt)
	s += fmt.Sprintf("  Updated: %s\n", wo.UpdatedAt)
	s += "\n"

	if m.notice != "" {
		s += HelpStyle.Render("  "+m.notice) + "\n\n"
	}

	s += HelpStyle.Render("  esc back • r refresh • c mark complete • s start time • q quit") + "\n"
	return s
}

type detailResultMsg struct {
	wo  *generated.WorkOrder
	err error
}

type completeResultMsg struct {
	err error
}

func fetchWorkOrderDetail(client *wenmar.Client, locationID string, id int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var resp *generated.ShowWorkOrderResponse
		var err error
		if locationID != "" {
			lc, lerr := client.ForLocation(ctx, locationID)
			if lerr != nil {
				return detailResultMsg{err: lerr}
			}
			resp, err = lc.ShowWorkOrder(ctx, id)
		} else {
			resp, err = client.ShowWorkOrder(ctx, id)
		}
		if err != nil {
			return detailResultMsg{err: err}
		}
		return detailResultMsg{wo: resp.JSON200}
	}
}

func markWorkOrderComplete(client *wenmar.Client, locationID string, id int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var err error
		if locationID != "" {
			lc, lerr := client.ForLocation(ctx, locationID)
			if lerr != nil {
				return completeResultMsg{err: lerr}
			}
			err = lc.StageTransition(ctx, id, "completed")
		} else {
			err = client.StageTransition(ctx, id, "completed")
		}
		return completeResultMsg{err: err}
	}
}
