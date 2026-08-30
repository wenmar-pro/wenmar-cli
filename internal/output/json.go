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

// Envelope is the JSON envelope for --json output.
type Envelope struct {
	OK          bool         `json:"ok"`
	Data        any          `json:"data,omitempty"`
	Summary     string       `json:"summary,omitempty"`
	Meta        *Meta        `json:"meta,omitempty"`
	Breadcrumbs []Breadcrumb `json:"breadcrumbs,omitempty"`
	Notice      string       `json:"notice,omitempty"`
}

func renderJSON(w io.Writer, data any, summary string, meta *Meta, breadcrumbs []Breadcrumb, notice string) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	envelope := Envelope{
		OK:          true,
		Data:        data,
		Summary:     summary,
		Meta:        meta,
		Breadcrumbs: breadcrumbs,
		Notice:      notice,
	}
	return enc.Encode(envelope)
}
