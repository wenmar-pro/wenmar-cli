package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// VehicleDetail shows a single vehicle record.
type VehicleDetail struct {
	client     *wenmar.Client
	locationID string
	id         int

	vehicle *wenmar.Vehicle
	loading bool
	err     error
}

func NewVehicleDetail(client *wenmar.Client, locationID string, id int) *VehicleDetail {
	return &VehicleDetail{
		client:     client,
		locationID: locationID,
		id:         id,
		loading:    true,
	}
}

func (m *VehicleDetail) Init() tea.Cmd {
	return fetchVehicleDetail(m.client, m.locationID, m.id)
}

func (m *VehicleDetail) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case vehicleDetailResultMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.vehicle = msg.vehicle
		}
	case tea.KeyMsg:
		switch {
		case keyMatches(msg, Keys.Refresh) || keyMatches(msg, Keys.RefreshAlt):
			m.loading = true
			return m, fetchVehicleDetail(m.client, m.locationID, m.id)
		}
	}
	return m, nil
}

func (m *VehicleDetail) View() string {
	if m.loading {
		return "  Loading vehicle...\n"
	}
	if m.err != nil {
		return fmt.Sprintf("  Error: %v\n", m.err)
	}
	if m.vehicle == nil {
		return "  Vehicle not found.\n"
	}

	v := m.vehicle
	s := TitleStyle.Render(fmt.Sprintf(" Vehicle #%d ", v.Id)) + "\n\n"

	s += DetailLabelStyle.Render("Vehicle\n")
	s += fmt.Sprintf("  %d %s %s\n", v.Year, v.Make, v.Model)
	if v.Submodel != nil {
		s += fmt.Sprintf("  Submodel: %s\n", stringify(v.Submodel))
	}
	if v.Vin != nil {
		s += fmt.Sprintf("  VIN: %s\n", stringify(v.Vin))
	}
	if v.LicensePlate != nil {
		s += fmt.Sprintf("  License plate: %s (%s %s)\n", stringify(v.LicensePlate), v.LicensePlateCountry, stringify(v.LicensePlateState))
	}
	if v.Color != nil {
		s += fmt.Sprintf("  Color: %s\n", stringify(v.Color))
	}
	if v.BodyStyle != nil {
		s += fmt.Sprintf("  Body style: %s\n", stringify(v.BodyStyle))
	}
	if v.Drivetrain != nil {
		s += fmt.Sprintf("  Drivetrain: %s\n", stringify(v.Drivetrain))
	}
	if v.Engine != nil {
		s += fmt.Sprintf("  Engine: %s\n", stringify(v.Engine))
	}
	if v.Transmission != nil {
		s += fmt.Sprintf("  Transmission: %s\n", stringify(v.Transmission))
	}
	if v.Odometer.Reading != nil {
		s += fmt.Sprintf("  Odometer: %s %s\n", stringify(v.Odometer.Reading), v.Odometer.Unit)
	}
	s += "\n"

	s += DetailLabelStyle.Render("Customer\n")
	s += fmt.Sprintf("  %s\n", v.Customer.FullName)
	s += fmt.Sprintf("  id: %d\n", v.Customer.Id)
	if v.Customer.Url != "" {
		s += fmt.Sprintf("  %s\n", v.Customer.Url)
	}
	s += "\n"

	s += DetailLabelStyle.Render("Location\n")
	s += fmt.Sprintf("  %s\n", v.Location.Name)
	if v.Location.Url != "" {
		s += fmt.Sprintf("  %s\n", v.Location.Url)
	}
	s += "\n"

	s += DetailLabelStyle.Render("Stats\n")
	s += fmt.Sprintf("  Lifetime revenue: %s\n", formatCents(v.LifetimeRevenueCents))
	s += fmt.Sprintf("  Open work orders: %d\n", v.OpenWorkOrdersCount)
	s += fmt.Sprintf("  Appointments: %d\n", v.AppointmentsCount)
	if v.AnnualSafetyExpiresAt != nil {
		s += fmt.Sprintf("  Annual safety expires: %s\n", stringify(v.AnnualSafetyExpiresAt))
	}
	if v.LastServicedAt != nil {
		s += fmt.Sprintf("  Last serviced: %s\n", stringify(v.LastServicedAt))
	}
	s += "\n"

	s += DetailLabelStyle.Render("Details\n")
	s += fmt.Sprintf("  Type: %s\n", v.Type)
	s += fmt.Sprintf("  Vehicle type: %s\n", v.VehicleType)
	if v.UnitNumber != nil {
		s += fmt.Sprintf("  Unit number: %s\n", stringify(v.UnitNumber))
	}
	if v.FleetIdentifier != nil {
		s += fmt.Sprintf("  Fleet identifier: %s\n", stringify(v.FleetIdentifier))
	}
	s += "\n"

	s += DetailLabelStyle.Render("Timestamps\n")
	s += fmt.Sprintf("  Created: %s\n", v.CreatedAt)
	s += fmt.Sprintf("  Updated: %s\n", v.UpdatedAt)
	s += "\n"

	s += HelpStyle.Render("  esc back • r refresh • q quit") + "\n"
	return s
}

type vehicleDetailResultMsg struct {
	vehicle *wenmar.Vehicle
	err     error
}

func fetchVehicleDetail(client *wenmar.Client, locationID string, id int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var resp *wenmar.ShowVehicleResponse
		var err error
		if locationID != "" {
			lc, lerr := client.ForLocation(ctx, locationID)
			if lerr != nil {
				return vehicleDetailResultMsg{err: lerr}
			}
			resp, err = lc.ShowVehicle(ctx, id)
		} else {
			resp, err = client.ShowVehicle(ctx, id)
		}
		if err != nil {
			return vehicleDetailResultMsg{err: err}
		}
		return vehicleDetailResultMsg{vehicle: resp.JSON200}
	}
}
