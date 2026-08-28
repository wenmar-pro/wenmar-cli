package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/auth"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	"github.com/wenmar-pro/wenmar-cli/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the TUI dispatch board",
	RunE:  runTUI,
}

func init() {
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

	token, err := auth.ResolveTokenFrom(tokenFlag, configPath)
	if err != nil {
		return err
	}
	baseURL := auth.ResolveBaseURLFrom(baseURLFlag, configPath)

	model := tui.NewBoard(baseURL, token)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
