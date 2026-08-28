package cmd

import (
	"net/url"
	"path"
	"strings"

	"github.com/wenmar-pro/wenmar-cli/internal/output"
	"github.com/spf13/cobra"
)

// ParseResult is the decomposed form of a Wenmar URL.
type ParseResult struct {
	Host         string            `json:"host,omitempty"`
	ResourceType string            `json:"resource_type,omitempty"`
	ID           string            `json:"id,omitempty"`
	Format       string            `json:"format,omitempty"`
	Path         string            `json:"path,omitempty"`
	QueryParams  map[string]string `json:"query_params,omitempty"`
}

var urlCmd = &cobra.Command{
	Use:   "url",
	Short: "URL utilities",
}

var urlParseCmd = &cobra.Command{
	Use:   "parse <url>",
	Short: "Parse a Wenmar URL into its components",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		result := parseWenmarURL(args[0])
		mode := output.ResolveModeStyled(mdFlag, jsonFlag, agentFlag, quietFlag, idsOnlyFlag, countFlag, jqFlag, htmlFlag, styledFlag)
		opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("url")}
		return output.Render(cmd.OutOrStdout(), result, "", nil, opts)
	},
}

func init() {
	urlCmd.AddCommand(urlParseCmd)
	rootCmd.AddCommand(urlCmd)
}

// knownResources are the public API flat routes the parser recognizes.
var knownResources = map[string]bool{
	"customers":   true,
	"vehicles":    true,
	"work_orders": true,
	"account":     true,
	"locations":   true,
}

// parseWenmarURL decomposes a Wenmar URL into resource type, id, and format.
// Known resource types match the public API flat routes:
//
//	/customers/7.json    → customers, id 7, format json
//	/vehicles.json?customer_id=7 → vehicles, format json, query customer_id
//	/work_orders/42.json → work_orders, id 42, format json
//
// Unknown paths return {host, path, format} with no resource type.
func parseWenmarURL(raw string) ParseResult {
	u, err := url.Parse(raw)
	if err != nil {
		return ParseResult{Path: raw}
	}

	result := ParseResult{
		Host: u.Host,
		Path: u.Path,
	}

	// Extract format (e.g. .json) and the path with extension stripped.
	seg := u.Path
	format := ""
	if i := strings.LastIndex(seg, "."); i >= 0 {
		format = seg[i+1:]
		seg = seg[:i]
		result.Format = format
	}

	cleaned := path.Clean(seg)
	parts := strings.Split(strings.Trim(cleaned, "/"), "/")

	// Strip known leading prefixes (e.g. /api) so the resource segment lines up.
	for len(parts) > 0 && (parts[0] == "api" || parts[0] == "v1") {
		parts = parts[1:]
	}

	if len(parts) == 0 {
		return result
	}

	// Only assign resource_type when the leading segment is a known resource.
	if !knownResources[parts[0]] {
		return result
	}

	// A show URL has exactly two segments: <resource>/<id>.
	if len(parts) == 2 {
		result.ResourceType = parts[0]
		result.ID = parts[1]
		return result
	}

	// A collection/index URL has one segment, possibly with query params.
	if len(parts) == 1 {
		result.ResourceType = parts[0]
		if u.RawQuery != "" {
			q := u.Query()
			params := make(map[string]string, len(q))
			for k, v := range q {
				if len(v) > 0 {
					params[k] = v[0]
				}
			}
			result.QueryParams = params
		}
		return result
	}

	// Unknown — leave resource_type empty.
	return result
}
