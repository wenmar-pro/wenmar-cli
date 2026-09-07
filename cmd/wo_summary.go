package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var woActivityCmd = &cobra.Command{
	Use:   "activity <work-order-id>",
	Short: "Show activity for a work order",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShow(cmd, args, "wo", "GET", func(a []string) string { return fmt.Sprintf("/work_orders/%s/activity", a[0]) },
			func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
				resp, err := client.ListWorkOrdersActivity(ctx, id, nil)
				if err != nil {
					return nil, err
				}
				return resp.JSON200, nil
			})
	},
}

var woVehicleHistoryCmd = &cobra.Command{
	Use:   "vehicle-history <work-order-id>",
	Short: "Show vehicle history for a work order",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShow(cmd, args, "wo", "GET", func(a []string) string { return fmt.Sprintf("/work_orders/%s/vehicle_history", a[0]) },
			func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
				resp, err := client.ListWorkOrdersVehicleHistory(ctx, id)
				if err != nil {
					return nil, err
				}
				return resp.JSON200, nil
			})
	},
}

var woAppointmentsCmd = &cobra.Command{
	Use:   "appointments <work-order-id>",
	Short: "Show appointments for a work order",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShow(cmd, args, "wo", "GET", func(a []string) string { return fmt.Sprintf("/work_orders/%s/appointments", a[0]) },
			func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
				resp, err := client.ListWorkOrdersAppointmentsRaw(ctx, id)
				if err != nil {
					return nil, err
				}
				return resp.JSON200, nil
			})
	},
}

var woAuthLogsCmd = &cobra.Command{
	Use:   "auth-logs <work-order-id>",
	Short: "Show authorization logs for a work order",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShow(cmd, args, "wo", "GET", func(a []string) string { return fmt.Sprintf("/work_orders/%s/authorization_logs", a[0]) },
			func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
				resp, err := client.ListWorkOrdersAuthorizationLogs(ctx, id)
				if err != nil {
					return nil, err
				}
				return resp.JSON200, nil
			})
	},
}

var (
	woNotesAddBody string
)

var woNotesCmd = &cobra.Command{
	Use:   "notes",
	Short: "Manage notes on a work order",
}

var woNotesAddCmd = &cobra.Command{
	Use:   "add <work-order-id>",
	Short: "Add a manual note to a work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrderNotesAdd,
}

func init() {
	woCmd.AddCommand(woActivityCmd, woVehicleHistoryCmd, woAppointmentsCmd, woAuthLogsCmd, woNotesCmd)
	woNotesCmd.AddCommand(woNotesAddCmd)

	woNotesAddCmd.Flags().StringVar(&woNotesAddBody, "body", "", "Note body (required)")
	_ = woNotesAddCmd.MarkFlagRequired("body")
}

func runWorkOrderNotesAdd(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("POST", fmt.Sprintf("/work_orders/%d/activity_logs", workOrderID))

	req := wenmar.CreateWorkOrdersActivityLogRequest{
		ActivityLog: struct {
			Body string `json:"body"`
		}{
			Body: woNotesAddBody,
		},
	}
	resp, err := client.CreateWorkOrdersActivityLog(context.Background(), workOrderID, req)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: createBreadcrumbs("wo", "0")}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON201), "Note added.", nil, opts)
}