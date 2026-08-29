package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-sdk/go/pkg/generated"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
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
	vehicleCreateMake         string
	vehicleCreateModel        string
	vehicleCreateYear         int
	vehicleCreateCustomer     int
	vehiclesDeleteDryRun      bool
	vehicleUpdateMake         string
	vehicleVin                string
	vehicleSubmodel           string
	vehicleBodyStyle          string
	vehicleEngine             string
	vehicleTransmission       string
	vehicleDrivetrain         string
	vehicleColor              string
	vehiclePlate              string
	vehiclePlateState         string
	vehicleOdometer           int
	vehicleOdometerUnit       string
	vehicleUnitNumber         string
	vehicleFleetIdentifier    string
	vehicleProductionDate     string
	vehicleNotes              string
	vehicleTagIDs             []int
	vehicleTransferCustomerID int
	vehicleTransferMode       string
	vehicleMergeSourceID      int
	vehiclePrefillVIN         string
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
	return runShow(cmd, args, "vehicles", "GET", idPath("/vehicles/"), func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		resp, err := client.ShowVehicle(ctx, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runVehiclesList(cmd *cobra.Command, args []string) error {
	return runList(cmd, "vehicles", "/vehicles", func(ctx context.Context, client *wenmar.Client) (any, error) {
		resp, err := client.ListVehicles(ctx)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runVehiclesCreate(cmd *cobra.Command, args []string) error {
	return runCreate(cmd, "vehicles", "/vehicles", "Vehicle created.", func() (any, error) {
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
		return body, nil
	}, func(ctx context.Context, client *wenmar.Client, body any) (any, error) {
		resp, err := client.CreateVehicle(ctx, body.(generated.CreateVehicleJSONRequestBody))
		if err != nil {
			return nil, err
		}
		return resp.JSON201, nil
	})
}

func runVehiclesUpdate(cmd *cobra.Command, args []string) error {
	return runUpdate(cmd, args, "vehicles", "/vehicles/", "", func(id int) (any, error) {
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
		return body, nil
	}, func(ctx context.Context, client *wenmar.Client, id int, body any) (any, error) {
		resp, err := client.UpdateVehicle(ctx, id, body.(generated.UpdateVehicleJSONRequestBody))
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runVehiclesDelete(cmd *cobra.Command, args []string) error {
	return runDelete(cmd, args, "Vehicle", "vehicles", "/vehicles/", vehiclesDeleteDryRun, func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		return client.DeleteVehicle(ctx, id)
	})
}

func runVehiclesDecodeVin(cmd *cobra.Command, args []string) error {
	return runList(cmd, "vehicles", "/vehicles/vin_decode", func(ctx context.Context, client *wenmar.Client) (any, error) {
		resp, err := client.DecodeVin(ctx, args[0])
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runVehiclesDuplicates(cmd *cobra.Command, args []string) error {
	return runList(cmd, "vehicles", "/vehicles/check_duplicate", func(ctx context.Context, client *wenmar.Client) (any, error) {
		vin := args[0]
		resp, err := client.CheckVehicleDuplicate(ctx, generated.CheckVehicleDuplicateParams{Vin: &vin})
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
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
	return runAction(cmd, args, "vehicles", "PATCH", func(a []string) string { return "/vehicles/" + a[0] + "/transfer" }, "Vehicle transferred.", func(id int) (any, error) {
		return generated.TransferVehicleJSONRequestBody{
			CustomerId: vehicleTransferCustomerID,
			Mode:       vehicleTransferMode,
		}, nil
	}, func(ctx context.Context, client *wenmar.Client, id int, body any) (any, error) {
		resp, err := client.TransferVehicle(ctx, id, body.(generated.TransferVehicleJSONRequestBody))
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runVehiclesMerge(cmd *cobra.Command, args []string) error {
	return runAction(cmd, args, "vehicles", "POST", func(a []string) string { return "/vehicles/" + a[0] + "/merge" }, "Vehicle merged.", func(id int) (any, error) {
		return generated.MergeVehicleJSONRequestBody{SourceVehicleId: vehicleMergeSourceID}, nil
	}, func(ctx context.Context, client *wenmar.Client, id int, body any) (any, error) {
		resp, err := client.MergeVehicle(ctx, id, body.(generated.MergeVehicleJSONRequestBody))
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runVehiclesPrefill(cmd *cobra.Command, args []string) error {
	return runList(cmd, "vehicles", "/vehicles/prefill", func(ctx context.Context, client *wenmar.Client) (any, error) {
		params := generated.PrefillVehicleParams{Vin: strPtr(vehiclePrefillVIN)}
		resp, err := client.PrefillVehicle(ctx, params)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runVehiclesLookup(cmd *cobra.Command, args []string) error {
	return runList(cmd, "vehicles", "/vehicles/lookup", func(ctx context.Context, client *wenmar.Client) (any, error) {
		resp, err := client.LookupVehicle(ctx, args[0])
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runVehiclesWorkOrders(cmd *cobra.Command, args []string) error {
	return runShow(cmd, args, "vehicles", "GET", func(a []string) string { return "/vehicles/" + a[0] + "/work_orders" }, func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		resp, err := client.ListVehicleWorkOrders(ctx, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}