package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var (
	woServicesAddName        string
	woServicesAddServiceType string
	woServicesAddPackageID   int

	woServicesUpdateName            string
	woServicesUpdatePosition        int
	woServicesUpdateTechnicianID    int
	woServicesUpdatePricingMode     string
	woServicesUpdateLaborTaxEnabled bool

	woServicesReorderServiceIDs []int

	woServicesAddPackagePackageID int
	woServicesAdjustTimeHours     int
	woServicesAdjustTimeMinutes   int
	woServicesToggleLineItemID    int
	woServicesUpdateCategoryID    int

	woServicesLineItemAddDescription string
	woServicesLineItemAddItemType    string
	woServicesLineItemAddHours       float32
	woServicesLineItemAddQuantity    int
	woServicesLineItemAddLaborRateID int
	woServicesLineItemAddUnitPrice   string
	woServicesLineItemAddTotal       string

	woServicesLineItemUpdateDescription string
	woServicesLineItemUpdatePartStatus  string
)

var woServicesCmd = &cobra.Command{
	Use:   "services <work-order-id>",
	Short: "Manage services on a work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrderServicesList,
}

var woServicesListCmd = &cobra.Command{
	Use:   "list <work-order-id>",
	Short: "List services on the work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrderServicesList,
}

var woServicesAddCmd = &cobra.Command{
	Use:   "add <work-order-id>",
	Short: "Add a service to the work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrderServicesAdd,
}

var woServicesUpdateCmd = &cobra.Command{
	Use:   "update <work-order-id> <service-id>",
	Short: "Update a service on the work order",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesUpdate,
}

var woServicesDeleteCmd = &cobra.Command{
	Use:   "delete <work-order-id> <service-id>",
	Short: "Delete a service from the work order",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesDelete,
}

var woServicesReorderCmd = &cobra.Command{
	Use:   "reorder <work-order-id>",
	Short: "Reorder services on the work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrderServicesReorder,
}

var woServicesCompleteCmd = &cobra.Command{
	Use:   "complete <work-order-id> <service-id>",
	Short: "Mark a service as complete",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesComplete,
}

var woServicesResetCompletionCmd = &cobra.Command{
	Use:   "reset-completion <work-order-id> <service-id>",
	Short: "Reset service completion",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesResetCompletion,
}

var woServicesUnauthorizeCmd = &cobra.Command{
	Use:   "unauthorize <work-order-id> <service-id>",
	Short: "Remove authorization from a service",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesUnauthorize,
}

var woServicesCopyCmd = &cobra.Command{
	Use:   "copy <work-order-id> <service-id>",
	Short: "Copy a service",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesCopy,
}

var woServicesAddPackageCmd = &cobra.Command{
	Use:   "add-package <work-order-id> <service-id>",
	Short: "Add a package to a service",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesAddPackage,
}

var woServicesPauseCmd = &cobra.Command{
	Use:   "pause <work-order-id> <service-id>",
	Short: "Pause a service",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesPause,
}

var woServicesPublishCmd = &cobra.Command{
	Use:   "publish <work-order-id> <service-id>",
	Short: "Publish a service",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesPublish,
}

var woServicesReviveCmd = &cobra.Command{
	Use:   "revive <work-order-id> <service-id>",
	Short: "Revive a service",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesRevive,
}

var woServicesAddTimeEntryCmd = &cobra.Command{
	Use:   "add-time-entry <work-order-id> <service-id>",
	Short: "Add a time entry to a service",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesAddTimeEntry,
}

var woServicesToggleLaborCompletionCmd = &cobra.Command{
	Use:   "toggle-labor-completion <work-order-id> <service-id>",
	Short: "Toggle labor completion for a line item",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesToggleLaborCompletion,
}

var woServicesAdjustTimeCmd = &cobra.Command{
	Use:   "adjust-time <work-order-id> <service-id>",
	Short: "Adjust logged time for a service",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesAdjustTime,
}

var woServicesUpdateCategoryCmd = &cobra.Command{
	Use:   "update-category <work-order-id> <service-id>",
	Short: "Update the category of a service",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesUpdateCategory,
}

var woServicesLineItemsCmd = &cobra.Command{
	Use:   "line-items <work-order-id> <service-id>",
	Short: "Manage line items on a service",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesLineItemsList,
}

var woServicesLineItemsAddCmd = &cobra.Command{
	Use:   "add <work-order-id> <service-id>",
	Short: "Add a line item to a service",
	Args:  cobra.ExactArgs(2),
	RunE:  runWorkOrderServicesLineItemsAdd,
}

var woServicesLineItemsUpdateCmd = &cobra.Command{
	Use:   "update <work-order-id> <service-id> <line-item-id>",
	Short: "Update a line item",
	Args:  cobra.ExactArgs(3),
	RunE:  runWorkOrderServicesLineItemsUpdate,
}

var woServicesLineItemsDeleteCmd = &cobra.Command{
	Use:   "delete <work-order-id> <service-id> <line-item-id>",
	Short: "Delete a line item",
	Args:  cobra.ExactArgs(3),
	RunE:  runWorkOrderServicesLineItemsDelete,
}

var woServicesLineItemsCopyCmd = &cobra.Command{
	Use:   "copy <work-order-id> <service-id> <line-item-id>",
	Short: "Copy a line item",
	Args:  cobra.ExactArgs(3),
	RunE:  runWorkOrderServicesLineItemsCopy,
}

var woServicesLineItemsInventoryAdditionCmd = &cobra.Command{
	Use:   "inventory-addition <work-order-id> <service-id> <line-item-id>",
	Short: "Add inventory to a line item",
	Args:  cobra.ExactArgs(3),
	RunE:  runWorkOrderServicesLineItemsInventoryAddition,
}

var woServicesLineItemsPriceRefreshCmd = &cobra.Command{
	Use:   "price-refresh <work-order-id> <service-id> <line-item-id>",
	Short: "Refresh line item pricing from inventory",
	Args:  cobra.ExactArgs(3),
	RunE:  runWorkOrderServicesLineItemsPriceRefresh,
}

func init() {
	woCmd.AddCommand(woServicesCmd)

	woServicesCmd.AddCommand(
		woServicesListCmd,
		woServicesAddCmd,
		woServicesUpdateCmd,
		woServicesDeleteCmd,
		woServicesReorderCmd,
		woServicesCompleteCmd,
		woServicesResetCompletionCmd,
		woServicesUnauthorizeCmd,
		woServicesCopyCmd,
		woServicesAddPackageCmd,
		woServicesPauseCmd,
		woServicesPublishCmd,
		woServicesReviveCmd,
		woServicesAddTimeEntryCmd,
		woServicesToggleLaborCompletionCmd,
		woServicesAdjustTimeCmd,
		woServicesUpdateCategoryCmd,
		woServicesLineItemsCmd,
	)

	woServicesLineItemsCmd.AddCommand(
		woServicesLineItemsAddCmd,
		woServicesLineItemsUpdateCmd,
		woServicesLineItemsDeleteCmd,
		woServicesLineItemsCopyCmd,
		woServicesLineItemsInventoryAdditionCmd,
		woServicesLineItemsPriceRefreshCmd,
	)

	woServicesAddCmd.Flags().StringVar(&woServicesAddName, "name", "", "Service name (required)")
	_ = woServicesAddCmd.MarkFlagRequired("name")
	woServicesAddCmd.Flags().StringVar(&woServicesAddServiceType, "service-type", "", "Service type")
	woServicesAddCmd.Flags().IntVar(&woServicesAddPackageID, "package-id", 0, "Package ID to convert into services")

	woServicesUpdateCmd.Flags().StringVar(&woServicesUpdateName, "name", "", "Service name")
	woServicesUpdateCmd.Flags().IntVar(&woServicesUpdatePosition, "position", 0, "Position")
	woServicesUpdateCmd.Flags().IntVar(&woServicesUpdateTechnicianID, "technician-id", 0, "Technician ID")
	woServicesUpdateCmd.Flags().StringVar(&woServicesUpdatePricingMode, "pricing-mode", "", "Pricing mode")
	woServicesUpdateCmd.Flags().BoolVar(&woServicesUpdateLaborTaxEnabled, "labor-tax-enabled", false, "Labor tax enabled")

	woServicesReorderCmd.Flags().IntSliceVar(&woServicesReorderServiceIDs, "service-ids", nil, "Ordered service IDs (required)")
	_ = woServicesReorderCmd.MarkFlagRequired("service-ids")

	woServicesAddPackageCmd.Flags().IntVar(&woServicesAddPackagePackageID, "package-id", 0, "Package ID (required)")
	_ = woServicesAddPackageCmd.MarkFlagRequired("package-id")

	woServicesAdjustTimeCmd.Flags().IntVar(&woServicesAdjustTimeHours, "hours", 0, "Hours (required)")
	_ = woServicesAdjustTimeCmd.MarkFlagRequired("hours")
	woServicesAdjustTimeCmd.Flags().IntVar(&woServicesAdjustTimeMinutes, "minutes", 0, "Minutes (required)")
	_ = woServicesAdjustTimeCmd.MarkFlagRequired("minutes")

	woServicesToggleLaborCompletionCmd.Flags().IntVar(&woServicesToggleLineItemID, "line-item-id", 0, "Line item ID (required)")
	_ = woServicesToggleLaborCompletionCmd.MarkFlagRequired("line-item-id")

	woServicesUpdateCategoryCmd.Flags().IntVar(&woServicesUpdateCategoryID, "category-id", 0, "Category ID (required)")
	_ = woServicesUpdateCategoryCmd.MarkFlagRequired("category-id")

	woServicesLineItemsAddCmd.Flags().StringVar(&woServicesLineItemAddDescription, "description", "", "Line item description (required)")
	_ = woServicesLineItemsAddCmd.MarkFlagRequired("description")
	woServicesLineItemsAddCmd.Flags().StringVar(&woServicesLineItemAddItemType, "item-type", "", "Item type (required)")
	_ = woServicesLineItemsAddCmd.MarkFlagRequired("item-type")
	woServicesLineItemsAddCmd.Flags().Float32Var(&woServicesLineItemAddHours, "hours", 0, "Labor hours")
	woServicesLineItemsAddCmd.Flags().IntVar(&woServicesLineItemAddQuantity, "quantity", 0, "Quantity")
	woServicesLineItemsAddCmd.Flags().IntVar(&woServicesLineItemAddLaborRateID, "labor-rate-id", 0, "Labor rate ID")
	woServicesLineItemsAddCmd.Flags().StringVar(&woServicesLineItemAddUnitPrice, "unit-price", "", "Unit price")
	woServicesLineItemsAddCmd.Flags().StringVar(&woServicesLineItemAddTotal, "total", "", "Total")

	woServicesLineItemsUpdateCmd.Flags().StringVar(&woServicesLineItemUpdateDescription, "description", "", "Description")
	woServicesLineItemsUpdateCmd.Flags().StringVar(&woServicesLineItemUpdatePartStatus, "part-status", "", "Part status")
}

func runWorkOrderServicesList(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("GET", fmt.Sprintf("/work_orders/%s/services", args[0]))

	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}

	resp, err := client.ListWorkOrderServices(context.Background(), workOrderID)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("wo")}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "", nil, opts)
}

func runWorkOrderServicesAdd(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("POST", fmt.Sprintf("/work_orders/%d/services", workOrderID))

	req := wenmar.CreateWorkOrdersServiceRequest{
		PackageId: intPtr(woServicesAddPackageID),
		WorkOrderService: struct {
			Name        string  `json:"name"`
			ServiceType *string `json:"service_type,omitempty"`
		}{
			Name:        woServicesAddName,
			ServiceType: strPtr(woServicesAddServiceType),
		},
	}
	if woServicesAddPackageID == 0 {
		req.PackageId = nil
	}

	resp, err := client.CreateWorkOrdersService(context.Background(), workOrderID, req)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: createBreadcrumbs("wo", "0")}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON201), "Service added.", nil, opts)
}

func runWorkOrderServicesUpdate(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("PATCH", fmt.Sprintf("/work_orders/%d/services/%d", workOrderID, serviceID))

	req := wenmar.UpdateWorkOrdersServiceRequest{
		WorkOrderService: struct {
			LaborTaxEnabled *bool   `json:"labor_tax_enabled,omitempty"`
			Name            *string `json:"name,omitempty"`
			Position        *int    `json:"position,omitempty"`
			PricingMode     *string `json:"pricing_mode,omitempty"`
			TechnicianId    *int    `json:"technician_id,omitempty"`
		}{
			Name:            strPtr(woServicesUpdateName),
			Position:        intPtr(woServicesUpdatePosition),
			TechnicianId:    intPtr(woServicesUpdateTechnicianID),
			PricingMode:     strPtr(woServicesUpdatePricingMode),
			LaborTaxEnabled: boolPtr(woServicesUpdateLaborTaxEnabled),
		},
	}
	if woServicesUpdatePosition == 0 {
		req.WorkOrderService.Position = nil
	}
	if woServicesUpdateTechnicianID == 0 {
		req.WorkOrderService.TechnicianId = nil
	}

	resp, err := client.UpdateWorkOrdersService(context.Background(), workOrderID, serviceID, req)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Service updated.", nil, opts)
}

func runWorkOrderServicesDelete(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("DELETE", fmt.Sprintf("/work_orders/%d/services/%d", workOrderID, serviceID))

	_, err = client.DeleteWorkOrdersService(context.Background(), workOrderID, serviceID)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), nil, "Service deleted.", nil, opts)
}

func runWorkOrderServicesReorder(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("PATCH", fmt.Sprintf("/work_orders/%d/services/reorder", workOrderID))

	req := wenmar.UpdateWorkOrdersServicesReorderRequest{ServiceIds: woServicesReorderServiceIDs}
	_, err = client.UpdateWorkOrdersServicesReorder(context.Background(), workOrderID, req)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), nil, "Services reordered.", nil, opts)
}

func runWorkOrderServicesComplete(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("POST", fmt.Sprintf("/work_orders/%d/services/%d/completion", workOrderID, serviceID))

	resp, err := client.CreateWorkOrdersServicesCompletion(context.Background(), workOrderID, serviceID, map[string]any{})
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Service marked complete.", nil, opts)
}

func runWorkOrderServicesResetCompletion(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("DELETE", fmt.Sprintf("/work_orders/%d/services/%d/completion", workOrderID, serviceID))

	resp, err := client.DeleteWorkOrdersServicesCompletion(context.Background(), workOrderID, serviceID)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Service completion reset.", nil, opts)
}

func runWorkOrderServicesUnauthorize(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("DELETE", fmt.Sprintf("/work_orders/%d/services/%d/authorization", workOrderID, serviceID))

	resp, err := client.DeleteWorkOrdersServicesAuthorization(context.Background(), workOrderID, serviceID)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Service unauthorized.", nil, opts)
}

func runWorkOrderServicesCopy(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("POST", fmt.Sprintf("/work_orders/%d/services/%d/copies", workOrderID, serviceID))

	resp, err := client.CreateWorkOrdersServicesCopy(context.Background(), workOrderID, serviceID, map[string]any{})
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON201), "Service copied.", nil, opts)
}

func runWorkOrderServicesAddPackage(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("POST", fmt.Sprintf("/work_orders/%d/services/%d/packages", workOrderID, serviceID))

	req := wenmar.CreateWorkOrdersServicesPackageRequest{PackageId: woServicesAddPackagePackageID}
	resp, err := client.CreateWorkOrdersServicesPackage(context.Background(), workOrderID, serviceID, req)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Package added to service.", nil, opts)
}

func runWorkOrderServicesPause(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("PATCH", fmt.Sprintf("/work_orders/%d/services/%d/pause", workOrderID, serviceID))

	resp, err := client.UpdateWorkOrdersServicesPause(context.Background(), workOrderID, serviceID, map[string]any{})
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Service paused.", nil, opts)
}

func runWorkOrderServicesPublish(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("PATCH", fmt.Sprintf("/work_orders/%d/services/%d/publish", workOrderID, serviceID))

	resp, err := client.UpdateWorkOrdersServicesPublish(context.Background(), workOrderID, serviceID, map[string]any{})
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Service published.", nil, opts)
}

func runWorkOrderServicesRevive(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("PATCH", fmt.Sprintf("/work_orders/%d/services/%d/revive", workOrderID, serviceID))

	resp, err := client.UpdateWorkOrdersServicesRevive(context.Background(), workOrderID, serviceID, map[string]any{})
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Service revived.", nil, opts)
}

func runWorkOrderServicesAddTimeEntry(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("POST", fmt.Sprintf("/work_orders/%d/services/%d/time_entries", workOrderID, serviceID))

	resp, err := client.CreateWorkOrdersServicesTimeEntry(context.Background(), workOrderID, serviceID, map[string]any{})
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Time entry added.", nil, opts)
}

func runWorkOrderServicesToggleLaborCompletion(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("PATCH", fmt.Sprintf("/work_orders/%d/services/%d/toggle_labor_completion", workOrderID, serviceID))

	req := wenmar.UpdateWorkOrdersServicesToggleLaborCompletionRequest{LineItemId: woServicesToggleLineItemID}
	resp, err := client.UpdateWorkOrdersServicesToggleLaborCompletion(context.Background(), workOrderID, serviceID, req)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Labor completion toggled.", nil, opts)
}

func runWorkOrderServicesAdjustTime(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("PATCH", fmt.Sprintf("/work_orders/%d/services/%d/adjust_time", workOrderID, serviceID))

	req := wenmar.UpdateWorkOrdersServicesAdjustTimeRequest{
		Hours:   woServicesAdjustTimeHours,
		Minutes: woServicesAdjustTimeMinutes,
	}
	resp, err := client.UpdateWorkOrdersServicesAdjustTime(context.Background(), workOrderID, serviceID, req)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Service time adjusted.", nil, opts)
}

func runWorkOrderServicesUpdateCategory(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("PATCH", fmt.Sprintf("/work_orders/%d/services/%d/update_category", workOrderID, serviceID))

	req := wenmar.UpdateWorkOrdersServicesUpdateCategoryRequest{CategoryId: woServicesUpdateCategoryID}
	resp, err := client.UpdateWorkOrdersServicesUpdateCategory(context.Background(), workOrderID, serviceID, req)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Service category updated.", nil, opts)
}

func runWorkOrderServicesLineItemsList(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("GET", fmt.Sprintf("/work_orders/%d/estimate", workOrderID))

	resp, err := client.ShowWorkOrderEstimate(context.Background(), workOrderID)
	if err != nil {
		return formatMissingLocationError(err)
	}

	var lineItems any
	if wo, ok := extractData(resp.JSON200).(map[string]any); ok {
		if services, ok := wo["services"].([]any); ok {
			for _, s := range services {
				if svc, ok := s.(map[string]any); ok {
					if id, ok := svc["id"].(float64); ok && int(id) == serviceID {
						lineItems = svc["line_items"]
						break
					}
				}
			}
		}
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), lineItems, "", nil, opts)
}

func runWorkOrderServicesLineItemsAdd(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("POST", fmt.Sprintf("/work_orders/%d/services/%d/line_items", workOrderID, serviceID))

	req := wenmar.CreateWorkOrdersServicesLineItemRequest{
		WorkOrderLineItem: struct {
			Description string   `json:"description"`
			Hours       *float32 `json:"hours,omitempty"`
			ItemType    string   `json:"item_type"`
			LaborRateId *int     `json:"labor_rate_id,omitempty"`
			Quantity    *int     `json:"quantity,omitempty"`
			Total       *string  `json:"total,omitempty"`
			UnitPrice   *string  `json:"unit_price,omitempty"`
		}{
			Description: woServicesLineItemAddDescription,
			ItemType:    woServicesLineItemAddItemType,
			Hours:       float32Ptr(woServicesLineItemAddHours),
			Quantity:    intPtr(woServicesLineItemAddQuantity),
			LaborRateId: intPtr(woServicesLineItemAddLaborRateID),
			Total:       strPtr(woServicesLineItemAddTotal),
			UnitPrice:   strPtr(woServicesLineItemAddUnitPrice),
		},
	}
	if woServicesLineItemAddHours == 0 {
		req.WorkOrderLineItem.Hours = nil
	}
	if woServicesLineItemAddQuantity == 0 {
		req.WorkOrderLineItem.Quantity = nil
	}
	if woServicesLineItemAddLaborRateID == 0 {
		req.WorkOrderLineItem.LaborRateId = nil
	}

	resp, err := client.CreateWorkOrdersServicesLineItem(context.Background(), workOrderID, serviceID, req)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON201), "Line item added.", nil, opts)
}

func runWorkOrderServicesLineItemsUpdate(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	lineItemID, err := parseInt(args[2])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("PATCH", fmt.Sprintf("/work_orders/%d/services/%d/line_items/%d", workOrderID, serviceID, lineItemID))

	req := wenmar.UpdateWorkOrdersServicesLineItemRequest{
		WorkOrderLineItem: struct {
			Description *string `json:"description,omitempty"`
			PartStatus  *string `json:"part_status,omitempty"`
		}{
			Description: strPtr(woServicesLineItemUpdateDescription),
			PartStatus:  strPtr(woServicesLineItemUpdatePartStatus),
		},
	}

	resp, err := client.UpdateWorkOrdersServicesLineItem(context.Background(), workOrderID, serviceID, lineItemID, req)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Line item updated.", nil, opts)
}

func runWorkOrderServicesLineItemsDelete(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	lineItemID, err := parseInt(args[2])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("DELETE", fmt.Sprintf("/work_orders/%d/services/%d/line_items/%d", workOrderID, serviceID, lineItemID))

	_, err = client.DeleteWorkOrdersServicesLineItem(context.Background(), workOrderID, serviceID, lineItemID)
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), nil, "Line item deleted.", nil, opts)
}

func runWorkOrderServicesLineItemsCopy(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	lineItemID, err := parseInt(args[2])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("POST", fmt.Sprintf("/work_orders/%d/services/%d/line_items/%d/copies", workOrderID, serviceID, lineItemID))

	resp, err := client.CreateWorkOrdersServicesLineItemsCopy(context.Background(), workOrderID, serviceID, lineItemID, map[string]any{})
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON201), "Line item copied.", nil, opts)
}

func runWorkOrderServicesLineItemsInventoryAddition(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	lineItemID, err := parseInt(args[2])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("POST", fmt.Sprintf("/work_orders/%d/services/%d/line_items/%d/inventory_addition", workOrderID, serviceID, lineItemID))

	resp, err := client.CreateWorkOrdersServicesLineItemsInventoryAddition(context.Background(), workOrderID, serviceID, lineItemID, map[string]any{})
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Inventory added.", nil, opts)
}

func runWorkOrderServicesLineItemsPriceRefresh(cmd *cobra.Command, args []string) error {
	workOrderID, err := parseInt(args[0])
	if err != nil {
		return err
	}
	serviceID, err := parseInt(args[1])
	if err != nil {
		return err
	}
	lineItemID, err := parseInt(args[2])
	if err != nil {
		return err
	}
	client, err := newScopedClient()
	if err != nil {
		return err
	}
	setRequest("POST", fmt.Sprintf("/work_orders/%d/services/%d/line_items/%d/price_refresh", workOrderID, serviceID, lineItemID))

	resp, err := client.CreateWorkOrdersServicesLineItemsPriceRefresh(context.Background(), workOrderID, serviceID, lineItemID, map[string]any{})
	if err != nil {
		return formatMissingLocationError(err)
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("wo", args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(resp.JSON200), "Line item price refreshed.", nil, opts)
}
