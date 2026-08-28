package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wenmar-pro/wenmar-sdk/go/pkg/generated"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// VehicleList is the vehicles tab.
type VehicleList struct {
	ListModel[generated.Vehicle]
	detail   *VehicleDetail
	inDetail bool
}

func NewVehicleList(client *wenmar.Client, locationID string) *VehicleList {
	m := &VehicleList{}
	m.client = client
	m.locationID = locationID
	m.loading = true
	return m
}

func (m *VehicleList) Title() string {
	return "Vehicles"
}

func (m *VehicleList) Init() tea.Cmd {
	return fetchVehicles(m.client, m.locationID)
}

func (m *VehicleList) Update(msg tea.Msg) (tab, tea.Cmd) {
	if m.inDetail && m.detail != nil {
		return m.updateDetail(msg)
	}
	return m.updateList(msg)
}

func (m *VehicleList) updateList(msg tea.Msg) (tab, tea.Cmd) {
	switch msg := msg.(type) {
	case vehicleListResultMsg:
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
				m.detail = NewVehicleDetail(m.client, m.locationID, item.Id)
				m.inDetail = true
				return m, m.detail.Init()
			}
		case keyMatches(msg, Keys.Refresh) || keyMatches(msg, Keys.RefreshAlt):
			m.startLoading()
			return m, fetchVehicles(m.client, m.locationID)
		}
	}
	return m, nil
}

func (m *VehicleList) updateDetail(msg tea.Msg) (tab, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if keyMatches(msg, Keys.Back) {
			m.inDetail = false
			m.detail = nil
			return m, nil
		}
	}
	model, cmd := m.detail.Update(msg)
	if dm, ok := model.(*VehicleDetail); ok {
		m.detail = dm
	}
	return m, cmd
}

func (m *VehicleList) View(width int) string {
	if m.inDetail && m.detail != nil {
		return m.detail.View()
	}
	return m.renderList("vehicles", vehicleHeaders, m.vehicleRow)
}

var vehicleHeaders = []string{
	fmt.Sprintf("%-6s %-12s %-12s %-18s %-20s %-10s", "Year", "Make", "Model", "VIN", "Customer", "Open WOs"),
}

func (m *VehicleList) vehicleRow(v generated.Vehicle) []string {
	return []string{fmt.Sprintf("%-6d %-12s %-12s %-18s %-20s %-10d",
		v.Year,
		truncate(v.Make, 12),
		truncate(v.Model, 12),
		truncate(stringify(v.Vin), 18),
		truncate(v.Customer.FullName, 20),
		v.OpenWorkOrdersCount,
	)}
}

type vehicleListResultMsg struct {
	items []generated.Vehicle
	err   error
}

func fetchVehicles(client *wenmar.Client, locationID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var resp *generated.ListVehiclesResponse
		var err error
		if locationID != "" {
			lc, lerr := client.ForLocation(ctx, locationID)
			if lerr != nil {
				return vehicleListResultMsg{err: lerr}
			}
			resp, err = lc.ListVehicles(ctx)
		} else {
			resp, err = client.ListVehicles(ctx)
		}
		if err != nil {
			return vehicleListResultMsg{err: err}
		}
		if resp.JSON200 == nil {
			return vehicleListResultMsg{items: nil}
		}
		return vehicleListResultMsg{items: *resp.JSON200}
	}
}
