//go:build !generated

package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

var serviceCategoriesCmd = &cobra.Command{
	Use:   "service-categories",
	Short: "Manage service categories",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var serviceCategoriesListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all service categories",
	RunE:    runServiceCategoriesList,
}

var serviceCategoriesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new service category",
	RunE:  runServiceCategoriesCreate,
}

var serviceCategoriesUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a service category by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceCategoriesUpdate,
}

var serviceCategoriesDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a service category by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceCategoriesDelete,
}

var serviceCategoriesDeactivateCmd = &cobra.Command{
	Use:   "deactivate <id>",
	Short: "Deactivate a service category by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceCategoriesDeactivate,
}

var serviceCategoriesReactivateCmd = &cobra.Command{
	Use:   "reactivate <id>",
	Short: "Reactivate a service category by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceCategoriesReactivate,
}

var serviceCategoriesMoveUpCmd = &cobra.Command{
	Use:   "move-up <id>",
	Short: "Move a service category up one position",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceCategoriesMoveUp,
}

var serviceCategoriesMoveDownCmd = &cobra.Command{
	Use:   "move-down <id>",
	Short: "Move a service category down one position",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceCategoriesMoveDown,
}

var serviceCategoriesSeedDefaultsCmd = &cobra.Command{
	Use:   "seed-defaults",
	Short: "Seed default service categories for the account",
	RunE:  runServiceCategoriesSeedDefaults,
}

var (
	serviceCategoryCreateName        string
	serviceCategoryCreateServiceType string
	serviceCategoryCreateIcon        string
	serviceCategoryUpdateName        string
	serviceCategoryDeleteDryRun      bool
)

func init() {
	serviceCategoriesCreateCmd.Flags().StringVar(&serviceCategoryCreateName, "name", "", "Category name (required)")
	serviceCategoriesCreateCmd.Flags().StringVar(&serviceCategoryCreateServiceType, "service-type", "maintenance", "Service type (maintenance, repair, diagnostic, recall, sublet)")
	serviceCategoriesCreateCmd.Flags().StringVar(&serviceCategoryCreateIcon, "icon", "wrench", "Icon (wrench, disc, engine, etc.)")
	serviceCategoriesCreateCmd.MarkFlagRequired("name")

	serviceCategoriesUpdateCmd.Flags().StringVar(&serviceCategoryUpdateName, "name", "", "New category name (required)")
	serviceCategoriesUpdateCmd.MarkFlagRequired("name")

	serviceCategoriesDeleteCmd.Flags().BoolVar(&serviceCategoryDeleteDryRun, "dry-run", false, "Preview what would be deleted without making an API call")

	serviceCategoriesCmd.AddCommand(
		serviceCategoriesListCmd,
		serviceCategoriesCreateCmd,
		serviceCategoriesUpdateCmd,
		serviceCategoriesDeleteCmd,
		serviceCategoriesDeactivateCmd,
		serviceCategoriesReactivateCmd,
		serviceCategoriesMoveUpCmd,
		serviceCategoriesMoveDownCmd,
		serviceCategoriesSeedDefaultsCmd,
	)
	rootCmd.AddCommand(serviceCategoriesCmd)
}

func runServiceCategoriesList(cmd *cobra.Command, args []string) error {
	return runList(cmd, "service_categories", "/service_categories", func(ctx context.Context, client *wenmar.Client) (any, error) {
		resp, err := client.ListServiceCategories(ctx)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runServiceCategoriesCreate(cmd *cobra.Command, args []string) error {
	return runCreate(cmd, "service_categories", "/service_categories", "Service category created.", func() (any, error) {
		return wenmar.CreateServiceCategoryRequest{
			ServiceCategory: struct {
				Icon        string `json:"icon"`
				Name        string `json:"name"`
				ServiceType string `json:"service_type"`
			}{
				Name:        serviceCategoryCreateName,
				ServiceType: serviceCategoryCreateServiceType,
				Icon:        serviceCategoryCreateIcon,
			},
		}, nil
	}, func(ctx context.Context, client *wenmar.Client, body any) (any, error) {
		resp, err := client.CreateServiceCategory(ctx, body.(wenmar.CreateServiceCategoryRequest))
		if err != nil {
			return nil, err
		}
		return resp.JSON201, nil
	})
}

func runServiceCategoriesUpdate(cmd *cobra.Command, args []string) error {
	return runUpdate(cmd, args, "service_categories", idPath("/service_categories/"), "Service category updated.", func(id int) (any, error) {
		return wenmar.UpdateServiceCategoryRequest{
			ServiceCategory: struct {
				Name string `json:"name"`
			}{Name: serviceCategoryUpdateName},
		}, nil
	}, func(ctx context.Context, client *wenmar.Client, id int, body any) (any, error) {
		resp, err := client.UpdateServiceCategory(ctx, id, body.(wenmar.UpdateServiceCategoryRequest))
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runServiceCategoriesDelete(cmd *cobra.Command, args []string) error {
	return runDelete(cmd, args, "Service category", "service_categories", idPath("/service_categories/"), serviceCategoryDeleteDryRun, func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		resp, err := client.DeleteServiceCategory(ctx, id)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runServiceCategoriesDeactivate(cmd *cobra.Command, args []string) error {
	return runServiceCategoryAction(cmd, args, "PATCH", func(a []string) string { return "/service_categories/" + a[0] + "/deactivate" }, "Service category deactivated.", func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		resp, err := client.DeactivateServiceCategory(ctx, id, wenmar.DeactivateServiceCategoryRequest{})
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runServiceCategoriesReactivate(cmd *cobra.Command, args []string) error {
	return runServiceCategoryAction(cmd, args, "PATCH", func(a []string) string { return "/service_categories/" + a[0] + "/reactivate" }, "Service category reactivated.", func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		resp, err := client.ReactivateServiceCategory(ctx, id, wenmar.ReactivateServiceCategoryRequest{})
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runServiceCategoriesMoveUp(cmd *cobra.Command, args []string) error {
	return runServiceCategoryAction(cmd, args, "PATCH", func(a []string) string { return "/service_categories/" + a[0] + "/move_up" }, "Service category moved up.", func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		resp, err := client.MoveUpServiceCategory(ctx, id, wenmar.MoveUpServiceCategoryRequest{})
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runServiceCategoriesMoveDown(cmd *cobra.Command, args []string) error {
	return runServiceCategoryAction(cmd, args, "PATCH", func(a []string) string { return "/service_categories/" + a[0] + "/move_down" }, "Service category moved down.", func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
		resp, err := client.MoveDownServiceCategory(ctx, id, wenmar.MoveDownServiceCategoryRequest{})
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runServiceCategoriesSeedDefaults(cmd *cobra.Command, args []string) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("POST", "/service_categories/seed_defaults")

	resp, err := client.SeedDefaultsServiceCategories(context.Background(), wenmar.SeedDefaultsServiceCategoriesRequest{})
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	summary := "Default service categories seeded."
	if resp.JSON200 != nil {
		summary = fmt.Sprintf("%d default service categories created.", resp.JSON200.Created)
	}
	mode := resolveMode()
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("service_categories")}
	return output.Render(cmd.OutOrStdout(), data, summary, nil, opts)
}

// runServiceCategoryAction is the shared skeleton for deactivate/reactivate/
// move_up/move_down, which all take an <id> arg, call a PATCH endpoint, and
// return a resource (or array for move_up/move_down).
func runServiceCategoryAction(cmd *cobra.Command, args []string, method string, pathFn func(args []string) string, summary string,
	action func(ctx context.Context, client *wenmar.Client, id int) (any, error)) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id must be an integer")
	}

	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest(method, pathFn(args))

	data, err := action(context.Background(), client, id)
	if err != nil {
		return err
	}

	mode := resolveMode()
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs("service_categories", args[0])}
	return output.Render(cmd.OutOrStdout(), data, summary, nil, opts)
}
