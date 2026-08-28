package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/wenmar-pro/wenmar-cli/internal/output"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
	"github.com/spf13/cobra"
)

var driversCmd = &cobra.Command{
	Use:   "drivers",
	Short: "Manage drivers",
}

var driversListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List drivers for a customer, paginated via the Link header",
	RunE:    runDriversList,
}

var driversShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single driver by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runDriversShow,
}

var driversCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new driver for a customer",
	RunE:  runDriversCreate,
}

var driversUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a driver by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runDriversUpdate,
}

var driversDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a driver by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runDriversDelete,
}

var (
	driversCustomerID     int
	driverCreateFullName  string
	driverCreatePhone     string
	driverUpdateFullName  string
	driversDeleteDryRun   bool
)

func init() {
	driversListCmd.Flags().IntVar(&driversCustomerID, "customer-id", 0, "Customer ID (required)")
	driversListCmd.MarkFlagRequired("customer-id")
	driversShowCmd.Flags().IntVar(&driversCustomerID, "customer-id", 0, "Customer ID (required)")
	driversShowCmd.MarkFlagRequired("customer-id")
	driversCreateCmd.Flags().IntVar(&driversCustomerID, "customer-id", 0, "Customer ID (required)")
	driversCreateCmd.Flags().StringVar(&driverCreateFullName, "full-name", "", "Driver full name (required)")
	driversCreateCmd.Flags().StringVar(&driverCreatePhone, "phone", "", "Driver phone")
	driversCreateCmd.MarkFlagRequired("customer-id")
	driversCreateCmd.MarkFlagRequired("full-name")
	driversUpdateCmd.Flags().IntVar(&driversCustomerID, "customer-id", 0, "Customer ID (required)")
	driversUpdateCmd.Flags().StringVar(&driverUpdateFullName, "full-name", "", "Driver full name")
	driversUpdateCmd.MarkFlagRequired("customer-id")
	driversDeleteCmd.Flags().IntVar(&driversCustomerID, "customer-id", 0, "Customer ID (required)")
	driversDeleteCmd.Flags().BoolVar(&driversDeleteDryRun, "dry-run", false, "Preview what would be deleted without making an API call")
	driversDeleteCmd.MarkFlagRequired("customer-id")

	driversCmd.AddCommand(driversListCmd, driversShowCmd, driversCreateCmd, driversUpdateCmd, driversDeleteCmd)
	rootCmd.AddCommand(driversCmd)
}

func runDriversList(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", fmt.Sprintf("/customers/%d/drivers", driversCustomerID))

	resp, err := client.ListDrivers(context.Background(), driversCustomerID)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("drivers")}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runDriversShow(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", fmt.Sprintf("/customers/%d/drivers/%s", driversCustomerID, args[0]))

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	resp, err := client.ShowDriver(context.Background(), driversCustomerID, id)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("drivers", args[0])}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runDriversCreate(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("POST", fmt.Sprintf("/customers/%d/drivers", driversCustomerID))

	body := wenmar.CreateDriverRequest{
		FullName: driverCreateFullName,
		Phone:    driverCreatePhone,
	}

	resp, err := client.CreateDriver(context.Background(), driversCustomerID, body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON201)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: createBreadcrumbs("drivers", "0")}
	return output.Render(cmd.OutOrStdout(), data, "Driver created.", nil, opts)
}

func runDriversUpdate(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("PATCH", fmt.Sprintf("/customers/%d/drivers/%s", driversCustomerID, args[0]))

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	body := wenmar.UpdateDriverRequest{
		FullName: driverUpdateFullName,
	}

	resp, err := client.UpdateDriver(context.Background(), driversCustomerID, id, body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("drivers", args[0])}
	return output.Render(cmd.OutOrStdout(), data, "Driver updated.", nil, opts)
}

func runDriversDelete(cmd *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	if driversDeleteDryRun {
		mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
		opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("drivers", args[0])}
		dryRunData := map[string]any{
			"dry_run":      true,
			"would_delete": fmt.Sprintf("driver:%d", id),
		}
		return output.Render(cmd.OutOrStdout(), dryRunData, fmt.Sprintf("Would delete driver %d (dry run).", id), nil, opts)
	}

	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("DELETE", fmt.Sprintf("/customers/%d/drivers/%s", driversCustomerID, args[0]))

	_, err = client.DeleteDriver(context.Background(), driversCustomerID, id)
	if err != nil {
		return err
	}

	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("drivers")}
	return output.Render(cmd.OutOrStdout(), nil, fmt.Sprintf("Driver %d deleted.", id), nil, opts)
}
