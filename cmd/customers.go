package cmd

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/errors"
	"github.com/wenmar-pro/wenmar-sdk/go/pkg/generated"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var customersCmd = &cobra.Command{
	Use:   "customers",
	Short: "Manage customers",
}

var customersListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all customers, paginated via the Link header",
	RunE:    runCustomersList,
}

var customersShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single customer by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runCustomersShow,
}

var customersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new customer",
	RunE:  runCustomersCreate,
}

var customersUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a customer by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runCustomersUpdate,
}

var customersMergeCmd = &cobra.Command{
	Use:   "merge <id>",
	Short: "Merge a source customer into this keeper",
	Args:  cobra.ExactArgs(1),
	RunE:  runCustomersMerge,
}

var customersLookupCmd = &cobra.Command{
	Use:   "lookup <query>",
	Short: "Search customers by name/email/phone",
	Args:  cobra.ExactArgs(1),
	RunE:  runCustomersLookup,
}

var customersDuplicatesCmd = &cobra.Command{
	Use:   "duplicates",
	Short: "Check for duplicate customers",
	RunE:  runCustomersDuplicates,
}

var customersVehiclesCmd = &cobra.Command{
	Use:   "vehicles <id>",
	Short: "List a customer's vehicles",
	Args:  cobra.ExactArgs(1),
	RunE:  runCustomersVehicles,
}

var customersWorkOrdersCmd = &cobra.Command{
	Use:   "work-orders <id>",
	Short: "List a customer's work orders",
	Args:  cobra.ExactArgs(1),
	RunE:  runCustomersWorkOrders,
}

var customersStatementsCmd = &cobra.Command{
	Use:   "statements <id>",
	Short: "List a customer's statements",
	Args:  cobra.ExactArgs(1),
	RunE:  runCustomersStatements,
}

var customerMergeSourceID int
var customerDuplicateFirstName string
var customerDuplicateLastName string
var customerDuplicateEmail string
var customerDuplicatePhone string

var (
	customerCreateFullName        string
	customerUpdateFullName        string
	customerCompanyName           string
	customerFleetIdentifier       string
	customerBillingTerms          string
	customerCreditLimit           string
	customerTaxExempt             bool
	customerTaxExemptNumber       string
	customerNotes                 string
	customerMarketingOptIn        bool
	customerDiscountPercent       string
	customerPoRequired            bool
	customerEmails                []string
	customerPhones                []string
	customerAddresses             []string
	customerTagIDs                []int
	customerRemoveEmailIDs        []int
	customerRemovePhoneIDs        []int
	customerRemoveAddressIDs      []int
)

func init() {
	customersListCmd.Flags().Int("page", 0, "Page number")
	customersCreateCmd.Flags().StringVar(&customerCreateFullName, "full-name", "", "Customer full name")
	customersCreateCmd.Flags().StringVar(&customerCompanyName, "company-name", "", "Company name")
	customersCreateCmd.Flags().StringVar(&customerFleetIdentifier, "fleet-identifier", "", "Fleet identifier")
	customersCreateCmd.Flags().StringVar(&customerBillingTerms, "billing-terms", "", "Billing terms (due_on_receipt, net_15, net_30, net_60)")
	customersCreateCmd.Flags().StringVar(&customerCreditLimit, "credit-limit", "", "Credit limit as a decimal string (e.g. 5000.00)")
	customersCreateCmd.Flags().BoolVar(&customerTaxExempt, "tax-exempt", false, "Tax exempt")
	customersCreateCmd.Flags().StringVar(&customerTaxExemptNumber, "tax-exempt-number", "", "Tax exempt number")
	customersCreateCmd.Flags().StringVar(&customerNotes, "notes", "", "Notes")
	customersCreateCmd.Flags().BoolVar(&customerMarketingOptIn, "marketing-opt-in", false, "Marketing opt-in")
	customersCreateCmd.Flags().StringVar(&customerDiscountPercent, "discount-percent", "", "Discount percent")
	customersCreateCmd.Flags().BoolVar(&customerPoRequired, "po-required", false, "PO required")
	customersCreateCmd.Flags().StringArrayVar(&customerEmails, "email", nil, "Email (label|address or address), repeatable")
	customersCreateCmd.Flags().StringArrayVar(&customerPhones, "phone", nil, "Phone (label|number or number), repeatable")
	customersCreateCmd.Flags().StringArrayVar(&customerAddresses, "address", nil, "Address (address1|city|state|postal_code|country), repeatable")
	customersCreateCmd.Flags().IntSliceVar(&customerTagIDs, "tag-id", nil, "Tag ID, repeatable")
	customersUpdateCmd.Flags().StringVar(&customerUpdateFullName, "full-name", "", "Customer full name")
	customersUpdateCmd.Flags().StringVar(&customerCompanyName, "company-name", "", "Company name")
	customersUpdateCmd.Flags().StringVar(&customerFleetIdentifier, "fleet-identifier", "", "Fleet identifier")
	customersUpdateCmd.Flags().StringVar(&customerBillingTerms, "billing-terms", "", "Billing terms (due_on_receipt, net_15, net_30, net_60)")
	customersUpdateCmd.Flags().StringVar(&customerCreditLimit, "credit-limit", "", "Credit limit as a decimal string (e.g. 5000.00)")
	customersUpdateCmd.Flags().BoolVar(&customerTaxExempt, "tax-exempt", false, "Tax exempt")
	customersUpdateCmd.Flags().StringVar(&customerTaxExemptNumber, "tax-exempt-number", "", "Tax exempt number")
	customersUpdateCmd.Flags().StringVar(&customerNotes, "notes", "", "Notes")
	customersUpdateCmd.Flags().BoolVar(&customerMarketingOptIn, "marketing-opt-in", false, "Marketing opt-in")
	customersUpdateCmd.Flags().StringVar(&customerDiscountPercent, "discount-percent", "", "Discount percent")
	customersUpdateCmd.Flags().BoolVar(&customerPoRequired, "po-required", false, "PO required")
	customersUpdateCmd.Flags().StringArrayVar(&customerEmails, "email", nil, "Email (label|address or address), repeatable")
	customersUpdateCmd.Flags().StringArrayVar(&customerPhones, "phone", nil, "Phone (label|number or number), repeatable")
	customersUpdateCmd.Flags().StringArrayVar(&customerAddresses, "address", nil, "Address (address1|city|state|postal_code|country), repeatable")
	customersUpdateCmd.Flags().IntSliceVar(&customerTagIDs, "tag-id", nil, "Tag ID, repeatable")
	customersUpdateCmd.Flags().IntSliceVar(&customerRemoveEmailIDs, "remove-email", nil, "Email ID to remove, repeatable")
	customersUpdateCmd.Flags().IntSliceVar(&customerRemovePhoneIDs, "remove-phone", nil, "Phone ID to remove, repeatable")
	customersUpdateCmd.Flags().IntSliceVar(&customerRemoveAddressIDs, "remove-address", nil, "Address ID to remove, repeatable")
	customersMergeCmd.Flags().IntVar(&customerMergeSourceID, "source-id", 0, "Source customer ID to merge into keeper (required)")
	customersMergeCmd.MarkFlagRequired("source-id")
	customersDuplicatesCmd.Flags().StringVar(&customerDuplicateFirstName, "first-name", "", "First name")
	customersDuplicatesCmd.Flags().StringVar(&customerDuplicateLastName, "last-name", "", "Last name")
	customersDuplicatesCmd.Flags().StringVar(&customerDuplicateEmail, "email", "", "Email")
	customersDuplicatesCmd.Flags().StringVar(&customerDuplicatePhone, "phone", "", "Phone")

	customersCmd.AddCommand(customersListCmd, customersShowCmd, customersCreateCmd, customersUpdateCmd,
		customersMergeCmd, customersLookupCmd, customersDuplicatesCmd, customersVehiclesCmd, customersWorkOrdersCmd, customersStatementsCmd)
	rootCmd.AddCommand(customersCmd)
}

// setRequest records the HTTP method and path for the current command so the
// error handler can show which request failed.
func setRequest(method, path string) {
	if currentDebugInfo == nil {
		currentDebugInfo = &errors.DebugInfo{}
	}
	currentDebugInfo.Method = method
	currentDebugInfo.Path = path
}

func runCustomersList(cmd *cobra.Command, args []string) error {
	return runListPaginated(cmd, "customers", "/customers", func(ctx context.Context, client *wenmar.Client) (any, *wenmar.Paginator, error) {
		resp, paginator, err := client.ListCustomersWithPagination(ctx)
		if err != nil {
			return nil, nil, err
		}
		return resp.JSON200, paginator, nil
	})
}

func runCustomersShow(cmd *cobra.Command, args []string) error {
	return runShow(cmd, args, "customers", "GET", idPath("/customers/"), func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		resp, err := client.ShowCustomer(ctx, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runCustomersCreate(cmd *cobra.Command, args []string) error {
	return runCreate(cmd, "customers", "/customers", "Customer created.", func() (any, error) {
		firstName, lastName := splitName(customerCreateFullName)
		body := generated.CreateCustomerJSONRequestBody{
			Customer: struct {
				AddressesAttributes *[]struct {
					Address1   string `json:"address1"`
					City       string `json:"city"`
					Country    string `json:"country"`
					IsBilling  bool   `json:"is_billing"`
					PostalCode string `json:"postal_code"`
					State      string `json:"state"`
				} `json:"addresses_attributes,omitempty"`
				BillingTerms     *string `json:"billing_terms,omitempty"`
				CompanyName      *string `json:"company_name,omitempty"`
				CreditLimitCents *string `json:"credit_limit_cents,omitempty"`
				DiscountPercent  *string `json:"discount_percent,omitempty"`
				EmailsAttributes *[]struct {
					Email   string `json:"email"`
					Label   string `json:"label"`
					Primary bool   `json:"primary"`
				} `json:"emails_attributes,omitempty"`
				FirstName        string  `json:"first_name"`
				FleetIdentifier  *string `json:"fleet_identifier,omitempty"`
				LastName         string  `json:"last_name"`
				MarketingOptIn   *bool   `json:"marketing_opt_in,omitempty"`
				Notes            *string `json:"notes,omitempty"`
				PhonesAttributes *[]struct {
					Label   string `json:"label"`
					Number  string `json:"number"`
					Primary bool   `json:"primary"`
				} `json:"phones_attributes,omitempty"`
				PoRequired      *bool          `json:"po_required,omitempty"`
				TagIds          *[]interface{} `json:"tag_ids,omitempty"`
				TaxExempt       *bool          `json:"tax_exempt,omitempty"`
				TaxExemptNumber *string        `json:"tax_exempt_number,omitempty"`
			}{
				FirstName: firstName,
				LastName:  lastName,
			},
		}
		applyCustomerFlags(&body)
		return body, nil
	}, func(ctx context.Context, client *wenmar.Client, body any) (any, error) {
		resp, err := client.CreateCustomer(ctx, body.(generated.CreateCustomerJSONRequestBody))
		if err != nil {
			return nil, err
		}
		return resp.JSON201, nil
	})
}

func runCustomersUpdate(cmd *cobra.Command, args []string) error {
	return runUpdate(cmd, args, "customers", "/customers/", "Customer updated.", func(id int) (any, error) {
		body := generated.UpdateCustomerJSONRequestBody{}
		applyCustomerUpdateFlags(&body)
		return body, nil
	}, func(ctx context.Context, client *wenmar.Client, id int, body any) (any, error) {
		resp, err := client.UpdateCustomer(ctx, id, body.(generated.UpdateCustomerJSONRequestBody))
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runCustomersMerge(cmd *cobra.Command, args []string) error {
	return runAction(cmd, args, "customers", "POST", func(a []string) string { return "/customers/" + a[0] + "/merge" }, "Customer merged.", func(id int) (any, error) {
		return generated.MergeCustomerJSONRequestBody{SourceCustomerId: customerMergeSourceID}, nil
	}, func(ctx context.Context, client *wenmar.Client, id int, body any) (any, error) {
		resp, err := client.MergeCustomer(ctx, id, body.(generated.MergeCustomerJSONRequestBody))
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runCustomersLookup(cmd *cobra.Command, args []string) error {
	return runList(cmd, "customers", "/customers/lookup", func(ctx context.Context, client *wenmar.Client) (any, error) {
		resp, err := client.LookupCustomer(ctx, args[0])
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runCustomersDuplicates(cmd *cobra.Command, args []string) error {
	return runList(cmd, "customers", "/customers/check_duplicate", func(ctx context.Context, client *wenmar.Client) (any, error) {
		params := generated.CheckCustomerDuplicateParams{
			FirstName: strPtr(customerDuplicateFirstName),
			LastName:  strPtr(customerDuplicateLastName),
			Email:     strPtr(customerDuplicateEmail),
		}
		resp, err := client.CheckCustomerDuplicate(ctx, params)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runCustomersVehicles(cmd *cobra.Command, args []string) error {
	return runShow(cmd, args, "customers", "GET", func(a []string) string { return "/customers/" + a[0] + "/vehicles" }, func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		resp, err := client.ListCustomerVehicles(ctx, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runCustomersWorkOrders(cmd *cobra.Command, args []string) error {
	return runShow(cmd, args, "customers", "GET", func(a []string) string { return "/customers/" + a[0] + "/work_orders" }, func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		resp, err := client.ListCustomerWorkOrders(ctx, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runCustomersStatements(cmd *cobra.Command, args []string) error {
	return runShow(cmd, args, "customers", "GET", func(a []string) string { return "/customers/" + a[0] + "/statements" }, func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		resp, err := client.ListCustomerStatements(ctx, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func splitName(full string) (string, string) {
	parts := strings.Fields(full)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func boolPtr(b bool) *bool { return &b }

func parseLabelValue(s string) (string, string) {
	parts := strings.SplitN(s, "|", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", parts[0]
}

func applyCustomerFlags(body *generated.CreateCustomerJSONRequestBody) {
	b := &body.Customer
	b.CompanyName = strPtr(customerCompanyName)
	b.FleetIdentifier = strPtr(customerFleetIdentifier)
	b.BillingTerms = strPtr(customerBillingTerms)
	b.CreditLimitCents = strPtr(customerCreditLimit)
	b.TaxExempt = boolPtr(customerTaxExempt)
	b.TaxExemptNumber = strPtr(customerTaxExemptNumber)
	b.Notes = strPtr(customerNotes)
	b.MarketingOptIn = boolPtr(customerMarketingOptIn)
	b.DiscountPercent = strPtr(customerDiscountPercent)
	b.PoRequired = boolPtr(customerPoRequired)

	if len(customerEmails) > 0 {
		emails := make([]struct {
			Email   string `json:"email"`
			Label   string `json:"label"`
			Primary bool   `json:"primary"`
		}, len(customerEmails))
		for i, e := range customerEmails {
			label, addr := parseLabelValue(e)
			emails[i] = struct {
				Email   string `json:"email"`
				Label   string `json:"label"`
				Primary bool   `json:"primary"`
			}{Email: addr, Label: label, Primary: i == 0}
		}
		b.EmailsAttributes = &emails
	}

	if len(customerPhones) > 0 {
		phones := make([]struct {
			Label   string `json:"label"`
			Number  string `json:"number"`
			Primary bool   `json:"primary"`
		}, len(customerPhones))
		for i, p := range customerPhones {
			label, num := parseLabelValue(p)
			phones[i] = struct {
				Label   string `json:"label"`
				Number  string `json:"number"`
				Primary bool   `json:"primary"`
			}{Label: label, Number: num, Primary: i == 0}
		}
		b.PhonesAttributes = &phones
	}

	if len(customerAddresses) > 0 {
		addresses := make([]struct {
			Address1   string `json:"address1"`
			City       string `json:"city"`
			Country    string `json:"country"`
			IsBilling  bool   `json:"is_billing"`
			PostalCode string `json:"postal_code"`
			State      string `json:"state"`
		}, len(customerAddresses))
		for i, a := range customerAddresses {
			parts := strings.Split(a, "|")
			addr := struct {
				Address1   string `json:"address1"`
				City       string `json:"city"`
				Country    string `json:"country"`
				IsBilling  bool   `json:"is_billing"`
				PostalCode string `json:"postal_code"`
				State      string `json:"state"`
			}{}
			if len(parts) > 0 {
				addr.Address1 = parts[0]
			}
			if len(parts) > 1 {
				addr.City = parts[1]
			}
			if len(parts) > 2 {
				addr.State = parts[2]
			}
			if len(parts) > 3 {
				addr.PostalCode = parts[3]
			}
			if len(parts) > 4 {
				addr.Country = parts[4]
			}
			addr.IsBilling = i == 0
			addresses[i] = addr
		}
		b.AddressesAttributes = &addresses
	}
}

func applyCustomerUpdateFlags(body *generated.UpdateCustomerJSONRequestBody) {
	b := &body.Customer
	b.CompanyName = strPtr(customerCompanyName)
	b.FleetIdentifier = strPtr(customerFleetIdentifier)
	b.BillingTerms = strPtr(customerBillingTerms)
	b.CreditLimitCents = strPtr(customerCreditLimit)
	b.TaxExempt = boolPtr(customerTaxExempt)
	b.Notes = strPtr(customerNotes)
	b.MarketingOptIn = boolPtr(customerMarketingOptIn)
	b.DiscountPercent = strPtr(customerDiscountPercent)
	b.PoRequired = boolPtr(customerPoRequired)

	if len(customerEmails) > 0 {
		emails := make([]struct {
			Email string  `json:"email"`
			Id    *int    `json:"id,omitempty"`
			Label *string `json:"label,omitempty"`
		}, len(customerEmails))
		for i, e := range customerEmails {
			label, addr := parseLabelValue(e)
			emails[i] = struct {
				Email string  `json:"email"`
				Id    *int    `json:"id,omitempty"`
				Label *string `json:"label,omitempty"`
			}{Email: addr, Label: &label}
			_ = i
		}
		b.EmailsAttributes = &emails
	}

	if len(customerRemoveEmailIDs) > 0 {
		removes := make([]struct {
			Email string  `json:"email"`
			Id    *int    `json:"id,omitempty"`
			Label *string `json:"label,omitempty"`
		}, 0)
		_ = removes // email _destroy not supported by the generated update schema
	}

	if len(customerPhones) > 0 {
		phones := make([]struct {
			UnderscoreDestroy *bool   `json:"_destroy,omitempty"`
			Id                *int    `json:"id,omitempty"`
			Label             *string `json:"label,omitempty"`
			Number            *string `json:"number,omitempty"`
			Primary           *bool   `json:"primary,omitempty"`
		}, len(customerPhones))
		for i, p := range customerPhones {
			label, num := parseLabelValue(p)
			prim := i == 0
			phones[i] = struct {
				UnderscoreDestroy *bool   `json:"_destroy,omitempty"`
				Id                *int    `json:"id,omitempty"`
				Label             *string `json:"label,omitempty"`
				Number            *string `json:"number,omitempty"`
				Primary           *bool   `json:"primary,omitempty"`
			}{Label: &label, Number: &num, Primary: &prim}
		}
		if b.PhonesAttributes != nil {
			*b.PhonesAttributes = append(*b.PhonesAttributes, phones...)
		} else {
			b.PhonesAttributes = &phones
		}
	}

	if len(customerRemovePhoneIDs) > 0 {
		dest := true
		removes := make([]struct {
			UnderscoreDestroy *bool   `json:"_destroy,omitempty"`
			Id                *int    `json:"id,omitempty"`
			Label             *string `json:"label,omitempty"`
			Number            *string `json:"number,omitempty"`
			Primary           *bool   `json:"primary,omitempty"`
		}, len(customerRemovePhoneIDs))
		for i, id := range customerRemovePhoneIDs {
			idv := id
			removes[i] = struct {
				UnderscoreDestroy *bool   `json:"_destroy,omitempty"`
				Id                *int    `json:"id,omitempty"`
				Label             *string `json:"label,omitempty"`
				Number            *string `json:"number,omitempty"`
				Primary           *bool   `json:"primary,omitempty"`
			}{UnderscoreDestroy: &dest, Id: &idv}
		}
		if b.PhonesAttributes != nil {
			*b.PhonesAttributes = append(*b.PhonesAttributes, removes...)
		} else {
			b.PhonesAttributes = &removes
		}
	}
}

// extractData converts the generated response's JSON200 field to a
// generic map/slice for the output renderer. The generated types have
// pointer fields and nested structs — we marshal to JSON and back to
// get a clean map[string]any or []map[string]any.
func extractData(json200 any) any {
	if json200 == nil {
		return nil
	}
	b, err := json.Marshal(json200)
	if err != nil {
		return json200
	}
	var result any
	if json.Unmarshal(b, &result) != nil {
		return json200
	}
	if m, ok := result.(map[string]any); ok {
		return m
	}
	return result
}