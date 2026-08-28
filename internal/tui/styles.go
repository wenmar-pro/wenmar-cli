package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	accentColor = lipgloss.Color("36")  // cyan
	mutedColor  = lipgloss.Color("245") // gray
	errorColor  = lipgloss.Color("196") // red
	warnColor   = lipgloss.Color("214") // orange

	// Styles
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(accentColor).
		Padding(0, 1)

	HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(mutedColor)

	RowStyle = lipgloss.NewStyle()

	SelectedStyle = lipgloss.NewStyle().
		Background(accentColor).
		Foreground(lipgloss.Color("15"))

	StatusPending    = lipgloss.NewStyle().Foreground(mutedColor)
	StatusInProgress = lipgloss.NewStyle().Foreground(warnColor)
	StatusCompleted  = lipgloss.NewStyle().Foreground(lipgloss.Color("34")) // green

	StatusOnline  = lipgloss.NewStyle().Foreground(lipgloss.Color("34")) // green
	StatusOffline = lipgloss.NewStyle().Foreground(errorColor)           // red

	DetailLabelStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(accentColor)

	DetailValueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("15"))

	BorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mutedColor).
		Padding(1, 2)

	HelpStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true)
)
