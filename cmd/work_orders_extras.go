package cmd

// work_orders_extras.go holds the work-order show command and its five tab
// sub-commands (estimate/wip/inspection/parts/payments), whose truncation
// check and tab-fetch switch are not per-operation derivable. Everything
// else under "workorders" is generated (gen_workorders.go).

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var workOrdersShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single work order by ID",
	Example: `  wenmar workorders show 100
  wenmar workorders show 100 --agent`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkOrdersShow,
}

var workOrdersEstimateCmd = &cobra.Command{
	Use:     "estimate <id>",
	Short:   "Show the estimate tab (services) for a work order",
	Example: `  wenmar workorders estimate 100`,
	Args:    cobra.ExactArgs(1),
	RunE:    runWorkOrdersTab("estimate"),
}

var workOrdersWipCmd = &cobra.Command{
	Use:     "wip <id>",
	Short:   "Show the work-in-progress tab (services) for a work order",
	Example: `  wenmar workorders wip 100`,
	Args:    cobra.ExactArgs(1),
	RunE:    runWorkOrdersTab("wip"),
}

var workOrdersInspectionCmd = &cobra.Command{
	Use:     "inspection <id>",
	Short:   "Show the inspection tab (inspection reports) for a work order",
	Example: `  wenmar workorders inspection 100`,
	Args:    cobra.ExactArgs(1),
	RunE:    runWorkOrdersTab("inspection"),
}

var workOrdersPartsCmd = &cobra.Command{
	Use:     "parts <id>",
	Short:   "Show the parts tab (services) for a work order",
	Example: `  wenmar workorders parts 100`,
	Args:    cobra.ExactArgs(1),
	RunE:    runWorkOrdersTab("parts"),
}

var workOrdersPaymentsCmd = &cobra.Command{
	Use:     "payments <id>",
	Short:   "Show the payments tab (payments) for a work order",
	Example: `  wenmar workorders payments 100`,
	Args:    cobra.ExactArgs(1),
	RunE:    runWorkOrdersTab("payments"),
}

func init() {
	workordersCmd.AddCommand(workOrdersShowCmd, workOrdersEstimateCmd, workOrdersWipCmd,
		workOrdersInspectionCmd, workOrdersPartsCmd, workOrdersPaymentsCmd)
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
	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("workorders", args[0]), Notice: notice}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

// runWorkOrdersTab returns a RunE that fetches a work order sub-collection
// (estimate/wip/inspection/parts/payments) and renders it generically.
func runWorkOrdersTab(tab string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return runShow(cmd, args, "workorders", "GET", func(a []string) string { return fmt.Sprintf("/work_orders/%s/%s", a[0], tab) }, func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
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
