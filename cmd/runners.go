package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// modeSpec snapshots the output-mode flags for ParseMode.
func modeSpec() output.ModeSpec {
	return output.ModeSpec{
		Output: outputFlag,
		JSON:   jsonFlag,
		Agent:  agentFlag,
		Quiet:  quietFlag,
		JQ:     jqFlag,
	}
}

// resolveMode resolves the output mode. ParseMode validated the flags in
// PersistentPreRunE; the error path here is defensive for direct handler
// calls (tests) and still propagates.
func resolveMode() (output.Mode, error) {
	return output.ParseMode(modeSpec())
}

// parseInt converts args[0] to an int with a consistent error message.
func parseInt(s string) (int, error) {
	id, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("id must be an integer, got %q", s)
	}
	return id, nil
}

// idPath returns a path function that builds prefix + args[0].
func idPath(prefix string) func(args []string) string {
	return func(args []string) string { return prefix + args[0] }
}

// runShow is the shared skeleton for all "show <id>" commands where the ID
// is an integer parsed from args[0].
//
// pathFn receives args and returns the full request path for diagnostics.
//
// Usage:
//
//	RunE: func(cmd *cobra.Command, args []string) error {
//	    return runShow(cmd, args, "vendors", "GET", func(args []string) string { return "/vendors/" + args[0] }, func(ctx context.Context, client *wenmar.Client, id int) (any, error) {
//	        resp, err := client.ShowVendor(ctx, id)
//	        if err != nil { return nil, err }
//	        return resp.JSON200, nil
//	    })
//	},
func runShow(cmd *cobra.Command, args []string, resource, method string, pathFn func(args []string) string,
	getter func(ctx context.Context, client *wenmar.Client, id int) (any, error)) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest(method, pathFn(args))

	id, err := parseInt(args[0])
	if err != nil {
		return err
	}

	respData, err := getter(context.Background(), client, id)
	if err != nil {
		return err
	}

	data := extractData(respData)
	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs(resource, args[0])}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

// runShowStr is like runShow but for resources whose ID is a string (e.g. locations).
func runShowStr(cmd *cobra.Command, args []string, resource, method string, pathFn func(args []string) string,
	getter func(ctx context.Context, client *wenmar.Client, id string) (any, error)) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest(method, pathFn(args))

	respData, err := getter(context.Background(), client, args[0])
	if err != nil {
		return err
	}

	data := extractData(respData)
	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs(resource, args[0])}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

// runList is the shared skeleton for simple "list" commands (no pagination meta).
func runList(cmd *cobra.Command, resource, path string,
	lister func(ctx context.Context, client *wenmar.Client) (any, error)) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", path)

	respData, err := lister(context.Background(), client)
	if err != nil {
		return err
	}

	data := extractData(respData)
	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs(resource)}
	return output.Render(cmd.OutOrStdout(), data, "", nil, opts)
}

// runListPaginated is the shared skeleton for list commands that report
// pagination metadata via a Paginator (Link header).
func runListPaginated(cmd *cobra.Command, resource, path string,
	lister func(ctx context.Context, client *wenmar.Client) (any, *wenmar.Paginator, error)) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", path)

	respData, paginator, err := lister(context.Background(), client)
	if err != nil {
		return err
	}

	data := extractData(respData)
	summary := fmt.Sprintf("Page 1. More results: %v", paginator.HasNext())
	meta := &output.Meta{HasNext: paginator.HasNext()}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	if mode == output.ModeIDsOnly || mode == output.ModeCount {
		output.PrintPaginationNotice(meta, 1)
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs(resource)}
	return output.Render(cmd.OutOrStdout(), data, summary, meta, opts)
}

// runListPaginatedWithAll is the shared skeleton for list commands that
// support --all auto-pagination. When allFlag is true and more pages exist,
// it follows the Paginator's next links until exhausted and merges every
// page's items into one result set. Otherwise it behaves like
// runListPaginated (page 1 + hasNext hint).
func runListPaginatedWithAll(cmd *cobra.Command, resource, path string, allFlag bool,
	lister func(ctx context.Context, client *wenmar.Client) (any, *wenmar.Paginator, error)) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("GET", path)

	respData, paginator, err := lister(context.Background(), client)
	if err != nil {
		return err
	}

	data := extractData(respData)
	summary := fmt.Sprintf("Page 1. More results: %v", paginator.HasNext())
	meta := &output.Meta{HasNext: paginator.HasNext()}
	pages := 1

	if allFlag && paginator.HasNext() {
		items, ok := data.([]any)
		if ok {
			for paginator.HasNext() {
				next, err := paginator.NextPage(context.Background())
				if err != nil {
					return err
				}
				pages++
				if nextPageItems, ok := next.([]any); ok {
					items = append(items, nextPageItems...)
				}
			}
			data = items
		}
		summary = fmt.Sprintf("Fetched all %d pages. More results: %v", pages, paginator.HasNext())
		meta = &output.Meta{HasNext: false}
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	if mode == output.ModeIDsOnly || mode == output.ModeCount {
		output.PrintPaginationNotice(meta, pages)
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs(resource)}
	return output.Render(cmd.OutOrStdout(), data, summary, meta, opts)
}

// runCreate is the shared skeleton for "create" commands.
func runCreate(cmd *cobra.Command, resource, path, summary string,
	bodyBuilder func() (any, error),
	sender func(ctx context.Context, client *wenmar.Client, body any) (any, error)) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("POST", path)

	body, err := bodyBuilder()
	if err != nil {
		return err
	}

	respData, err := sender(context.Background(), client, body)
	if err != nil {
		return err
	}

	data := extractData(respData)
	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: createBreadcrumbs(resource, "0")}
	return output.Render(cmd.OutOrStdout(), data, summary, nil, opts)
}

// runUpdate is the shared skeleton for "update <id>" commands where the ID
// is an integer.
func runUpdate(cmd *cobra.Command, args []string, resource string, pathFn func(args []string) string, summary string,
	bodyBuilder func(id int) (any, error),
	sender func(ctx context.Context, client *wenmar.Client, id int, body any) (any, error)) error {
	return runAction(cmd, args, resource, "PATCH", pathFn, summary, bodyBuilder, sender)
}

// runAction is the shared skeleton for id-scoped mutation commands (PATCH/POST
// to /resource/{id}[/sub]). It parses the int id, builds the body, calls the
// sender, and renders the response with show breadcrumbs.
func runAction(cmd *cobra.Command, args []string, resource, method string, pathFn func(args []string) string, summary string,
	bodyBuilder func(id int) (any, error),
	sender func(ctx context.Context, client *wenmar.Client, id int, body any) (any, error)) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest(method, pathFn(args))

	id, err := parseInt(args[0])
	if err != nil {
		return err
	}

	body, err := bodyBuilder(id)
	if err != nil {
		return err
	}

	respData, err := sender(context.Background(), client, id, body)
	if err != nil {
		return err
	}

	data := extractData(respData)
	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs(resource, args[0])}
	return output.Render(cmd.OutOrStdout(), data, summary, nil, opts)
}

// runActionNoBody is the shared skeleton for id-scoped action commands whose
// request body has no scalar fields (service category deactivate/reactivate/
// move-up/move-down). It parses the id, calls the SDK, renders the response.
func runActionNoBody(cmd *cobra.Command, args []string, resource, method string, pathFn func(args []string) string, summary string,
	action func(ctx context.Context, client *wenmar.Client, id int) (any, error)) error {
	id, err := parseInt(args[0])
	if err != nil {
		return err
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

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs(resource, args[0])}
	return output.Render(cmd.OutOrStdout(), extractData(data), summary, nil, opts)
}

// runSeedAction is the skeleton for POST-collection actions with empty
// bodies (e.g. service categories seed-defaults): no id, call, render.
func runSeedAction(cmd *cobra.Command, resource, path string, summary string,
	action func(ctx context.Context, client *wenmar.Client) (any, error)) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("POST", path)

	data, err := action(context.Background(), client)
	if err != nil {
		return err
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs(resource)}
	return output.Render(cmd.OutOrStdout(), extractData(data), summary, nil, opts)
}

// runDelete is the shared skeleton for "delete <id>" commands, including
// the dry-run block. resourceLabel is the display name (e.g. "Driver",
// "Work order"); resourceSlug is the slug for breadcrumbs (e.g. "drivers",
// "work_orders").
func runDelete(cmd *cobra.Command, args []string, resourceLabel, resourceSlug string, pathFn func(args []string) string,
	dryRun bool,
	deleter func(ctx context.Context, client *wenmar.Client, id int) (any, error)) error {
	id, err := parseInt(args[0])
	if err != nil {
		return err
	}

	if dryRun {
		mode, err := resolveMode()
		if err != nil {
			return err
		}
		opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs(resourceSlug, args[0])}
		dryRunData := map[string]any{
			"dry_run":      true,
			"would_delete": fmt.Sprintf("%s:%d", resourceSlug, id),
		}
		return output.Render(cmd.OutOrStdout(), dryRunData, fmt.Sprintf("Would delete %s %d (dry run).", resourceLabel, id), nil, opts)
	}

	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("DELETE", pathFn(args))

	_, err = deleter(context.Background(), client, id)
	if err != nil {
		return err
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs(resourceSlug)}
	return output.Render(cmd.OutOrStdout(), nil, fmt.Sprintf("%s %d deleted.", resourceLabel, id), nil, opts)
}
