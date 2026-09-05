package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/auth"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	"github.com/wenmar-pro/wenmar-cli/internal/errors"
	"github.com/wenmar-pro/wenmar-cli/internal/exports"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
)

var (
	exportFormat     string
	exportFilters    []string
	exportOutput     string
	exportInline     bool
	exportForceAsync bool
	exportList       bool
	exportMaxWait    time.Duration
)

var exportCmd = &cobra.Command{
	Use:   "export [resource]",
	Short: "Export shop data to CSV or JSON",
	Long: `Create and download a resource export from the Wenmar Pro API.

Discovery:
  wenmar export --list                          Show all exportable resources and accepted filters.

Create exports:
  wenmar export customers -o customers.csv
  wenmar export customers --filter q=Acme --filter status=active -o customers.csv
  wenmar export inspections --format json -o inspections.json
  wenmar export customers --inline -o -            Write inline base64 data directly to stdout.
  wenmar export customers --force-async -o wo.csv  Always enqueue a background job and poll.

Output defaults to stdout when -o is omitted or set to "-".`,
	Args: func(cmd *cobra.Command, args []string) error {
		if exportList {
			if len(args) > 0 {
				return fmt.Errorf("resource not allowed with --list")
			}
			return nil
		}
		if len(args) != 1 {
			return fmt.Errorf("requires exactly one resource name; use --list to see available resources")
		}
		return nil
	},
	RunE:    runExport,
	GroupID: "resources",
}

func runExport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	configPath := configPathFlag
	if configPath == "" {
		if p, err := config.ConfigPath(); err == nil {
			configPath = p
		}
	}

	rt, err := auth.ResolveTokenWithSource(tokenFlag, configPath)
	if err != nil {
		return err
	}
	baseURL := auth.ResolveBaseURLFrom(baseURLFlag, configPath)
	locationID := auth.ResolveLocationID(locationFlag, configPath)

	currentDebugInfo = &errors.DebugInfo{
		TokenSource: string(rt.Source),
		TokenMasked: errors.MaskToken(rt.Token),
		BaseURL:     baseURL,
	}

	client := exports.NewClient(baseURL, rt.Token, locationID, nil)

	if exportList {
		setRequest("GET", "/exports/schema.json")
		schema, err := client.Schema(ctx)
		if err != nil {
			return err
		}
		mode, err := resolveMode()
		if err != nil {
			return err
		}
		rows := make([]map[string]any, len(schema.Resources))
		for i, r := range schema.Resources {
			rows[i] = map[string]any{
				"name":    r.Name,
				"formats": r.Formats,
				"filters": len(r.Filters),
			}
		}
		opts := output.Options{Mode: mode, JQFilter: jqFlag}
		return output.Render(cmd.OutOrStdout(), rows, fmt.Sprintf("%d exportable resources", len(schema.Resources)), nil, opts)
	}

	resource := args[0]
	format := exportFormat
	if format == "" {
		format = "csv"
	}
	filters, err := parseExportFilters(exportFilters)
	if err != nil {
		return err
	}

	setRequest("POST", "/exports.json")
	created, err := client.Create(ctx, exports.CreateRequest{
		Resource:   resource,
		Format:     format,
		Filters:    filters,
		Inline:     exportInline,
		ForceAsync: exportForceAsync,
	})
	if err != nil {
		return err
	}

	out := exportOutput
	if out == "" {
		out = "-"
	}

	var data []byte
	var filename string
	var contentType string

	if exportInline {
		data, err = exports.DownloadInline(created)
		if err != nil {
			return err
		}
		filename = exportFilename(resource, format)
		contentType = "application/octet-stream"
	} else {
		setRequest("GET", created.DownloadURL)
		data, filename, contentType, err = client.Download(ctx, created.DownloadURL, exportMaxWait)
		if err != nil {
			return err
		}
		if filename == "" {
			filename = exportFilename(resource, format)
		}
	}

	if out == "-" {
		if _, err := cmd.OutOrStdout().Write(data); err != nil {
			return err
		}
	} else {
		if err := os.WriteFile(out, data, 0644); err != nil {
			return fmt.Errorf("write export file: %w", err)
		}
	}

	mode, err := resolveMode()
	if err != nil {
		return err
	}
	result := map[string]any{
		"resource":      resource,
		"format":        format,
		"export_log_id": created.ExportLogID,
		"row_count":     created.RowCount,
		"status":        created.Status,
		"filename":      filename,
		"content_type":  contentType,
		"destination":   out,
		"bytes":         len(data),
	}
	summary := fmt.Sprintf("Exported %d %s rows", created.RowCount, resource)
	opts := output.Options{Mode: mode, JQFilter: jqFlag}
	return output.Render(cmd.OutOrStdout(), result, summary, nil, opts)
}

func parseExportFilters(raw []string) (map[string]any, error) {
	filters := make(map[string]any, len(raw))
	for _, pair := range raw {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --filter %q: expected key=value", pair)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("invalid --filter %q: key is empty", pair)
		}
		filters[key] = value
	}
	return filters, nil
}

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", "", "Export format: csv or json (default csv)")
	exportCmd.Flags().StringArrayVar(&exportFilters, "filter", nil, "Filter as key=value (repeatable)")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file path (default stdout, use - for stdout)")
	exportCmd.Flags().BoolVar(&exportInline, "inline", false, "Request inline base64 data and skip polling")
	exportCmd.Flags().BoolVar(&exportForceAsync, "force-async", false, "Always enqueue an async export and poll for it")
	exportCmd.Flags().BoolVar(&exportList, "list", false, "List exportable resources and filters")
	exportCmd.Flags().DurationVar(&exportMaxWait, "max-wait", 5*time.Minute, "Maximum time to poll for an async export")
	rootCmd.AddCommand(exportCmd)
}

// exportFilename picks a sensible default filename when the server does not
// provide one via Content-Disposition.
func exportFilename(resource, format string) string {
	ext := ".csv"
	if format == "json" {
		ext = ".json"
	}
	return fmt.Sprintf("%s_export%s", resource, ext)
}
