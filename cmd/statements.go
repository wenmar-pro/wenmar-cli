package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/wenmar-pro/wenmar-cli/internal/output"
	"github.com/spf13/cobra"
)

var statementsCmd = &cobra.Command{
	Use:   "statements",
	Short: "Manage statements",
}

var statementsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List statements for a customer, paginated via the Link header",
	RunE:    runStatementsList,
}

var statementsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single statement by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatementsShow,
}

var statementsCustomerID int

func init() {
	statementsListCmd.Flags().IntVar(&statementsCustomerID, "customer-id", 0, "Customer ID (required)")
	statementsListCmd.MarkFlagRequired("customer-id")

	statementsCmd.AddCommand(statementsListCmd, statementsShowCmd)
	rootCmd.AddCommand(statementsCmd)
}

func runStatementsList(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", fmt.Sprintf("/customers/%d/statements", statementsCustomerID))

	resp, err := client.ListStatements(context.Background(), statementsCustomerID)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("statements")}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runStatementsShow(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", "/statements/"+args[0])

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	resp, err := client.ShowStatement(context.Background(), id)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("statements", args[0])}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}
