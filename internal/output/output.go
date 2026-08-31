package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
)

type Mode int

const (
	ModeDefault Mode = iota // human table (also forced by --styled when piped)
	ModeJSON                // --json: full envelope {ok, data, summary, meta, breadcrumbs}
	ModeAgent               // --agent: raw JSON, no envelope (also the piped default)
	ModeJQ                  // --jq <expr>
	ModeIDsOnly             // --ids-only: one ID per line
	ModeCount               // --count: bare integer count
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

// ModeSpec carries the output-mode flags from the command line. All six
// are peers — there is no umbrella flag. More than one set is an error.
type ModeSpec struct {
	JSON    bool
	Agent   bool
	JQ      string
	IDsOnly bool
	Styled  bool
	Count   bool
}

// ParseMode resolves the output mode from ModeSpec. Exactly one mode flag
// may be set — combining them errors with the offending flag names rather
// than silently picking a winner. With nothing set: table on a terminal,
// raw JSON when piped.
func ParseMode(spec ModeSpec) (Mode, error) {
	active := []string{}
	if spec.JSON {
		active = append(active, "--json")
	}
	if spec.Agent {
		active = append(active, "--agent")
	}
	if spec.JQ != "" {
		active = append(active, "--jq")
	}
	if spec.IDsOnly {
		active = append(active, "--ids-only")
	}
	if spec.Styled {
		active = append(active, "--styled")
	}
	if spec.Count {
		active = append(active, "--count")
	}

	if len(active) > 1 {
		return 0, fmt.Errorf("conflicting output flags: %s (use only one)", strings.Join(active, ", "))
	}

	switch {
	case spec.JQ != "":
		return ModeJQ, nil
	case spec.IDsOnly:
		return ModeIDsOnly, nil
	case spec.Styled:
		return ModeDefault, nil // forces the human table even when piped
	case spec.Agent:
		return ModeAgent, nil
	case spec.JSON:
		return ModeJSON, nil
	case spec.Count:
		return ModeCount, nil
	}

	// Auto-switch: piped stdout gets machine-readable raw JSON.
	if !isTerminal(os.Stdout) {
		return ModeAgent, nil
	}
	return ModeDefault, nil
}

func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd())
}

func Render(w io.Writer, data any, summary string, meta *Meta, opts Options) error {
	switch opts.Mode {
	case ModeDefault:
		return renderMarkdown(w, data, summary)
	case ModeJSON:
		return renderJSON(w, data, summary, meta, opts.Breadcrumbs, opts.Notice)
	case ModeAgent:
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

// PrintPaginationNotice writes a pagination hint to stderr when there are
// more pages and the mode is IDs-only or count, so stdout stays pipeable.
func PrintPaginationNotice(meta *Meta, page int) {
	if meta == nil || !meta.HasNext {
		return
	}
	fmt.Fprintf(os.Stderr, "Page %d of more. Use --page %d for next.\n", page, page+1)
}
