package output

import (
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-isatty"
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
	ModeHTML
)

// Breadcrumb is an actionable navigation hint emitted in the JSON envelope.
type Breadcrumb struct {
	Action string `json:"action"` // "show", "create", "update", "delete"
	Cmd    string `json:"cmd"`    // "wenmar work_orders show <id>"
}

type Options struct {
	Mode        Mode
	JQFilter    string
	Breadcrumbs []Breadcrumb
	Notice      string
}

type Meta struct {
	Page    int  `json:"page,omitempty"`
	HasNext bool `json:"has_next"`
}

// ResolveMode resolves the output mode from the explicit flags. If no
// explicit mode is set and stdout is not a TTY, it auto-switches to
// ModeQuiet (raw JSON) so piped output is machine-readable. --styled forces
// ModeDefault (human tables) even when piped.
func ResolveMode(md, json, agent, quiet, idsOnly, count bool, jq string) Mode {
	return ResolveModeStyled(md, json, agent, quiet, idsOnly, count, jq, false, false)
}

// ResolveModeStyled is ResolveMode with --html and --styled support.
func ResolveModeStyled(md, json, agent, quiet, idsOnly, count bool, jq string, html, styled bool) Mode {
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
	if html {
		return ModeHTML
	}
	if md {
		return ModeMD
	}
	if styled {
		return ModeDefault
	}
	// Auto-switch: when stdout is not a TTY and no explicit mode is set,
	// emit raw JSON so piped output is machine-readable.
	if !isTerminal(os.Stdout) {
		return ModeQuiet
	}
	return ModeDefault
}

func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd())
}

// CaptureBreadcrumbs derives the leaf invocation from os.Args. Only emitted in
// JSON envelope mode.
func CaptureBreadcrumbs() []Breadcrumb {
	if len(os.Args) < 2 {
		return nil
	}
	return []Breadcrumb{{Action: "show", Cmd: joinArgs(os.Args)}}
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
		return renderJSON(w, data, summary, meta, opts.Breadcrumbs, opts.Notice)
	case ModeAgent, ModeQuiet:
		return renderJSONRaw(w, data)
	case ModeJQ:
		return renderJQ(w, data, opts.JQFilter)
	case ModeIDsOnly:
		return renderIDsOnly(w, data)
	case ModeCount:
		return renderCount(w, data)
	case ModeHTML:
		return renderHTML(w, data, summary)
	default:
		return fmt.Errorf("unknown output mode")
	}
}

// PrintPaginationNotice writes a pagination hint to stderr when there are
// more pages and the mode is IDs-only or count, so stdout stays pipeable.
func PrintPaginationNotice(meta *Meta, page int) {
	if meta == nil || !meta.HasNext {
		return
	}
	fmt.Fprintf(os.Stderr, "Page %d of more. Use --page %d for next.\n", page, page+1)
}
