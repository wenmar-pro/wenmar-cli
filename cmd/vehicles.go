package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/wenmar-pro/wenmar-cli/internal/output"
	"github.com/spf13/cobra"
)

var vehiclesCmd = &cobra.Command{
	Use:   "vehicles",
	Short: "Manage vehicles",
}

var vehiclesShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single vehicle by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runVehiclesShow,
}

func init() {
	vehiclesCmd.AddCommand(vehiclesShowCmd)
	rootCmd.AddCommand(vehiclesCmd)
}

func runVehiclesShow(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	resp, err := client.ShowVehicle(context.Background(), id)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}
