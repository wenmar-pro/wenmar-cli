package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/wenmar-pro/wenmar-cli/internal/output"
	"github.com/wenmar-pro/wenmar-sdk/go/pkg/generated"
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

var workOrdersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new work order",
	RunE:  runWorkOrdersCreate,
}

var workOrdersUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a work order by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrdersUpdate,
}

var workOrdersDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a work order by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrdersDelete,
}

var (
	workOrderCreateCustomer     int
	workOrderCreateVehicle      int
	workOrderUpdateIntakeMethod string
	workOrdersDeleteDryRun      bool
)

func init() {
	workOrdersListCmd.Flags().Int("page", 0, "Page number")

	workOrdersCreateCmd.Flags().IntVar(&workOrderCreateCustomer, "customer-id", 0, "Customer ID (required)")
	workOrdersCreateCmd.Flags().IntVar(&workOrderCreateVehicle, "vehicle-id", 0, "Vehicle ID (required)")
	workOrdersCreateCmd.MarkFlagRequired("customer-id")
	workOrdersCreateCmd.MarkFlagRequired("vehicle-id")

	workOrdersUpdateCmd.Flags().StringVar(&workOrderUpdateIntakeMethod, "intake-method", "", "Intake method (e.g. drop_off, walk_in)")
	workOrdersDeleteCmd.Flags().BoolVar(&workOrdersDeleteDryRun, "dry-run", false, "Preview what would be deleted without making an API call")

	workOrdersCmd.AddCommand(workOrdersListCmd, workOrdersShowCmd, workOrdersCreateCmd, workOrdersUpdateCmd, workOrdersDeleteCmd)
	rootCmd.AddCommand(workOrdersCmd)
}

func runWorkOrdersList(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}
	setRequest("GET", "/work_orders")

	resp, paginator, err := client.ListWorkOrdersWithPagination(context.Background())
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	summary := fmt.Sprintf("Page 1. More results: %v", paginator.HasNext())
	meta := &output.Meta{HasNext: paginator.HasNext()}

	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), data, summary, meta, opts)
}

func runWorkOrdersShow(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}
	setRequest("GET", "/work_orders/"+args[0])

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	resp, err := client.ShowWorkOrder(context.Background(), id)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runWorkOrdersCreate(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}
	setRequest("POST", "/work_orders")

	body := generated.CreateWorkOrderJSONRequestBody{
		WorkOrder: struct {
			CustomerId int `json:"customer_id"`
			VehicleId  int `json:"vehicle_id"`
		}{
			CustomerId: workOrderCreateCustomer,
			VehicleId:  workOrderCreateVehicle,
		},
	}

	resp, err := client.CreateWorkOrder(context.Background(), body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON201)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), data, "Work order created.", nil, opts)
}

func runWorkOrdersUpdate(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}
	setRequest("PATCH", "/work_orders/"+args[0])

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	body := generated.UpdateWorkOrderJSONRequestBody{
		WorkOrder: struct {
			IntakeMethod string `json:"intake_method"`
		}{
			IntakeMethod: workOrderUpdateIntakeMethod,
		},
	}

	resp, err := client.UpdateWorkOrder(context.Background(), id, body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), data, "Work order updated.", nil, opts)
}

func runWorkOrdersDelete(cmd *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	if workOrdersDeleteDryRun {
		mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
		opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
		dryRunData := map[string]any{
			"dry_run":      true,
			"would_delete": fmt.Sprintf("work_order:%d", id),
		}
		return output.Render(cmd.OutOrStdout(), dryRunData, fmt.Sprintf("Would delete work order %d (dry run).", id), nil, opts)
	}

	client, err := newSDKClient()
	if err != nil {
		return err
	}
	setRequest("DELETE", "/work_orders/"+args[0])

	_, err = client.DeleteWorkOrder(context.Background(), id)
	if err != nil {
		return err
	}

	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), nil, fmt.Sprintf("Work order %d deleted.", id), nil, opts)
}
