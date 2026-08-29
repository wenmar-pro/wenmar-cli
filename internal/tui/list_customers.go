package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// CustomerList is the customers tab.
type CustomerList struct {
	ListModel[wenmar.Customer]
	detail   *CustomerDetail
	inDetail bool
}

func NewCustomerList(client *wenmar.Client, locationID string) *CustomerList {
	m := &CustomerList{}
	m.client = client
	m.locationID = locationID
	m.loading = true
	return m
}

func (m *CustomerList) Title() string {
	return "Customers"
}

func (m *CustomerList) Init() tea.Cmd {
	return fetchCustomers(m.client, m.locationID)
}

func (m *CustomerList) Update(msg tea.Msg) (tab, tea.Cmd) {
	if m.inDetail && m.detail != nil {
		return m.updateDetail(msg)
	}
	return m.updateList(msg)
}

func (m *CustomerList) updateList(msg tea.Msg) (tab, tea.Cmd) {
	switch msg := msg.(type) {
	case customerListResultMsg:
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
				m.detail = NewCustomerDetail(m.client, m.locationID, item.Id)
				m.inDetail = true
				return m, m.detail.Init()
			}
		case keyMatches(msg, Keys.Refresh) || keyMatches(msg, Keys.RefreshAlt):
			m.startLoading()
			return m, fetchCustomers(m.client, m.locationID)
		}
	}
	return m, nil
}

func (m *CustomerList) updateDetail(msg tea.Msg) (tab, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if keyMatches(msg, Keys.Back) {
			m.inDetail = false
			m.detail = nil
			return m, nil
		}
	}
	model, cmd := m.detail.Update(msg)
	if dm, ok := model.(*CustomerDetail); ok {
		m.detail = dm
	}
	return m, cmd
}

func (m *CustomerList) View(width int) string {
	if m.inDetail && m.detail != nil {
		return m.detail.View()
	}
	return m.renderList("customers", customerHeaders, m.customerRow)
}

var customerHeaders = []string{
	fmt.Sprintf("%-24s %-10s %-10s %-12s %-15s", "Name", "Type", "Vehicles", "Balance", "Updated"),
}

func (m *CustomerList) customerRow(c wenmar.Customer) []string {
	return []string{fmt.Sprintf("%-24s %-10s %-10d %-12s %-15s",
		truncate(c.FullName, 24),
		truncate(c.Type, 10),
		c.VehiclesCount,
		formatCents(c.OutstandingBalanceCents),
		c.UpdatedAt,
	)}
}

type customerListResultMsg struct {
	items []wenmar.Customer
	err   error
}

func fetchCustomers(client *wenmar.Client, locationID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var resp *wenmar.ListCustomersResponse
		var err error
		if locationID != "" {
			lc, lerr := client.ForLocation(ctx, locationID)
			if lerr != nil {
				return customerListResultMsg{err: lerr}
			}
			resp, err = lc.ListCustomers(ctx)
		} else {
			resp, err = client.ListCustomers(ctx)
		}
		if err != nil {
			return customerListResultMsg{err: err}
		}
		if resp.JSON200 == nil {
			return customerListResultMsg{items: nil}
		}
		return customerListResultMsg{items: *resp.JSON200}
	}
}
