//go:build !generated

package cmd

import (
	"context"

	"github.com/spf13/cobra"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
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
	return runList(cmd, "account", "/account", func(ctx context.Context, client *wenmar.Client) (any, error) {
		resp, err := client.ListAccount(ctx)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}
