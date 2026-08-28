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
	client, err := newSDKClient()
	if err != nil {
		return err
	}

	resp, err := client.ListAccount(context.Background())
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveMode(mdFlag, jsonFlag, agentFlag, quietFlag, jqFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: output.CaptureBreadcrumbs()}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}
