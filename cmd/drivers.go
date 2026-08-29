//go:build !generated
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
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
	driversCustomerID    int
	driverCreateFullName string
	driverCreatePhone    string
	driverUpdateFullName string
	driversDeleteDryRun  bool
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
	return runList(cmd, "drivers", fmt.Sprintf("/customers/%d/drivers", driversCustomerID), func(ctx context.Context, client *wenmar.Client) (any, error) {
		resp, err := client.ListDrivers(ctx, driversCustomerID)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runDriversShow(cmd *cobra.Command, args []string) error {
	return runShow(cmd, args, "drivers", "GET", idPath(fmt.Sprintf("/customers/%d/drivers/", driversCustomerID)), func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		resp, err := client.ShowDriver(ctx, driversCustomerID, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runDriversCreate(cmd *cobra.Command, args []string) error {
	return runCreate(cmd, "drivers", fmt.Sprintf("/customers/%d/drivers", driversCustomerID), "Driver created.", func() (any, error) {
		return wenmar.CreateDriverRequest{
			FullName: driverCreateFullName,
			Phone:    driverCreatePhone,
		}, nil
	}, func(ctx context.Context, client *wenmar.Client, body any) (any, error) {
		resp, err := client.CreateDriver(ctx, driversCustomerID, body.(wenmar.CreateDriverRequest))
		if err != nil {
			return nil, err
		}
		return resp.JSON201, nil
	})
}

func runDriversUpdate(cmd *cobra.Command, args []string) error {
	return runUpdate(cmd, args, "drivers", fmt.Sprintf("/customers/%d/drivers/", driversCustomerID), "Driver updated.", func(id int) (any, error) {
		return wenmar.UpdateDriverRequest{
			FullName: driverUpdateFullName,
		}, nil
	}, func(ctx context.Context, client *wenmar.Client, id int, body any) (any, error) {
		resp, err := client.UpdateDriver(ctx, driversCustomerID, id, body.(wenmar.UpdateDriverRequest))
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runDriversDelete(cmd *cobra.Command, args []string) error {
	return runDelete(cmd, args, "Driver", "drivers", func(a []string) string { return fmt.Sprintf("/customers/%d/drivers/%s", driversCustomerID, a[0]) }, driversDeleteDryRun, func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		return client.DeleteDriver(ctx, driversCustomerID, id)
	})
}