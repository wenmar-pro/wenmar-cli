package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/wenmar-pro/wenmar-cli/internal/output"
	"github.com/spf13/cobra"
)

var workOrdersCmd = &cobra.Command{
	Use:     "work_orders",
	Aliases: []string{"wo"},
	Short:   "Manage work orders",
}

var workOrdersListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all work orders, paginated via the Link header",
	RunE:    runWorkOrdersList,
}

var workOrdersShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single work order by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrdersShow,
}

func init() {
	workOrdersListCmd.Flags().Int("page", 0, "Page number")
	workOrdersCmd.AddCommand(workOrdersListCmd, workOrdersShowCmd)
	rootCmd.AddCommand(workOrdersCmd)
}

func runWorkOrdersList(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}

	pageFlag, _ := cmd.Flags().GetInt("page")
	var pagePtr *int
	if pageFlag > 0 {
		pagePtr = &pageFlag
	}

	resp, paginator, err := client.ListWorkOrdersWithPagination(context.Background(), pagePtr)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	summary := fmt.Sprintf("Page 1. More results: %v", paginator.HasNext())
	meta := &output.Meta{HasNext: paginator.HasNext()}

	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag}
	return output.Render(cmd.OutOrStdout(), data, summary, meta, opts)
}

func runWorkOrdersShow(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	resp, err := client.ShowWorkOrder(context.Background(), id)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}
