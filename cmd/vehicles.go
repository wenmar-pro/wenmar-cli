package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/wenmar-pro/wenmar-cli/internal/output"
	"github.com/wenmar-pro/wenmar-sdk/go/pkg/generated"
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
	RunE:  runVehiclesShow,
}

var vehiclesListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all vehicles",
	RunE:    runVehiclesList,
}

var vehiclesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new vehicle",
	RunE:  runVehiclesCreate,
}

var vehiclesUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a vehicle by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runVehiclesUpdate,
}

var vehiclesDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a vehicle by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runVehiclesDelete,
}

var vehiclesDecodeVinCmd = &cobra.Command{
	Use:   "decode-vin <vin>",
	Short: "Decode a VIN into make/model",
	Args:  cobra.ExactArgs(1),
	RunE:  runVehiclesDecodeVin,
}

var vehiclesDuplicatesCmd = &cobra.Command{
	Use:   "duplicates <vin>",
	Short: "Check for duplicate vehicles matching a VIN",
	Args:  cobra.ExactArgs(1),
	RunE:  runVehiclesDuplicates,
}

var (
	vehicleCreateMake     string
	vehicleCreateModel    string
	vehicleCreateYear     int
	vehicleCreateCustomer int
	vehiclesDeleteDryRun  bool
)

func init() {
	vehiclesCreateCmd.Flags().StringVar(&vehicleCreateMake, "make", "", "Vehicle make (required)")
	vehiclesCreateCmd.Flags().StringVar(&vehicleCreateModel, "model", "", "Vehicle model (required)")
	vehiclesCreateCmd.Flags().IntVar(&vehicleCreateYear, "year", 0, "Vehicle year (required)")
	vehiclesCreateCmd.Flags().IntVar(&vehicleCreateCustomer, "customer-id", 0, "Customer ID (required)")
	vehiclesCreateCmd.MarkFlagRequired("make")
	vehiclesCreateCmd.MarkFlagRequired("model")
	vehiclesCreateCmd.MarkFlagRequired("year")
	vehiclesCreateCmd.MarkFlagRequired("customer-id")
	vehiclesDeleteCmd.Flags().BoolVar(&vehiclesDeleteDryRun, "dry-run", false, "Preview what would be deleted without making an API call")

	vehiclesCmd.AddCommand(vehiclesShowCmd, vehiclesListCmd, vehiclesCreateCmd, vehiclesUpdateCmd, vehiclesDeleteCmd, vehiclesDecodeVinCmd, vehiclesDuplicatesCmd)
	rootCmd.AddCommand(vehiclesCmd)
}

func runVehiclesShow(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	resp, err := client.ShowVehicle(context.Background(), id)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runVehiclesList(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}

	resp, err := client.ListVehicles(context.Background())
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runVehiclesCreate(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}

	body := generated.CreateVehicleJSONRequestBody{
		Vehicle: struct {
			CustomerId int    `json:"customer_id"`
			Make       string `json:"make"`
			Model      string `json:"model"`
			Year       int    `json:"year"`
		}{
			CustomerId: vehicleCreateCustomer,
			Make:       vehicleCreateMake,
			Model:      vehicleCreateModel,
			Year:       vehicleCreateYear,
		},
	}

	resp, err := client.CreateVehicle(context.Background(), body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON201)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), data, "Vehicle created.", nil, opts)
}

func runVehiclesUpdate(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	body := generated.UpdateVehicleJSONRequestBody{
		Vehicle: struct {
			Make string `json:"make"`
		}{Make: vehicleCreateMake},
	}

	resp, err := client.UpdateVehicle(context.Background(), id, body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), data, "Vehicle updated.", nil, opts)
}

func runVehiclesDelete(cmd *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	if vehiclesDeleteDryRun {
		mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
		opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
		dryRunData := map[string]any{
			"dry_run":      true,
			"would_delete": fmt.Sprintf("vehicle:%d", id),
		}
		return output.Render(cmd.OutOrStdout(), dryRunData, fmt.Sprintf("Would delete vehicle %d (dry run).", id), nil, opts)
	}

	client, err := newSDKClient()
	if err != nil {
		return err
	}

	_, err = client.DeleteVehicle(context.Background(), id)
	if err != nil {
		return err
	}

	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), nil, fmt.Sprintf("Vehicle %d deleted.", id), nil, opts)
}

func runVehiclesDecodeVin(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}

	resp, err := client.DecodeVin(context.Background(), args[0])
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runVehiclesDuplicates(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}

	resp, err := client.CheckDuplicate(context.Background(), args[0])
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}
