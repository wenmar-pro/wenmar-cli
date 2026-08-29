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

	TabActiveStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(accentColor).
		Padding(0, 1)

	TabInactiveStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Padding(0, 1)

	TabBarStyle = lipgloss.NewStyle().
		MarginBottom(1)

	TopBarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("15")).
		Padding(0, 1)

	SearchBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	SearchBarFocusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("24"))

	SidebarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("15")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	SidebarActiveItemStyle = lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color("36")).
		Foreground(lipgloss.Color("15"))

	SidebarInactiveItemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	BadgeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	QuickActionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("36"))

	UserStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))
)
