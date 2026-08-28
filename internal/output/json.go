package output

import (
	"encoding/json"
	"io"
)

func renderJSONRaw(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func renderJSON(w io.Writer, data any, summary string, meta *Meta, breadcrumbs []Breadcrumb, notice string) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	envelope := map[string]any{
		"ok":   true,
		"data": data,
	}
	if summary != "" {
		envelope["summary"] = summary
	}
	if meta != nil {
		envelope["meta"] = meta
	}
	if len(breadcrumbs) > 0 {
		envelope["breadcrumbs"] = breadcrumbs
	}
	if notice != "" {
		envelope["notice"] = notice
	}
	return enc.Encode(envelope)
}
