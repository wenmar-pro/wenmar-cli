package output

import (
	"fmt"
	"html"
	"io"
	"sort"
)

// renderHTML renders data as a simple HTML document. Lists become tables,
// single entities become definition lists, and work orders get a richer
// layout with customer/vehicle info.
func renderHTML(w io.Writer, data any, title string) error {
	if title == "" {
		title = "Wenmar"
	}
	fmt.Fprintf(w, "<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n<title>%s</title>\n", html.EscapeString(title))
	fmt.Fprint(w, "<style>body{font-family:sans-serif;margin:2rem}table{border-collapse:collapse;width:100%}th,td{border:1px solid #ccc;padding:6px 10px;text-align:left}th{background:#f0f0f0}.badge{display:inline-block;padding:2px 8px;border-radius:10px;font-size:12px}.pending{background:#eee}.in_progress{background:#ffd}.completed{background:#dfd}</style>\n")
	fmt.Fprint(w, "</head>\n<body>\n")

	switch v := data.(type) {
	case []map[string]any:
		renderHTMLTable(w, v)
	case []any:
		renderHTMLTable(w, toMaps(v))
	case map[string]any:
		renderHTMLObject(w, v)
	default:
		fmt.Fprintf(w, "<p>%s</p>\n", html.EscapeString(fmt.Sprintf("%v", data)))
	}

	fmt.Fprint(w, "</body>\n</html>\n")
	return nil
}

func toMaps(items []any) []map[string]any {
	maps := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			maps = append(maps, m)
		}
	}
	return maps
}

func renderHTMLTable(w io.Writer, items []map[string]any) {
	if len(items) == 0 {
		fmt.Fprint(w, "<p>No results.</p>\n")
		return
	}

	// Collect the union of keys, preserving order of the first item.
	keys := make([]string, 0, len(items[0]))
	seen := map[string]bool{}
	for _, item := range items {
		for k := range item {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}

	fmt.Fprint(w, "<table>\n<thead>\n<tr>\n")
	for _, k := range keys {
		fmt.Fprintf(w, "<th>%s</th>\n", html.EscapeString(k))
	}
	fmt.Fprint(w, "</tr>\n</thead>\n<tbody>\n")
	for _, item := range items {
		fmt.Fprint(w, "<tr>\n")
		for _, k := range keys {
			fmt.Fprintf(w, "<td>%s</td>\n", html.EscapeString(fmt.Sprintf("%v", item[k])))
		}
		fmt.Fprint(w, "</tr>\n")
	}
	fmt.Fprint(w, "</tbody>\n</table>\n")
}

func renderHTMLObject(w io.Writer, obj map[string]any) {
	// Work order special-case: show customer/vehicle nicely.
	if wo, ok := obj["work_order_number"]; ok {
		renderHTMLWorkOrder(w, obj, wo)
		return
	}

	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Fprint(w, "<dl>\n")
	for _, k := range keys {
		fmt.Fprintf(w, "<dt><strong>%s</strong></dt>\n", html.EscapeString(k))
		fmt.Fprintf(w, "<dd>%s</dd>\n", html.EscapeString(fmt.Sprintf("%v", obj[k])))
	}
	fmt.Fprint(w, "</dl>\n")
}

func renderHTMLWorkOrder(w io.Writer, obj map[string]any, wo any) {
	fmt.Fprintf(w, "<h1>Work Order %s</h1>\n", html.EscapeString(fmt.Sprintf("%v", wo)))
	if status, ok := obj["status"].(string); ok {
		fmt.Fprintf(w, "<span class=\"badge %s\">%s</span>\n", html.EscapeString(status), html.EscapeString(status))
	}
	fmt.Fprint(w, "<h2>Customer</h2>\n")
	if c, ok := obj["customer"].(map[string]any); ok {
		fmt.Fprintf(w, "<p>%s</p>\n", html.EscapeString(fmt.Sprintf("%v", c["full_name"])))
	}
	fmt.Fprint(w, "<h2>Vehicle</h2>\n")
	if v, ok := obj["vehicle"].(map[string]any); ok {
		fmt.Fprintf(w, "<p>%s %s %s</p>\n",
			html.EscapeString(fmt.Sprintf("%v", v["year"])),
			html.EscapeString(fmt.Sprintf("%v", v["make"])),
			html.EscapeString(fmt.Sprintf("%v", v["model"])))
		if vin, ok := v["vin"].(string); ok && vin != "" {
			fmt.Fprintf(w, "<p>VIN: %s</p>\n", html.EscapeString(vin))
		}
	}
}
