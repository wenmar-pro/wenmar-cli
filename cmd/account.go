package cmd

import (
	"context"

	"github.com/wenmar-pro/wenmar-cli/internal/output"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Show the current account",
}

var accountShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show account details",
	RunE:  runAccountShow,
}

func init() {
	accountCmd.AddCommand(accountShowCmd)
	rootCmd.AddCommand(accountCmd)
}

func runAccountShow(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", "/account")

	resp, err := client.ListAccount(context.Background())
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("account")}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}
