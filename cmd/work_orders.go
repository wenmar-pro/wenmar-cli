package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
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

var workOrdersEstimateCmd = &cobra.Command{
	Use:   "estimate <id>",
	Short: "Show the estimate tab (services) for a work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrdersTab("estimate"),
}

var workOrdersWipCmd = &cobra.Command{
	Use:   "wip <id>",
	Short: "Show the work-in-progress tab (services) for a work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrdersTab("wip"),
}

var workOrdersInspectionCmd = &cobra.Command{
	Use:   "inspection <id>",
	Short: "Show the inspection tab (inspection reports) for a work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrdersTab("inspection"),
}

var workOrdersPartsCmd = &cobra.Command{
	Use:   "parts <id>",
	Short: "Show the parts tab (services) for a work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrdersTab("parts"),
}

var workOrdersPaymentsCmd = &cobra.Command{
	Use:   "payments <id>",
	Short: "Show the payments tab (payments) for a work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrdersTab("payments"),
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

	workOrdersCmd.AddCommand(workOrdersListCmd, workOrdersShowCmd, workOrdersCreateCmd, workOrdersUpdateCmd, workOrdersDeleteCmd,
		workOrdersEstimateCmd, workOrdersWipCmd, workOrdersInspectionCmd, workOrdersPartsCmd, workOrdersPaymentsCmd)
	rootCmd.AddCommand(workOrdersCmd)
}

func runWorkOrdersList(cmd *cobra.Command, args []string) error {
	return runListPaginated(cmd, "work_orders", "/work_orders", func(ctx context.Context, client *wenmar.Client) (any, *wenmar.Paginator, error) {
		resp, paginator, err := client.ListWorkOrdersWithPagination(ctx)
		if err != nil {
			return nil, nil, err
		}
		return resp.JSON200, paginator, nil
	})
}

func runWorkOrdersShow(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", "/work_orders/"+args[0])

	id, err := parseInt(args[0])
	if err != nil {
		return err
	}

	resp, err := client.ShowWorkOrder(context.Background(), id)
	if err != nil {
		return err
	}

	notice, err := checkTruncatedResponse(resp.HTTPResponse, "Response was truncated. Some line items may be missing.")
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := resolveMode()
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("work_orders", args[0]), Notice: notice}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runWorkOrdersCreate(cmd *cobra.Command, args []string) error {
	return runCreate(cmd, "work_orders", "/work_orders", "Work order created.", func() (any, error) {
		return wenmar.CreateWorkOrderRequest{
			CustomerID: workOrderCreateCustomer,
			VehicleID:  workOrderCreateVehicle,
		}, nil
	}, func(ctx context.Context, client *wenmar.Client, body any) (any, error) {
		resp, err := client.CreateWorkOrder(ctx, body.(wenmar.CreateWorkOrderRequest))
		if err != nil {
			return nil, err
		}
		return resp.JSON201, nil
	})
}

func runWorkOrdersUpdate(cmd *cobra.Command, args []string) error {
	return runUpdate(cmd, args, "work_orders", "/work_orders/", "Work order updated.", func(id int) (any, error) {
		return wenmar.UpdateWorkOrderRequest{
			IntakeMethod: workOrderUpdateIntakeMethod,
		}, nil
	}, func(ctx context.Context, client *wenmar.Client, id int, body any) (any, error) {
		resp, err := client.UpdateWorkOrder(ctx, id, body.(wenmar.UpdateWorkOrderRequest))
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runWorkOrdersDelete(cmd *cobra.Command, args []string) error {
	return runDelete(cmd, args, "Work order", "work_orders", "/work_orders/", workOrdersDeleteDryRun, func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		return client.DeleteWorkOrder(ctx, id)
	})
}

// runWorkOrdersTab returns a RunE that fetches a work order sub-collection
// (estimate/wip/inspection/parts/payments) and renders it generically.
func runWorkOrdersTab(tab string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return runShow(cmd, args, "work_orders", "GET", func(a []string) string { return fmt.Sprintf("/work_orders/%s/%s", a[0], tab) }, func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
			resp, err := fetchWorkOrderTab(ctx, client, id, tab)
			if err != nil {
				return nil, err
			}
			return resp, nil
		})
	}
}

func fetchWorkOrderTab(ctx context.Context, client *wenmar.Client, id int, tab string) (any, error) {
	switch tab {
	case "estimate":
		resp, err := client.ShowWorkOrderEstimate(ctx, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	case "wip":
		resp, err := client.ShowWorkOrderWip(ctx, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	case "inspection":
		resp, err := client.ShowWorkOrderInspection(ctx, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	case "parts":
		resp, err := client.ShowWorkOrderParts(ctx, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	case "payments":
		resp, err := client.ShowWorkOrderPayments(ctx, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	default:
		return nil, fmt.Errorf("unknown tab: %s", tab)
	}
}