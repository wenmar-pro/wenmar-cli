package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/wenmar-pro/wenmar-cli/internal/auth"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
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

var (
	customerCreateFullName string
	customerCreateEmail    string
	customerCreatePhone    string
	customerUpdateFullName string
)

func init() {
	customersListCmd.Flags().Int("page", 0, "Page number")
	customersCreateCmd.Flags().StringVar(&customerCreateFullName, "full-name", "", "Customer full name (required)")
	customersCreateCmd.Flags().StringVar(&customerCreateEmail, "email", "", "Customer email")
	customersCreateCmd.Flags().StringVar(&customerCreatePhone, "phone", "", "Customer phone")
	customersCreateCmd.MarkFlagRequired("full-name")
	customersUpdateCmd.Flags().StringVar(&customerUpdateFullName, "full-name", "", "Customer full name")

	customersCmd.AddCommand(customersListCmd, customersShowCmd, customersCreateCmd, customersUpdateCmd)
	rootCmd.AddCommand(customersCmd)
}

func newSDKClient() (*wenmar.Client, error) {
	configPath := configPathFlag
	if configPath == "" {
		p, err := config.ConfigPath()
		if err == nil {
			configPath = p
		}
	}
	token, err := auth.ResolveTokenFrom(tokenFlag, configPath)
	if err != nil {
		return nil, err
	}
	baseURL := auth.ResolveBaseURLFrom(baseURLFlag, configPath)
	return wenmar.NewClient(baseURL, token)
}

func runCustomersList(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}

	resp, paginator, err := client.ListCustomersWithPagination(context.Background())
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	summary := fmt.Sprintf("Page 1. More results: %v", paginator.HasNext())
	meta := &output.Meta{HasNext: paginator.HasNext()}

	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), data, summary, meta, opts)
}

func runCustomersShow(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	resp, err := client.ShowCustomer(context.Background(), id)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runCustomersCreate(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}

	firstName, lastName := splitName(customerCreateFullName)
	body := generated.CreateCustomerJSONRequestBody{
		Customer: struct {
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		}{
			FirstName: firstName,
			LastName:  lastName,
		},
	}

	resp, err := client.CreateCustomer(context.Background(), body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON201)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), data, "Customer created.", nil, opts)
}

func runCustomersUpdate(cmd *cobra.Command, args []string) error {
	client, err := newSDKClient()
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	body := generated.UpdateCustomerJSONRequestBody{}
	resp, err := client.UpdateCustomer(context.Background(), id, body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), data, "Customer updated.", nil, opts)
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
		return m
	}
	return result
}

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
func jsonUnmarshal(b []byte, v any) error {
	return json.Unmarshal(b, v)
}
