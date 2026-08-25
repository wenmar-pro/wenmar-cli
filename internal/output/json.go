package output

import (
	"encoding/json"
	"io"
)

func renderJSON(w io.Writer, data any, summary string, meta *Meta, raw bool) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if raw {
		return enc.Encode(data)
	}

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
	return enc.Encode(envelope)
}
