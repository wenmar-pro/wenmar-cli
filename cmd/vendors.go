package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/wenmar-pro/wenmar-cli/internal/output"
	"github.com/spf13/cobra"
)

var vendorsCmd = &cobra.Command{
	Use:   "vendors",
	Short: "Manage vendors",
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
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", "/vendors")

	resp, err := client.ListVendors(context.Background())
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("vendors")}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runVendorsShow(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", "/vendors/"+args[0])

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	resp, err := client.ShowVendor(context.Background(), id)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("vendors", args[0])}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}
