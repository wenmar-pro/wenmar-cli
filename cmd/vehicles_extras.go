package cmd

// vehicles_extras.go holds the two vehicle commands whose wrapper bodies
// use pointer-typed fields with omitempty tags (not per-operation
// derivable by the generator): create and update. Everything else under
// "vehicles" is generated (gen_vehicles.go).

import (
	"context"

	"github.com/spf13/cobra"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

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

var (
	vehicleCreateMake      string
	vehicleCreateModel     string
	vehicleCreateYear      int
	vehicleCreateCustomer  int
	vehicleUpdateMake      string
	vehicleVin             string
	vehicleSubmodel        string
	vehicleBodyStyle       string
	vehicleEngine          string
	vehicleTransmission    string
	vehicleDrivetrain      string
	vehicleColor           string
	vehiclePlate           string
	vehiclePlateState      string
	vehicleOdometer        int
	vehicleOdometerUnit    string
	vehicleUnitNumber      string
	vehicleFleetIdentifier string
	vehicleProductionDate  string
	vehicleNotes           string
	vehicleTagIDs          []int
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

	vehiclesCmd.AddCommand(vehiclesCreateCmd, vehiclesUpdateCmd)
}

func runVehiclesCreate(cmd *cobra.Command, args []string) error {
	return runCreate(cmd, "vehicles", "/vehicles", "Vehicle created.", func() (any, error) {
		req := wenmar.CreateVehicleRequest{
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
		applyVehicleFlags(&req)
		return req, nil
	}, func(ctx context.Context, client *wenmar.Client, body any) (any, error) {
		resp, err := client.CreateVehicle(ctx, body.(wenmar.CreateVehicleRequest))
		if err != nil {
			return nil, err
		}
		return resp.JSON201, nil
	})
}

func runVehiclesUpdate(cmd *cobra.Command, args []string) error {
	return runUpdate(cmd, args, "vehicles", idPath("/vehicles/"), "", func(id int) (any, error) {
		req := wenmar.UpdateVehicleRequest{
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
			}{Make: vehicleUpdateMake},
		}
		applyVehicleUpdateFlags(&req)
		return req, nil
	}, func(ctx context.Context, client *wenmar.Client, id int, body any) (any, error) {
		resp, err := client.UpdateVehicle(ctx, id, body.(wenmar.UpdateVehicleRequest))
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func applyVehicleFlags(req *wenmar.CreateVehicleRequest) {
	req.Vehicle.Vin = strPtr(vehicleVin)
	req.Vehicle.Submodel = strPtr(vehicleSubmodel)
	req.Vehicle.BodyStyle = strPtr(vehicleBodyStyle)
	req.Vehicle.Engine = strPtr(vehicleEngine)
	req.Vehicle.Transmission = strPtr(vehicleTransmission)
	req.Vehicle.Drivetrain = strPtr(vehicleDrivetrain)
	req.Vehicle.Color = strPtr(vehicleColor)
	req.Vehicle.LicensePlate = strPtr(vehiclePlate)
	req.Vehicle.LicensePlateState = strPtr(vehiclePlateState)
	req.Vehicle.UnitNumber = strPtr(vehicleUnitNumber)
	req.Vehicle.FleetIdentifier = strPtr(vehicleFleetIdentifier)
	req.Vehicle.ProductionDate = strPtr(vehicleProductionDate)
	req.Vehicle.Notes = strPtr(vehicleNotes)
	if vehicleOdometer != 0 {
		req.Vehicle.OdometerReading = &vehicleOdometer
	}
	req.Vehicle.OdometerUnit = strPtr(vehicleOdometerUnit)
	if len(vehicleTagIDs) > 0 {
		tags := make([]interface{}, len(vehicleTagIDs))
		for i, id := range vehicleTagIDs {
			tags[i] = id
		}
		req.Vehicle.VehicleTagIds = &tags
	}
}

func applyVehicleUpdateFlags(req *wenmar.UpdateVehicleRequest) {
	req.Vehicle.Vin = strPtr(vehicleVin)
	req.Vehicle.Submodel = strPtr(vehicleSubmodel)
	req.Vehicle.BodyStyle = strPtr(vehicleBodyStyle)
	req.Vehicle.Engine = strPtr(vehicleEngine)
	req.Vehicle.Transmission = strPtr(vehicleTransmission)
	req.Vehicle.Drivetrain = strPtr(vehicleDrivetrain)
	req.Vehicle.Color = strPtr(vehicleColor)
	req.Vehicle.LicensePlate = strPtr(vehiclePlate)
	req.Vehicle.LicensePlateState = strPtr(vehiclePlateState)
	req.Vehicle.Notes = strPtr(vehicleNotes)
	if vehicleOdometer != 0 {
		req.Vehicle.OdometerReading = &vehicleOdometer
	}
	req.Vehicle.OdometerUnit = strPtr(vehicleOdometerUnit)
}
