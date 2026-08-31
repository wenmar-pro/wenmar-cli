package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// CustomerDetail shows a single customer record.
type CustomerDetail struct {
	client     *wenmar.Client
	locationID string
	id         int

	customer *wenmar.Customer
	loading  bool
	err      error
}

func NewCustomerDetail(client *wenmar.Client, locationID string, id int) *CustomerDetail {
	return &CustomerDetail{
		client:     client,
		locationID: locationID,
		id:         id,
		loading:    true,
	}
}

func (m *CustomerDetail) Init() tea.Cmd {
	return fetchCustomerDetail(m.client, m.locationID, m.id)
}

func (m *CustomerDetail) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case customerDetailResultMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.customer = msg.customer
		}
	case tea.KeyMsg:
		switch {
		case keyMatches(msg, Keys.Refresh) || keyMatches(msg, Keys.RefreshAlt):
			m.loading = true
			return m, fetchCustomerDetail(m.client, m.locationID, m.id)
		}
	}
	return m, nil
}

func (m *CustomerDetail) View() string {
	if m.loading {
		return "  Loading customer...\n"
	}
	if m.err != nil {
		return fmt.Sprintf("  Error: %v\n", m.err)
	}
	if m.customer == nil {
		return "  Customer not found.\n"
	}

	c := m.customer
	s := TitleStyle.Render(fmt.Sprintf(" Customer #%d ", c.Id)) + "\n\n"

	s += DetailLabelStyle.Render("Name: ") + DetailValueStyle.Render(c.FullName) + "\n"
	s += DetailLabelStyle.Render("Type: ") + DetailValueStyle.Render(c.Type) + "\n"
	if c.CompanyName != nil {
		s += DetailLabelStyle.Render("Company: ") + DetailValueStyle.Render(stringify(c.CompanyName)) + "\n"
	}
	s += "\n"

	s += DetailLabelStyle.Render("Contact\n")
	s += fmt.Sprintf("  Emails: %d\n", c.EmailsCount)
	s += fmt.Sprintf("  Phones: %d\n", c.PhonesCount)
	s += fmt.Sprintf("  Addresses: %d\n", len(c.Addresses))
	s += "\n"

	s += DetailLabelStyle.Render("Location\n")
	s += fmt.Sprintf("  %s\n", c.Location.Name)
	if c.Location.Url != "" {
		s += fmt.Sprintf("  %s\n", c.Location.Url)
	}
	s += "\n"

	s += DetailLabelStyle.Render("Financials\n")
	s += fmt.Sprintf("  Outstanding balance: %s\n", formatCents(c.OutstandingBalanceCents))
	s += fmt.Sprintf("  Store credit: %s\n", formatCents(c.StoreCreditCents))
	s += fmt.Sprintf("  Total revenue: %s\n", formatCents(c.TotalRevenueCents))
	s += "\n"

	s += DetailLabelStyle.Render("Details\n")
	s += fmt.Sprintf("  Vehicles: %d\n", c.VehiclesCount)
	if c.WorkOrdersUrl != "" {
		s += fmt.Sprintf("  Work orders: %s\n", c.WorkOrdersUrl)
	}
	s += fmt.Sprintf("  Marketing opt-in: %t\n", c.MarketingOptIn)
	s += fmt.Sprintf("  Tax exempt: %t\n", c.TaxExempt)
	s += "\n"

	s += DetailLabelStyle.Render("Timestamps\n")
	s += fmt.Sprintf("  Created: %s\n", c.CreatedAt)
	s += fmt.Sprintf("  Updated: %s\n", c.UpdatedAt)
	s += "\n"

	s += HelpStyle.Render("  esc back • r refresh • q quit") + "\n"
	return s
}

type customerDetailResultMsg struct {
	customer *wenmar.Customer
	err      error
}

func fetchCustomerDetail(client *wenmar.Client, locationID string, id int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var resp *wenmar.ShowCustomerResponse
		var err error
		if locationID != "" {
			lc := client.ForLocation(locationID)
			resp, err = lc.ShowCustomer(ctx, id)
		} else {
			resp, err = client.ShowCustomer(ctx, id)
		}
		if err != nil {
			return customerDetailResultMsg{err: err}
		}
		return customerDetailResultMsg{customer: resp.JSON200}
	}
}
