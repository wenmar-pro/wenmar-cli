//go:build !generated

package cmd

import (
	"context"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
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
	customersListQuery           string
	customersListType            string
	customersListHasVehicle      bool
	customersListHasBalance      bool
	customersListLastVisitMonths int
	customersListTagIDs          []int
	customersListPerPage         int
	customersListPage            int
	customersListAll             bool
)

var (
	customerCreateFullName   string
	customerUpdateFullName   string
	customerCompanyName      string
	customerFleetIdentifier  string
	customerBillingTerms     string
	customerCreditLimit      string
	customerTaxExempt        bool
	customerTaxExemptNumber  string
	customerNotes            string
	customerMarketingOptIn   bool
	customerDiscountPercent  string
	customerPoRequired       bool
	customerEmails           []string
	customerPhones           []string
	customerAddresses        []string
	customerTagIDs           []int
	customerRemovePhoneIDs   []int
)

func init() {
	customersListCmd.Flags().StringVar(&customersListQuery, "query", "", "Full-text search (name, email, phone, company, fleet ID, tag names)")
	customersListCmd.Flags().StringVar(&customersListType, "type", "", "Customer type (fleet or individual)")
	customersListCmd.Flags().BoolVar(&customersListHasVehicle, "has-vehicle", false, "Only customers with at least one vehicle")
	customersListCmd.Flags().BoolVar(&customersListHasBalance, "has-balance", false, "Only customers with an outstanding balance")
	customersListCmd.Flags().IntVar(&customersListLastVisitMonths, "last-visit-months", 0, "Inactive filter: last visit over N months ago")
	customersListCmd.Flags().IntSliceVar(&customersListTagIDs, "tag-ids", nil, "Filter by customer tag IDs (comma-separated)")
	customersListCmd.Flags().IntVar(&customersListPerPage, "per-page", 0, "Results per page (max 200)")
	customersListCmd.Flags().IntVar(&customersListPage, "page", 0, "Page number")
	customersListCmd.Flags().BoolVar(&customersListAll, "all", false, "Fetch all pages by following pagination links")
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
	customersUpdateCmd.Flags().IntSliceVar(&customerRemovePhoneIDs, "remove-phone", nil, "Phone ID to remove, repeatable")
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

func runCustomersList(cmd *cobra.Command, args []string) error {
	if customersListHasFilters() {
		params := customersListBuildParams()
		return runListPaginatedWithAll(cmd, "customers", "/customers", customersListAll, func(ctx context.Context, client *wenmar.Client) (any, *wenmar.Paginator, error) {
			resp, paginator, err := client.ListCustomersWithParamsWithPagination(ctx, params)
			if err != nil {
				return nil, nil, err
			}
			return resp.JSON200, paginator, nil
		})
	}

	return runListPaginatedWithAll(cmd, "customers", "/customers", customersListAll, func(ctx context.Context, client *wenmar.Client) (any, *wenmar.Paginator, error) {
		resp, paginator, err := client.ListCustomersWithPagination(ctx)
		if err != nil {
			return nil, nil, err
		}
		return resp.JSON200, paginator, nil
	})
}

func customersListHasFilters() bool {
	return customersListQuery != "" ||
		customersListType != "" ||
		customersListHasVehicle ||
		customersListHasBalance ||
		customersListLastVisitMonths > 0 ||
		len(customersListTagIDs) > 0 ||
		customersListPerPage > 0 ||
		customersListPage > 0
}

func customersListBuildParams() wenmar.ListCustomersParams {
	params := wenmar.ListCustomersParams{}
	if customersListQuery != "" {
		params.Query = &customersListQuery
	}
	if customersListType != "" {
		params.Type = &customersListType
	}
	if customersListHasVehicle {
		params.HasVehicle = &customersListHasVehicle
	}
	if customersListHasBalance {
		params.HasBalance = &customersListHasBalance
	}
	if customersListLastVisitMonths > 0 {
		params.LastVisitMonths = &customersListLastVisitMonths
	}
	if len(customersListTagIDs) > 0 {
		ids := make([]string, len(customersListTagIDs))
		for i, id := range customersListTagIDs {
			ids[i] = strconv.Itoa(id)
		}
		params.TagIds = &ids
	}
	if customersListPerPage > 0 {
		params.PerPage = &customersListPerPage
	}
	if customersListPage > 0 {
		params.Page = &customersListPage
	}
	return params
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
		req := wenmar.CreateCustomerRequest{
			FirstName: firstName,
			LastName:  lastName,
		}
		applyCustomerFlags(&req)
		return req, nil
	}, func(ctx context.Context, client *wenmar.Client, body any) (any, error) {
		resp, err := client.CreateCustomer(ctx, body.(wenmar.CreateCustomerRequest))
		if err != nil {
			return nil, err
		}
		return resp.JSON201, nil
	})
}

func runCustomersUpdate(cmd *cobra.Command, args []string) error {
	return runUpdate(cmd, args, "customers", idPath("/customers/"), "Customer updated.", func(id int) (any, error) {
		req := wenmar.UpdateCustomerRequest{}
		applyCustomerUpdateFlags(&req)
		return req, nil
	}, func(ctx context.Context, client *wenmar.Client, id int, body any) (any, error) {
		resp, err := client.UpdateCustomer(ctx, id, body.(wenmar.UpdateCustomerRequest))
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runCustomersMerge(cmd *cobra.Command, args []string) error {
	return runAction(cmd, args, "customers", "POST", func(a []string) string { return "/customers/" + a[0] + "/merge" }, "Customer merged.", func(id int) (any, error) {
		return wenmar.MergeCustomerRequest{SourceCustomerID: customerMergeSourceID}, nil
	}, func(ctx context.Context, client *wenmar.Client, id int, body any) (any, error) {
		resp, err := client.MergeCustomer(ctx, id, body.(wenmar.MergeCustomerRequest))
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
		params := wenmar.CheckCustomerDuplicateParams{
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

func parseLabelValue(s string) (string, string) {
	parts := strings.SplitN(s, "|", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", parts[0]
}

func applyCustomerFlags(req *wenmar.CreateCustomerRequest) {
	req.CompanyName = strPtr(customerCompanyName)
	req.FleetIdentifier = strPtr(customerFleetIdentifier)
	req.BillingTerms = strPtr(customerBillingTerms)
	req.CreditLimitCents = strPtr(customerCreditLimit)
	req.TaxExempt = boolPtr(customerTaxExempt)
	req.TaxExemptNumber = strPtr(customerTaxExemptNumber)
	req.Notes = strPtr(customerNotes)
	req.MarketingOptIn = boolPtr(customerMarketingOptIn)
	req.DiscountPercent = strPtr(customerDiscountPercent)
	req.PoRequired = boolPtr(customerPoRequired)

	if len(customerEmails) > 0 {
		emails := make([]wenmar.EmailAttribute, len(customerEmails))
		for i, e := range customerEmails {
			label, addr := parseLabelValue(e)
			emails[i] = wenmar.EmailAttribute{Email: addr, Label: label, Primary: i == 0}
		}
		req.Emails = &emails
	}

	if len(customerPhones) > 0 {
		phones := make([]wenmar.PhoneAttribute, len(customerPhones))
		for i, p := range customerPhones {
			label, num := parseLabelValue(p)
			phones[i] = wenmar.PhoneAttribute{Label: label, Number: num, Primary: i == 0}
		}
		req.Phones = &phones
	}

	if len(customerAddresses) > 0 {
		addresses := make([]wenmar.AddressAttribute, len(customerAddresses))
		for i, a := range customerAddresses {
			parts := strings.Split(a, "|")
			addr := wenmar.AddressAttribute{}
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
		req.Addresses = &addresses
	}
}

func applyCustomerUpdateFlags(req *wenmar.UpdateCustomerRequest) {
	req.CompanyName = strPtr(customerCompanyName)
	req.FleetIdentifier = strPtr(customerFleetIdentifier)
	req.BillingTerms = strPtr(customerBillingTerms)
	req.CreditLimitCents = strPtr(customerCreditLimit)
	req.TaxExempt = boolPtr(customerTaxExempt)
	req.Notes = strPtr(customerNotes)
	req.MarketingOptIn = boolPtr(customerMarketingOptIn)
	req.DiscountPercent = strPtr(customerDiscountPercent)
	req.PoRequired = boolPtr(customerPoRequired)

	if len(customerEmails) > 0 {
		emails := make([]wenmar.EmailUpdateAttribute, len(customerEmails))
		for i, e := range customerEmails {
			label, addr := parseLabelValue(e)
			emails[i] = wenmar.EmailUpdateAttribute{Email: addr, Label: &label}
		}
		req.Emails = &emails
	}

	if len(customerPhones) > 0 {
		phones := make([]wenmar.PhoneUpdateAttribute, len(customerPhones))
		for i, p := range customerPhones {
			label, num := parseLabelValue(p)
			prim := i == 0
			phones[i] = wenmar.PhoneUpdateAttribute{Label: &label, Number: &num, Primary: &prim}
		}
		req.Phones = &phones
	}

	if len(customerRemovePhoneIDs) > 0 {
		dest := true
		removes := make([]wenmar.PhoneUpdateAttribute, len(customerRemovePhoneIDs))
		for i, id := range customerRemovePhoneIDs {
			idv := id
			removes[i] = wenmar.PhoneUpdateAttribute{UnderscoreDestroy: &dest, Id: &idv}
		}
		if req.Phones != nil {
			*req.Phones = append(*req.Phones, removes...)
		} else {
			req.Phones = &removes
		}
	}
}
