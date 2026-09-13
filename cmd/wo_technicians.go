package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var woTechniciansAddTechnicianID int

var woTechniciansCmd = &cobra.Command{
	Use:   "technicians <work-order-id>",
	Short: "Manage technician assignments on a work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrderTechniciansAdd,
}

var woTechniciansAddCmd = &cobra.Command{
	Use:   "add <work-order-id>",
	Short: "Assign a technician to the work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrderTechniciansAdd,
}

func init() {
	woCmd.AddCommand(woTechniciansCmd)
	woTechniciansCmd.AddCommand(woTechniciansAddCmd)

	woTechniciansAddCmd.Flags().IntVar(&woTechniciansAddTechnicianID, "technician-id", 0, "Technician ID (required)")
	_ = woTechniciansAddCmd.MarkFlagRequired("technician-id")
}

// runWorkOrderTechniciansAdd assigns a technician to a work order. The
// work-order association is a query param, so it is passed via the SDK's
// params struct rather than the path.
func runWorkOrderTechniciansAdd(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("POST", fmt.Sprintf("/tech_assignments?work_order_id=%d", workOrderID))

	resp, err := client.CreateWorkOrderTechAssignment(context.Background(),
		&wenmar.CreateWorkOrderTechAssignmentParams{WorkOrderId: intPtr(workOrderID)},
		wenmar.CreateWorkOrderTechAssignmentRequest{TechnicianId: woTechniciansAddTechnicianID},
	)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: createBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Technician assigned.", nil, opts)
}
