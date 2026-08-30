package cmd

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/auth"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	"github.com/wenmar-pro/wenmar-cli/internal/tui"
)

var (
	tuiLocation  string
	tuiInterval  time.Duration
	tuiWorkOrder int
	tuiRemote    bool
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the TUI dispatch board",
	RunE:  runTUI,
}

func init() {
	tuiCmd.Flags().StringVar(&tuiLocation, "location", "", "Location ID to scope requests")
	tuiCmd.Flags().DurationVar(&tuiInterval, "interval", 10*time.Second, "Polling interval")
	tuiCmd.Flags().IntVar(&tuiWorkOrder, "work-order", 0, "Jump to a work order detail view")
	tuiCmd.Flags().BoolVar(&tuiRemote, "remote", false, "Desktop integration view spec")
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	configPath := configPathFlag
	if configPath == "" {
		p, err := config.ConfigPath()
		if err == nil {
			configPath = p
		}
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	locationID := tuiLocation
	if locationID == "" {
		locationID = auth.ResolveLocationID(locationFlag, configPath)
	}

	model := tui.NewApp(client, locationID, tuiInterval)
	model.SetInitialWorkOrder(tuiWorkOrder)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
