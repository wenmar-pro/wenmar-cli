package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/wenmar-pro/wenmar-cli/internal/auth"
	"github.com/wenmar-pro/wenmar-cli/internal/errors"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
	"github.com/wenmar-pro/wenmar-sdk/go/pkg/generated"
	"github.com/spf13/cobra"
)

var customersCmd = &cobra.Command{
	Use:   "customers",
	Short: "Manage customers",
}

var customersListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all customers, paginated via the Link header",
	Run:     runCustomersList,
}

var customersShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single customer by ID",
	Args:  cobra.ExactArgs(1),
	Run:   runCustomersShow,
}

var customersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new customer",
	Run:   runCustomersCreate,
}

var (
	customerCreateFullName string
	customerCreateEmail    string
	customerCreatePhone    string
)

func init() {
	customersListCmd.Flags().Int("page", 0, "Page number")
	customersCreateCmd.Flags().StringVar(&customerCreateFullName, "full-name", "", "Customer full name (required)")
	customersCreateCmd.Flags().StringVar(&customerCreateEmail, "email", "", "Customer email")
	customersCreateCmd.Flags().StringVar(&customerCreatePhone, "phone", "", "Customer phone")
	customersCreateCmd.MarkFlagRequired("full-name")

	customersCmd.AddCommand(customersListCmd, customersShowCmd, customersCreateCmd)
	rootCmd.AddCommand(customersCmd)
}

func newSDKClient() (*wenmar.Client, error) {
	token, err := auth.ResolveToken(tokenFlag, "")
	if err != nil {
		return nil, err
	}
	baseURL := auth.ResolveBaseURL(baseURLFlag)
	return wenmar.NewClient(baseURL, token)
}

func runCustomersList(cmd *cobra.Command, args []string) {
	client, err := newSDKClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(errors.ExitGeneric)
	}

	pageFlag, _ := cmd.Flags().GetInt("page")
	var pagePtr *int
	if pageFlag > 0 {
		pagePtr = &pageFlag
	}

	resp, paginator, err := client.ListCustomersWithPagination(context.Background(), pagePtr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(errors.ExitCode(err))
	}

	data := extractData(resp.JSON200)
	summary := fmt.Sprintf("Page 1. More results: %v", paginator.HasNext())
	meta := &output.Meta{HasNext: paginator.HasNext()}

	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag}
	output.Render(os.Stdout, data, summary, meta, opts)
}

func runCustomersShow(cmd *cobra.Command, args []string) {
	client, err := newSDKClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(errors.ExitGeneric)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "id must be an integer")
		os.Exit(errors.ExitGeneric)
	}

	resp, err := client.ShowCustomer(context.Background(), id)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(errors.ExitCode(err))
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag}
	output.Render(os.Stdout, data, "", nil, opts)
}

func runCustomersCreate(cmd *cobra.Command, args []string) {
	client, err := newSDKClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(errors.ExitGeneric)
	}

	body := generated.CreateCustomerJSONRequestBody{
		Customer: &struct {
			Email    *string `json:"email,omitempty"`
			FullName string  `json:"full_name"`
			Phone    *string `json:"phone,omitempty"`
		}{
			FullName: customerCreateFullName,
		},
	}
	if customerCreateEmail != "" {
		body.Customer.Email = &customerCreateEmail
	}
	if customerCreatePhone != "" {
		body.Customer.Phone = &customerCreatePhone
	}

	resp, err := client.CreateCustomer(context.Background(), body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(errors.ExitCode(err))
	}

	data := extractData(resp.JSON201)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag}
	output.Render(os.Stdout, data, "Customer created.", nil, opts)
}

// extractData converts the generated response's JSON200 field to a
// generic map/slice for the output renderer. The generated types have
// pointer fields and nested structs — we marshal to JSON and back to
// get a clean map[string]any or []map[string]any.
func extractData(json200 any) any {
	if json200 == nil {
		return nil
	}
	// Use JSON round-trip to convert typed structs to generic maps
	b, err := jsonMarshal(json200)
	if err != nil {
		return json200
	}
	var result any
	if jsonUnmarshal(b, &result) != nil {
		return json200
	}
	if m, ok := result.(map[string]any); ok {
		if d, ok := m["data"]; ok {
			return d
		}
		return m
	}
	return result
}

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
func jsonUnmarshal(b []byte, v any) error {
	return json.Unmarshal(b, v)
}
