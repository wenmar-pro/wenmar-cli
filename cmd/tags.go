package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
	"github.com/wenmar-pro/wenmar-sdk/go/pkg/generated"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
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
	return runList(cmd, "tags", "/settings/tags", func(ctx context.Context, client *wenmar.Client) (any, error) {
		resp, err := client.ListTags(ctx)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runTagsCreate(cmd *cobra.Command, args []string) error {
	if tagsType == "vehicle" {
		return runCreate(cmd, "tags", "/vehicle_tags", "Vehicle tag created.", func() (any, error) {
			return generated.CreateVehicleTagJSONRequestBody{Name: tagsName}, nil
		}, func(ctx context.Context, client *wenmar.Client, body any) (any, error) {
			resp, err := client.CreateVehicleTag(ctx, body.(generated.CreateVehicleTagJSONRequestBody))
			if err != nil {
				return nil, err
			}
			return resp.JSON201, nil
		})
	}

	return runCreate(cmd, "tags", "/customer_tags", "Customer tag created.", func() (any, error) {
		return generated.CreateCustomerTagJSONRequestBody{Name: tagsName}, nil
	}, func(ctx context.Context, client *wenmar.Client, body any) (any, error) {
		resp, err := client.CreateCustomerTag(ctx, body.(generated.CreateCustomerTagJSONRequestBody))
		if err != nil {
			return nil, err
		}
		return resp.JSON201, nil
	})
}

func runTagsDelete(cmd *cobra.Command, args []string) error {
	return runTagsMutation(cmd, "Tag deleted.", func() generated.UpdateTagsJSONRequestBody {
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
		return body
	})
}

func runTagsRename(cmd *cobra.Command, args []string) error {
	return runTagsMutation(cmd, fmt.Sprintf("Tag %d renamed to %s.", tagsID, tagsName), func() generated.UpdateTagsJSONRequestBody {
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
		return body
	})
}

// runTagsMutation is the shared skeleton for tags delete/rename, which both
// PATCH /settings/tags and render the response.
func runTagsMutation(cmd *cobra.Command, summary string, bodyBuilder func() generated.UpdateTagsJSONRequestBody) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("PATCH", "/settings/tags")

	body := bodyBuilder()
	resp, err := client.UpdateTags(context.Background(), body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := resolveMode()
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("tags")}
	return output.Render(cmd.OutOrStdout(), data, summary, nil, opts)
}