//go:build !generated

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var statementsCmd = &cobra.Command{
	Use:   "statements",
	Short: "Manage statements",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
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
	return runList(cmd, "statements", fmt.Sprintf("/customers/%d/statements", statementsCustomerID), func(ctx context.Context, client *wenmar.Client) (any, error) {
		resp, err := client.ListStatements(ctx, statementsCustomerID)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runStatementsShow(cmd *cobra.Command, args []string) error {
	return runShow(cmd, args, "statements", "GET", idPath("/statements/"), func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		resp, err := client.ShowStatement(ctx, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}
