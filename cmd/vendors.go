//go:build !generated

package cmd

import (
	"context"

	"github.com/spf13/cobra"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var vendorsCmd = &cobra.Command{
	Use:   "vendors",
	Short: "Manage vendors",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var vendorsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all vendors, paginated via the Link header",
	RunE:    runVendorsList,
}

var vendorsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single vendor by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runVendorsShow,
}

func init() {
	vendorsCmd.AddCommand(vendorsListCmd, vendorsShowCmd)
	rootCmd.AddCommand(vendorsCmd)
}

func runVendorsList(cmd *cobra.Command, args []string) error {
	return runList(cmd, "vendors", "/vendors", func(ctx context.Context, client *wenmar.Client) (any, error) {
		resp, err := client.ListVendors(ctx)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runVendorsShow(cmd *cobra.Command, args []string) error {
	return runShow(cmd, args, "vendors", "GET", idPath("/vendors/"), func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		resp, err := client.ShowVendor(ctx, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}
