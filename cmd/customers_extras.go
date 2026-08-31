package cmd

// customers_extras.go holds the two customer commands whose bodies are
// not per-operation derivable: create (splitName + label|value parsing
// of emails/phones/addresses) and update (same + _destroy removals).
// Everything else under "customers" is generated (gen_customers.go).

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/spf13/cobra"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

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

var (
	customerCreateFullName  string
	customerUpdateFullName  string
	customerCompanyName     string
	customerFleetIdentifier string
	customerBillingTerms    string
	customerCreditLimit     string
	customerTaxExempt       bool
	customerTaxExemptNumber string
	customerNotes           string
	customerMarketingOptIn  bool
	customerDiscountPercent string
	customerPoRequired      bool
	customerEmails          []string
	customerPhones          []string
	customerAddresses       []string
	customerTagIDs          []int
	customerRemovePhoneIDs  []int
)

func init() {
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

	customersCmd.AddCommand(customersCreateCmd, customersUpdateCmd)
}

func runCustomersCreate(cmd *cobra.Command, args []string) error {
	return runCreate(cmd, "customers", "/customers", "Customer created.", func() (any, error) {
		firstName, lastName := splitName(customerCreateFullName)
		req := wenmar.CreateCustomerRequest{}
		req.Customer.FirstName = firstName
		req.Customer.LastName = lastName
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
	customer := map[string]any{
		"company_name":       strPtr(customerCompanyName),
		"fleet_identifier":   strPtr(customerFleetIdentifier),
		"billing_terms":      strPtr(customerBillingTerms),
		"credit_limit_cents": strPtr(customerCreditLimit),
		"tax_exempt":         boolPtr(customerTaxExempt),
		"tax_exempt_number":  strPtr(customerTaxExemptNumber),
		"notes":              strPtr(customerNotes),
		"marketing_opt_in":   boolPtr(customerMarketingOptIn),
		"discount_percent":   strPtr(customerDiscountPercent),
		"po_required":        boolPtr(customerPoRequired),
	}

	if len(customerEmails) > 0 {
		emails := make([]map[string]any, len(customerEmails))
		for i, e := range customerEmails {
			label, addr := parseLabelValue(e)
			emails[i] = map[string]any{"email": addr, "label": label, "primary": i == 0}
		}
		customer["emails_attributes"] = emails
	}

	if len(customerPhones) > 0 {
		phones := make([]map[string]any, len(customerPhones))
		for i, p := range customerPhones {
			label, num := parseLabelValue(p)
			phones[i] = map[string]any{"label": label, "number": num, "primary": i == 0}
		}
		customer["phones_attributes"] = phones
	}

	if len(customerAddresses) > 0 {
		addresses := make([]map[string]any, len(customerAddresses))
		for i, a := range customerAddresses {
			parts := strings.Split(a, "|")
			addr := map[string]any{"is_billing": i == 0}
			if len(parts) > 0 {
				addr["address1"] = parts[0]
			}
			if len(parts) > 1 {
				addr["city"] = parts[1]
			}
			if len(parts) > 2 {
				addr["state"] = parts[2]
			}
			if len(parts) > 3 {
				addr["postal_code"] = parts[3]
			}
			if len(parts) > 4 {
				addr["country"] = parts[4]
			}
			addresses[i] = addr
		}
		customer["addresses_attributes"] = addresses
	}

	mergeInto(req, map[string]any{"customer": customer})
}

func applyCustomerUpdateFlags(req *wenmar.UpdateCustomerRequest) {
	customer := map[string]any{
		"company_name":       strPtr(customerCompanyName),
		"fleet_identifier":   strPtr(customerFleetIdentifier),
		"billing_terms":      strPtr(customerBillingTerms),
		"credit_limit_cents": strPtr(customerCreditLimit),
		"tax_exempt":         boolPtr(customerTaxExempt),
		"notes":              strPtr(customerNotes),
		"marketing_opt_in":   boolPtr(customerMarketingOptIn),
		"discount_percent":   strPtr(customerDiscountPercent),
		"po_required":        boolPtr(customerPoRequired),
	}

	if len(customerEmails) > 0 {
		emails := make([]map[string]any, len(customerEmails))
		for i, e := range customerEmails {
			label, addr := parseLabelValue(e)
			emails[i] = map[string]any{"email": addr, "label": label}
		}
		customer["emails_attributes"] = emails
	}

	if len(customerPhones) > 0 {
		phones := make([]map[string]any, len(customerPhones))
		for i, p := range customerPhones {
			label, num := parseLabelValue(p)
			phones[i] = map[string]any{"label": label, "number": num, "primary": i == 0}
		}
		customer["phones_attributes"] = phones
	}

	if len(customerRemovePhoneIDs) > 0 {
		removes := make([]map[string]any, len(customerRemovePhoneIDs))
		for i, id := range customerRemovePhoneIDs {
			removes[i] = map[string]any{"_destroy": true, "id": id}
		}
		if existing, ok := customer["phones_attributes"].([]map[string]any); ok {
			customer["phones_attributes"] = append(existing, removes...)
		} else {
			customer["phones_attributes"] = removes
		}
	}

	mergeInto(req, map[string]any{"customer": customer})
}

// mergeInto unmarshals a map into an existing request struct, preserving any
// fields already set on it (e.g. first_name/last_name set by the caller).
func mergeInto(dst any, src map[string]any) {
	b, err := json.Marshal(src)
	if err != nil {
		return
	}
	_ = json.Unmarshal(b, dst)
}
