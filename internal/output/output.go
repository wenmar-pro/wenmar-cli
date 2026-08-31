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

// ModeSpec carries the output-mode flags from the command line. --output
// is the canonical selector; --json/--agent/--quiet/--jq are sugar.
type ModeSpec struct {
	Output string
	JSON   bool
	Agent  bool
	Quiet  bool
	JQ     string
}

// modeNames maps --output values to Modes. "styled" and "table" both render
// the default table; "styled" exists to override the pipe auto-switch.
var modeNames = map[string]Mode{
	"table":    ModeDefault,
	"styled":   ModeDefault,
	"md":       ModeMD,
	"json":     ModeJSON,
	"agent":    ModeAgent,
	"quiet":    ModeQuiet,
	"ids-only": ModeIDsOnly,
	"count":    ModeCount,
	"html":     ModeHTML,
}

// ParseMode resolves the output mode from ModeSpec. It errors on unknown
// mode names and on combinations that would silently pick a winner
// (--output plus any sugar, or two sugar flags together). With nothing set
// and a piped stdout it auto-switches to ModeQuiet so piped output is
// machine-readable.
func ParseMode(spec ModeSpec) (Mode, error) {
	if spec.Output != "" {
		if spec.JSON || spec.Agent || spec.Quiet || spec.JQ != "" {
			return 0, fmt.Errorf("--output cannot be combined with --json/--agent/--quiet/--jq — pick one")
		}
		m, ok := modeNames[spec.Output]
		if !ok {
			return 0, fmt.Errorf("unknown output mode %q (valid: table, md, json, agent, quiet, ids-only, count, html, styled)", spec.Output)
		}
		return m, nil
	}

	n := 0
	for _, set := range []bool{spec.JSON, spec.Agent, spec.Quiet, spec.JQ != ""} {
		if set {
			n++
		}
	}
	if n > 1 {
		return 0, fmt.Errorf("conflicting output flags — use --output <mode> to select one")
	}
	switch {
	case spec.JQ != "":
		return ModeJQ, nil
	case spec.Quiet:
		return ModeQuiet, nil
	case spec.Agent:
		return ModeAgent, nil
	case spec.JSON:
		return ModeJSON, nil
	}

	// Auto-switch: piped stdout gets machine-readable raw JSON.
	if !isTerminal(os.Stdout) {
		return ModeQuiet, nil
	}
	return ModeDefault, nil
}

func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd())
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
