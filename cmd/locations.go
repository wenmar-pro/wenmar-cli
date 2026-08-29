//go:build !generated
package cmd

import (
	"context"

	"github.com/spf13/cobra"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var locationsCmd = &cobra.Command{
	Use:   "locations",
	Short: "Show location details",
}

var locationsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a location by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runLocationsShow,
}

func init() {
	locationsCmd.AddCommand(locationsShowCmd)
	rootCmd.AddCommand(locationsCmd)
}

func runLocationsShow(cmd *cobra.Command, args []string) error {
	return runShowStr(cmd, args, "locations", "GET", idPath("/locations/"), func(ctx context.Context, client *wenmar.Client, id string) (any, error) {
		resp, err := client.ShowLocation(ctx, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}