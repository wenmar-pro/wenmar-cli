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

var vehiclesTransferCmd = &cobra.Command{
	Use:   "transfer <id>",
	Short: "Transfer a vehicle to a new customer",
	Args:  cobra.ExactArgs(1),
	RunE:  runVehiclesTransfer,
}

var vehiclesMergeCmd = &cobra.Command{
	Use:   "merge <id>",
	Short: "Merge a source vehicle into this keeper",
	Args:  cobra.ExactArgs(1),
	RunE:  runVehiclesMerge,
}

var vehiclesPrefillCmd = &cobra.Command{
	Use:   "prefill",
	Short: "Prefill vehicle data from a VIN",
	RunE:  runVehiclesPrefill,
}

var vehiclesLookupCmd = &cobra.Command{
	Use:   "lookup <query>",
	Short: "Search vehicles by make/model/plate/vin",
	Args:  cobra.ExactArgs(1),
	RunE:  runVehiclesLookup,
}

var vehiclesWorkOrdersCmd = &cobra.Command{
	Use:   "work-orders <id>",
	Short: "List a vehicle's work orders",
	Args:  cobra.ExactArgs(1),
	RunE:  runVehiclesWorkOrders,
}

var (
	vehicleCreateMake           string
	vehicleCreateModel          string
	vehicleCreateYear           int
	vehicleCreateCustomer       int
	vehiclesDeleteDryRun        bool
	vehicleUpdateMake           string
	vehicleVin                  string
	vehicleSubmodel             string
	vehicleBodyStyle            string
	vehicleEngine               string
	vehicleTransmission         string
	vehicleDrivetrain           string
	vehicleColor                string
	vehiclePlate                string
	vehiclePlateState           string
	vehicleOdometer             int
	vehicleOdometerUnit         string
	vehicleUnitNumber           string
	vehicleFleetIdentifier      string
	vehicleProductionDate       string
	vehicleNotes                string
	vehicleTagIDs               []int
	vehicleTransferCustomerID   int
	vehicleTransferMode         string
	vehicleMergeSourceID        int
	vehiclePrefillVIN           string
)

func init() {
	vehiclesCreateCmd.Flags().StringVar(&vehicleCreateMake, "make", "", "Vehicle make (required)")
	vehiclesCreateCmd.Flags().StringVar(&vehicleCreateModel, "model", "", "Vehicle model (required)")
	vehiclesCreateCmd.Flags().IntVar(&vehicleCreateYear, "year", 0, "Vehicle year (required)")
	vehiclesCreateCmd.Flags().IntVar(&vehicleCreateCustomer, "customer-id", 0, "Customer ID (required)")
	vehiclesCreateCmd.Flags().StringVar(&vehicleVin, "vin", "", "VIN")
	vehiclesCreateCmd.Flags().StringVar(&vehicleSubmodel, "submodel", "", "Submodel")
	vehiclesCreateCmd.Flags().StringVar(&vehicleBodyStyle, "body-style", "", "Body style")
	vehiclesCreateCmd.Flags().StringVar(&vehicleEngine, "engine", "", "Engine")
	vehiclesCreateCmd.Flags().StringVar(&vehicleTransmission, "transmission", "", "Transmission (automatic, manual, cvt, dct)")
	vehiclesCreateCmd.Flags().StringVar(&vehicleDrivetrain, "drivetrain", "", "Drivetrain (fwd, rwd, awd, four_wd)")
	vehiclesCreateCmd.Flags().StringVar(&vehicleColor, "color", "", "Color")
	vehiclesCreateCmd.Flags().StringVar(&vehiclePlate, "plate", "", "License plate")
	vehiclesCreateCmd.Flags().StringVar(&vehiclePlateState, "plate-state", "", "License plate state (e.g. ca_on)")
	vehiclesCreateCmd.Flags().IntVar(&vehicleOdometer, "odometer", 0, "Odometer reading")
	vehiclesCreateCmd.Flags().StringVar(&vehicleOdometerUnit, "odometer-unit", "", "Odometer unit (km, mi)")
	vehiclesCreateCmd.Flags().StringVar(&vehicleUnitNumber, "unit-number", "", "Unit number")
	vehiclesCreateCmd.Flags().StringVar(&vehicleFleetIdentifier, "fleet-identifier", "", "Fleet identifier")
	vehiclesCreateCmd.Flags().StringVar(&vehicleProductionDate, "production-date", "", "Production date (YYYY-MM-DD)")
	vehiclesCreateCmd.Flags().StringVar(&vehicleNotes, "notes", "", "Notes")
	vehiclesCreateCmd.Flags().IntSliceVar(&vehicleTagIDs, "tag-id", nil, "Vehicle tag ID, repeatable")
	vehiclesCreateCmd.MarkFlagRequired("make")
	vehiclesCreateCmd.MarkFlagRequired("model")
	vehiclesCreateCmd.MarkFlagRequired("year")
	vehiclesCreateCmd.MarkFlagRequired("customer-id")

	vehiclesUpdateCmd.Flags().StringVar(&vehicleUpdateMake, "make", "", "Vehicle make")
	vehiclesUpdateCmd.Flags().StringVar(&vehicleVin, "vin", "", "VIN")
	vehiclesUpdateCmd.Flags().StringVar(&vehicleSubmodel, "submodel", "", "Submodel")
	vehiclesUpdateCmd.Flags().StringVar(&vehicleBodyStyle, "body-style", "", "Body style")
	vehiclesUpdateCmd.Flags().StringVar(&vehicleEngine, "engine", "", "Engine")
	vehiclesUpdateCmd.Flags().StringVar(&vehicleTransmission, "transmission", "", "Transmission")
	vehiclesUpdateCmd.Flags().StringVar(&vehicleDrivetrain, "drivetrain", "", "Drivetrain")
	vehiclesUpdateCmd.Flags().StringVar(&vehicleColor, "color", "", "Color")
	vehiclesUpdateCmd.Flags().StringVar(&vehiclePlate, "plate", "", "License plate")
	vehiclesUpdateCmd.Flags().StringVar(&vehiclePlateState, "plate-state", "", "License plate state")
	vehiclesUpdateCmd.Flags().IntVar(&vehicleOdometer, "odometer", 0, "Odometer reading")
	vehiclesUpdateCmd.Flags().StringVar(&vehicleOdometerUnit, "odometer-unit", "", "Odometer unit")
	vehiclesUpdateCmd.Flags().StringVar(&vehicleUnitNumber, "unit-number", "", "Unit number")
	vehiclesUpdateCmd.Flags().StringVar(&vehicleFleetIdentifier, "fleet-identifier", "", "Fleet identifier")
	vehiclesUpdateCmd.Flags().StringVar(&vehicleProductionDate, "production-date", "", "Production date")
	vehiclesUpdateCmd.Flags().StringVar(&vehicleNotes, "notes", "", "Notes")

	vehiclesTransferCmd.Flags().IntVar(&vehicleTransferCustomerID, "customer-id", 0, "New customer ID (required)")
	vehiclesTransferCmd.Flags().StringVar(&vehicleTransferMode, "mode", "vehicle_only", "Transfer mode (vehicle_only, vehicle_and_history, everything)")
	vehiclesTransferCmd.MarkFlagRequired("customer-id")
	vehiclesMergeCmd.Flags().IntVar(&vehicleMergeSourceID, "source-id", 0, "Source vehicle ID (required)")
	vehiclesMergeCmd.MarkFlagRequired("source-id")
	vehiclesPrefillCmd.Flags().StringVar(&vehiclePrefillVIN, "vin", "", "VIN")

	vehiclesDeleteCmd.Flags().BoolVar(&vehiclesDeleteDryRun, "dry-run", false, "Preview what would be deleted without making an API call")

	vehiclesCmd.AddCommand(vehiclesShowCmd, vehiclesListCmd, vehiclesCreateCmd, vehiclesUpdateCmd, vehiclesDeleteCmd, vehiclesDecodeVinCmd, vehiclesDuplicatesCmd,
		vehiclesTransferCmd, vehiclesMergeCmd, vehiclesPrefillCmd, vehiclesLookupCmd, vehiclesWorkOrdersCmd)
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

	body := generated.CreateVehicleJSONRequestBody{
		Vehicle: struct {
			BodyStyle         *string        `json:"body_style,omitempty"`
			Color             *string        `json:"color,omitempty"`
			CustomerId        int            `json:"customer_id"`
			Drivetrain        *string        `json:"drivetrain,omitempty"`
			Engine            *string        `json:"engine,omitempty"`
			FleetIdentifier   *string        `json:"fleet_identifier,omitempty"`
			LicensePlate      *string        `json:"license_plate,omitempty"`
			LicensePlateState *string        `json:"license_plate_state,omitempty"`
			Make              string         `json:"make"`
			Model             string         `json:"model"`
			Notes             *string        `json:"notes,omitempty"`
			OdometerReading   *int           `json:"odometer_reading,omitempty"`
			OdometerUnit      *string        `json:"odometer_unit,omitempty"`
			ProductionDate    *string        `json:"production_date,omitempty"`
			Submodel          *string        `json:"submodel,omitempty"`
			Transmission      *string        `json:"transmission,omitempty"`
			UnitNumber        *string        `json:"unit_number,omitempty"`
			VehicleTagIds     *[]interface{} `json:"vehicle_tag_ids,omitempty"`
			Vin               *string        `json:"vin,omitempty"`
			Year              int            `json:"year"`
		}{
			CustomerId: vehicleCreateCustomer,
			Make:       vehicleCreateMake,
			Model:      vehicleCreateModel,
			Year:       vehicleCreateYear,
		},
	}
	applyVehicleFlags(&body)

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

	body := generated.UpdateVehicleJSONRequestBody{
		Vehicle: struct {
			BodyStyle         *string `json:"body_style,omitempty"`
			Color             *string `json:"color,omitempty"`
			Drivetrain        *string `json:"drivetrain,omitempty"`
			Engine            *string `json:"engine,omitempty"`
			LicensePlate      *string `json:"license_plate,omitempty"`
			LicensePlateState *string `json:"license_plate_state,omitempty"`
			Make              string  `json:"make"`
			Model             *string `json:"model,omitempty"`
			Notes             *string `json:"notes,omitempty"`
			OdometerReading   *int    `json:"odometer_reading,omitempty"`
			OdometerUnit      *string `json:"odometer_unit,omitempty"`
			Submodel          *string `json:"submodel,omitempty"`
			Transmission      *string `json:"transmission,omitempty"`
			Vin               *string `json:"vin,omitempty"`
			Year              *int    `json:"year,omitempty"`
		}{
			Make: vehicleUpdateMake,
		},
	}
	applyVehicleUpdateFlags(&body)

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

	vin := args[0]
	resp, err := client.CheckVehicleDuplicate(context.Background(), generated.CheckVehicleDuplicateParams{Vin: &vin})
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("vehicles", args[0])}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func applyVehicleFlags(body *generated.CreateVehicleJSONRequestBody) {
	b := &body.Vehicle
	b.Vin = strPtr(vehicleVin)
	b.Submodel = strPtr(vehicleSubmodel)
	b.BodyStyle = strPtr(vehicleBodyStyle)
	b.Engine = strPtr(vehicleEngine)
	b.Transmission = strPtr(vehicleTransmission)
	b.Drivetrain = strPtr(vehicleDrivetrain)
	b.Color = strPtr(vehicleColor)
	b.LicensePlate = strPtr(vehiclePlate)
	b.LicensePlateState = strPtr(vehiclePlateState)
	b.UnitNumber = strPtr(vehicleUnitNumber)
	b.FleetIdentifier = strPtr(vehicleFleetIdentifier)
	b.ProductionDate = strPtr(vehicleProductionDate)
	b.Notes = strPtr(vehicleNotes)
	if vehicleOdometer != 0 {
		b.OdometerReading = &vehicleOdometer
	}
	b.OdometerUnit = strPtr(vehicleOdometerUnit)
	if len(vehicleTagIDs) > 0 {
		tags := make([]interface{}, len(vehicleTagIDs))
		for i, id := range vehicleTagIDs {
			tags[i] = id
		}
		b.VehicleTagIds = &tags
	}
}

func applyVehicleUpdateFlags(body *generated.UpdateVehicleJSONRequestBody) {
	b := &body.Vehicle
	b.Vin = strPtr(vehicleVin)
	b.Submodel = strPtr(vehicleSubmodel)
	b.BodyStyle = strPtr(vehicleBodyStyle)
	b.Engine = strPtr(vehicleEngine)
	b.Transmission = strPtr(vehicleTransmission)
	b.Drivetrain = strPtr(vehicleDrivetrain)
	b.Color = strPtr(vehicleColor)
	b.LicensePlate = strPtr(vehiclePlate)
	b.LicensePlateState = strPtr(vehiclePlateState)
	b.Notes = strPtr(vehicleNotes)
	if vehicleOdometer != 0 {
		b.OdometerReading = &vehicleOdometer
	}
	b.OdometerUnit = strPtr(vehicleOdometerUnit)
}

func runVehiclesTransfer(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("PATCH", "/vehicles/"+args[0]+"/transfer")

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}
	body := generated.TransferVehicleJSONRequestBody{
		CustomerId: vehicleTransferCustomerID,
		Mode:       vehicleTransferMode,
	}
	resp, err := client.TransferVehicle(context.Background(), id, body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("vehicles", args[0])}
	return output.Render(cmd.OutOrStdout(), data, "Vehicle transferred.", nil, opts)
}

func runVehiclesMerge(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("POST", "/vehicles/"+args[0]+"/merge")

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}
	body := generated.MergeVehicleJSONRequestBody{SourceVehicleId: vehicleMergeSourceID}
	resp, err := client.MergeVehicle(context.Background(), id, body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("vehicles", args[0])}
	return output.Render(cmd.OutOrStdout(), data, "Vehicle merged.", nil, opts)
}

func runVehiclesPrefill(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", "/vehicles/prefill")

	params := generated.PrefillVehicleParams{Vin: strPtr(vehiclePrefillVIN)}
	resp, err := client.PrefillVehicle(context.Background(), params)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("vehicles")}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runVehiclesLookup(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", "/vehicles/lookup")

	resp, err := client.LookupVehicle(context.Background(), args[0])
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("vehicles")}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runVehiclesWorkOrders(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", "/vehicles/"+args[0]+"/work_orders")

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}
	resp, err := client.ListVehicleWorkOrders(context.Background(), id)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("vehicles")}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}
