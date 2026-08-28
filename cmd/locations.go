package cmd

import (
	"context"

	"github.com/wenmar-pro/wenmar-cli/internal/output"
	"github.com/spf13/cobra"
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
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", "/locations/"+args[0])

	resp, err := client.ShowLocation(context.Background(), args[0])
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("locations", args[0])}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}
