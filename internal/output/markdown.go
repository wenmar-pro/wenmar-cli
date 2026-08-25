package output

import (
	"fmt"
	"io"
	"strings"
)

func renderMarkdown(w io.Writer, data any, summary string) error {
	items := toSlice(data)
	if len(items) == 0 {
		fmt.Fprintln(w, "No results.")
		if summary != "" {
			fmt.Fprintln(w, summary)
		}
		return nil
	}

	headers := extractHeaders(items[0])

	// Header row
	fmt.Fprintf(w, "| %s |\n", strings.Join(headers, " | "))
	// Separator
	seps := make([]string, len(headers))
	for i := range seps {
		seps[i] = "---"
	}
	fmt.Fprintf(w, "| %s |\n", strings.Join(seps, " | "))
	// Data rows
	for _, item := range items {
		values := make([]string, len(headers))
		for i, h := range headers {
			values[i] = fmt.Sprintf("%v", item[h])
		}
		fmt.Fprintf(w, "| %s |\n", strings.Join(values, " | "))
	}

	if summary != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, summary)
	}
	return nil
}

func toSlice(data any) []map[string]any {
	switch v := data.(type) {
	case []map[string]any:
		return v
	case map[string]any:
		if d, ok := v["data"]; ok {
			if arr, ok := d.([]map[string]any); ok {
				return arr
			}
		}
		return []map[string]any{v}
	default:
		return nil
	}
}

func extractHeaders(item map[string]any) []string {
	// Deterministic order: id first, then alphabetical
	headers := []string{}
	if _, ok := item["id"]; ok {
		headers = append(headers, "id")
	}
	for k := range item {
		if k != "id" {
			headers = append(headers, k)
		}
	}
	// Sort non-id headers
	if len(headers) > 1 {
		sorted := headers[1:]
		for i := 0; i < len(sorted); i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[i] > sorted[j] {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
	}
	return headers
}
