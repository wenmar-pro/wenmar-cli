package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// WorkOrderList is the work orders tab. It embeds the generic list model and
// provides work-order-specific fetching, column rendering, and detail
// navigation.
type WorkOrderList struct {
	ListModel[wenmar.WorkOrder]
	detail   *DetailModel
	inDetail bool
}

func NewWorkOrderList(client *wenmar.Client, locationID string) *WorkOrderList {
	m := &WorkOrderList{}
	m.client = client
	m.locationID = locationID
	m.loading = true
	return m
}

func (m *WorkOrderList) Title() string {
	return "Work Orders"
}

// OpenDetail jumps directly to the detail view for the given work order ID.
func (m *WorkOrderList) OpenDetail(id int) tea.Cmd {
	m.detail = NewDetailModel(m.client, m.locationID, id)
	m.inDetail = true
	return m.detail.Init()
}

func (m *WorkOrderList) Init() tea.Cmd {
	return fetchWorkOrders(m.client, m.locationID)
}

func (m *WorkOrderList) Update(msg tea.Msg) (tab, tea.Cmd) {
	if m.inDetail && m.detail != nil {
		return m.updateDetail(msg)
	}
	return m.updateList(msg)
}

func (m *WorkOrderList) updateList(msg tea.Msg) (tab, tea.Cmd) {
	switch msg := msg.(type) {
	case workOrderListResultMsg:
		if msg.err != nil {
			m.setError(msg.err)
		} else {
			m.setItems(msg.items)
		}
		return m, nil
	case tea.KeyMsg:
		switch {
		case keyMatches(msg, Keys.Down):
			m.moveDown()
		case keyMatches(msg, Keys.Up):
			m.moveUp()
		case keyMatches(msg, Keys.Enter):
			if item, ok := m.selected(); ok {
				m.detail = NewDetailModel(m.client, m.locationID, item.Id)
				m.inDetail = true
				return m, m.detail.Init()
			}
		case keyMatches(msg, Keys.Refresh) || keyMatches(msg, Keys.RefreshAlt):
			m.startLoading()
			return m, fetchWorkOrders(m.client, m.locationID)
		}
	}
	return m, nil
}

func (m *WorkOrderList) updateDetail(msg tea.Msg) (tab, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if keyMatches(msg, Keys.Back) {
			m.inDetail = false
			m.detail = nil
			return m, nil
		}
	}
	model, cmd := m.detail.Update(msg)
	if dm, ok := model.(*DetailModel); ok {
		m.detail = dm
	}
	return m, cmd
}

func (m *WorkOrderList) View(width int) string {
	if m.inDetail && m.detail != nil {
		return m.detail.View()
	}
	return m.renderList("work orders", workOrderHeaders, m.workOrderRow)
}

var workOrderHeaders = []string{
	fmt.Sprintf("%-8s %-20s %-20s %-12s %-15s", "WO#", "Customer", "Vehicle", "Status", "Updated"),
}

func (m *WorkOrderList) workOrderRow(wo wenmar.WorkOrder) []string {
	return []string{fmt.Sprintf("%-8d %-20s %-20s %-12s %-15s",
		wo.WorkOrderNumber,
		truncate(wo.Customer.FullName, 20),
		truncate(fmt.Sprintf("%s %s", wo.Vehicle.Make, wo.Vehicle.Model), 20),
		statusColored(wo.Status),
		m.refreshed.Format("15:04:05"),
	)}
}

type workOrderListResultMsg struct {
	items []wenmar.WorkOrder
	err   error
}

// fetchWorkOrders fetches the work order list through the SDK client,
// populating the nested customer and vehicle fields.
func fetchWorkOrders(client *wenmar.Client, locationID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var resp *wenmar.ListWorkOrdersResponse
		var err error
		if locationID != "" {
			lc, lerr := client.ForLocation(ctx, locationID)
			if lerr != nil {
				return workOrderListResultMsg{err: lerr}
			}
			resp, err = lc.ListWorkOrders(ctx)
		} else {
			resp, err = client.ListWorkOrders(ctx)
		}
		if err != nil {
			return workOrderListResultMsg{err: err}
		}
		if resp.JSON200 == nil {
			return workOrderListResultMsg{items: nil}
		}
		return workOrderListResultMsg{items: *resp.JSON200}
	}
}
