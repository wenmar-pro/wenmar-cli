package output

import (
	"fmt"
	"io"
	"os"
)

type Mode int

const (
	ModeDefault Mode = iota
	ModeMD
	ModeJSON
	ModeAgent
	ModeJQ
	ModeQuiet
	ModeIDsOnly
	ModeCount
)

type Breadcrumb struct {
	Cmd string `json:"cmd"`
}

type Options struct {
	Mode        Mode
	JQFilter    string
	Breadcrumbs []Breadcrumb
}

type Meta struct {
	Page    int  `json:"page,omitempty"`
	HasNext bool `json:"has_next"`
}

func ResolveMode(md, json, agent, quiet, idsOnly, count bool, jq string) Mode {
	if jq != "" {
		return ModeJQ
	}
	if count {
		return ModeCount
	}
	if idsOnly {
		return ModeIDsOnly
	}
	if agent {
		return ModeAgent
	}
	if quiet {
		return ModeQuiet
	}
	if json {
		return ModeJSON
	}
	if md {
		return ModeMD
	}
	return ModeDefault
}

// CaptureBreadcrumbs derives the leaf invocation from os.Args. Only emitted in
// JSON envelope mode.
func CaptureBreadcrumbs() []Breadcrumb {
	if len(os.Args) < 2 {
		return nil
	}
	// Join from the program name; skip flags is out of scope — bc3 emits the
	// command string as invoked.
	return []Breadcrumb{{Cmd: joinArgs(os.Args)}}
}

func joinArgs(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}
	return out
}

func Render(w io.Writer, data any, summary string, meta *Meta, opts Options) error {
	switch opts.Mode {
	case ModeMD, ModeDefault:
		return renderMarkdown(w, data, summary)
	case ModeJSON:
		return renderJSON(w, data, summary, meta, opts.Breadcrumbs)
	case ModeAgent, ModeQuiet:
		return renderJSONRaw(w, data)
	case ModeJQ:
		return renderJQ(w, data, opts.JQFilter)
	case ModeIDsOnly:
		return renderIDsOnly(w, data)
	case ModeCount:
		return renderCount(w, data)
	default:
		return fmt.Errorf("unknown output mode")
	}
}
