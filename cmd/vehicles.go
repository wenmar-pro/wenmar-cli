package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/wenmar-pro/wenmar-cli/internal/errors"
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
	Run:   runVehiclesShow,
}

func init() {
	vehiclesCmd.AddCommand(vehiclesShowCmd)
	rootCmd.AddCommand(vehiclesCmd)
}

func runVehiclesShow(cmd *cobra.Command, args []string) {
	client, err := newSDKClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(errors.ExitGeneric)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "id must be an integer")
		os.Exit(errors.ExitGeneric)
	}

	resp, err := client.ShowVehicle(context.Background(), id)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(errors.ExitCode(err))
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag}
	output.Render(os.Stdout, data, "", nil, opts)
}
