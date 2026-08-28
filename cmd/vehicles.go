package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/wenmar-pro/wenmar-cli/internal/output"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
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
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", "/vehicles/"+args[0])

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	resp, err := client.ShowVehicle(context.Background(), id)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("vehicles", args[0])}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runVehiclesList(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", "/vehicles")

	resp, err := client.ListVehicles(context.Background())
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("vehicles")}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runVehiclesCreate(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("POST", "/vehicles")

	body := wenmar.CreateVehicleRequest{
		CustomerID: vehicleCreateCustomer,
		Make:       vehicleCreateMake,
		Model:      vehicleCreateModel,
		Year:       vehicleCreateYear,
	}

	resp, err := client.CreateVehicle(context.Background(), body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON201)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: createBreadcrumbs("vehicles", "9")}
	return output.Render(cmd.OutOrStdout(), data, "Vehicle created.", nil, opts)
}

func runVehiclesUpdate(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("PATCH", "/vehicles/"+args[0])

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	body := wenmar.UpdateVehicleRequest{
		Make: vehicleCreateMake,
	}

	resp, err := client.UpdateVehicle(context.Background(), id, body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("vehicles")}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runVehiclesDelete(cmd *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	if vehiclesDeleteDryRun {
		mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
		opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("vehicles", args[0])}
		dryRunData := map[string]any{
			"dry_run":      true,
			"would_delete": fmt.Sprintf("vehicle:%d", id),
		}
		return output.Render(cmd.OutOrStdout(), dryRunData, fmt.Sprintf("Would delete vehicle %d (dry run).", id), nil, opts)
	}

	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("DELETE", "/vehicles/"+args[0])

	_, err = client.DeleteVehicle(context.Background(), id)
	if err != nil {
		return err
	}

	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("vehicles")}
	return output.Render(cmd.OutOrStdout(), nil, fmt.Sprintf("Vehicle %d deleted.", id), nil, opts)
}

func runVehiclesDecodeVin(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", "/vehicles/vin_decode")

	resp, err := client.DecodeVin(context.Background(), args[0])
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("vehicles")}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runVehiclesDuplicates(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", "/vehicles/check_duplicate")

	resp, err := client.CheckDuplicate(context.Background(), args[0])
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("vehicles", args[0])}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}
