package cmd

import (
	"context"
	"fmt"

	"github.com/wenmar-pro/wenmar-cli/internal/output"
	"github.com/wenmar-pro/wenmar-sdk/go/pkg/generated"
	"github.com/spf13/cobra"
)

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "Manage customer and vehicle tags",
}

var tagsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all customer and vehicle tags",
	RunE:    runTagsList,
}

var tagsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a customer or vehicle tag",
	RunE:  runTagsCreate,
}

var tagsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a customer or vehicle tag",
	RunE:  runTagsDelete,
}

var tagsRenameCmd = &cobra.Command{
	Use:   "rename",
	Short: "Rename a customer or vehicle tag",
	RunE:  runTagsRename,
}

var (
	tagsType string
	tagsName string
	tagsID   int
)

func init() {
	tagsCreateCmd.Flags().StringVar(&tagsType, "type", "customer", "Tag type (customer or vehicle)")
	tagsCreateCmd.Flags().StringVar(&tagsName, "name", "", "Tag name (required)")
	tagsCreateCmd.MarkFlagRequired("name")
	tagsDeleteCmd.Flags().StringVar(&tagsType, "type", "customer", "Tag type (customer or vehicle)")
	tagsDeleteCmd.Flags().IntVar(&tagsID, "id", 0, "Tag ID (required)")
	tagsDeleteCmd.MarkFlagRequired("id")
	tagsRenameCmd.Flags().StringVar(&tagsType, "type", "customer", "Tag type (customer or vehicle)")
	tagsRenameCmd.Flags().IntVar(&tagsID, "id", 0, "Tag ID (required)")
	tagsRenameCmd.Flags().StringVar(&tagsName, "name", "", "New tag name (required)")
	tagsRenameCmd.MarkFlagRequired("id")
	tagsRenameCmd.MarkFlagRequired("name")

	tagsCmd.AddCommand(tagsListCmd, tagsCreateCmd, tagsDeleteCmd, tagsRenameCmd)
	rootCmd.AddCommand(tagsCmd)
}

func runTagsList(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", "/settings/tags")

	resp, err := client.ListTags(context.Background())
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("tags")}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

func runTagsCreate(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	if tagsType == "vehicle" {
		setRequest("POST", "/vehicle_tags")
		resp, err := client.CreateVehicleTag(context.Background(), generated.CreateVehicleTagJSONRequestBody{Name: tagsName})
		if err != nil {
			return err
		}
		data := extractData(resp.JSON201)
		mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
		opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("tags")}
		return output.Render(cmd.OutOrStdout(), data, "Vehicle tag created.", nil, opts)
	}

	setRequest("POST", "/customer_tags")
	resp, err := client.CreateCustomerTag(context.Background(), generated.CreateCustomerTagJSONRequestBody{Name: tagsName})
	if err != nil {
		return err
	}
	data := extractData(resp.JSON201)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("tags")}
	return output.Render(cmd.OutOrStdout(), data, "Customer tag created.", nil, opts)
}

func runTagsDelete(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("PATCH", "/settings/tags")

	body := generated.UpdateTagsJSONRequestBody{}
	destroy := "1"
	if tagsType == "vehicle" {
		body.VehicleTags = &[]struct {
			UnderscoreDestroy string `json:"_destroy"`
			Id                int    `json:"id"`
		}{{UnderscoreDestroy: destroy, Id: tagsID}}
	} else {
		body.CustomerTags = []struct {
			UnderscoreDestroy *string `json:"_destroy,omitempty"`
			Id                int     `json:"id"`
			Name              *string `json:"name,omitempty"`
		}{{UnderscoreDestroy: &destroy, Id: tagsID}}
	}
	resp, err := client.UpdateTags(context.Background(), body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("tags")}
	return output.Render(cmd.OutOrStdout(), data, "Tag deleted.", nil, opts)
}

func runTagsRename(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("PATCH", "/settings/tags")

	body := generated.UpdateTagsJSONRequestBody{}
	name := tagsName
	if tagsType == "vehicle" {
		body.VehicleTags = &[]struct {
			UnderscoreDestroy string `json:"_destroy"`
			Id                int    `json:"id"`
		}{{Id: tagsID}}
	} else {
		body.CustomerTags = []struct {
			UnderscoreDestroy *string `json:"_destroy,omitempty"`
			Id                int     `json:"id"`
			Name              *string `json:"name,omitempty"`
		}{{Id: tagsID, Name: &name}}
	}
	resp, err := client.UpdateTags(context.Background(), body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("tags")}
	return output.Render(cmd.OutOrStdout(), data, fmt.Sprintf("Tag %d renamed to %s.", tagsID, tagsName), nil, opts)
}
