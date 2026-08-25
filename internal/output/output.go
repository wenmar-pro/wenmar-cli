package output

import (
	"fmt"
	"io"
)

type Mode int

const (
	ModeDefault Mode = iota
	ModeMD
	ModeJSON
	ModeAgent
	ModeJQ
)

type Options struct {
	Mode     Mode
	JQFilter string
}

type Meta struct {
	Page    int  `json:"page,omitempty"`
	HasNext bool `json:"has_next"`
}

func ResolveMode(md, json, agent bool, jq string) Mode {
	if jq != "" {
		return ModeJQ
	}
	if agent {
		return ModeAgent
	}
	if json {
		return ModeJSON
	}
	if md {
		return ModeMD
	}
	return ModeDefault
}

func Render(w io.Writer, data any, summary string, meta *Meta, opts Options) error {
	switch opts.Mode {
	case ModeMD, ModeDefault:
		return renderMarkdown(w, data, summary)
	case ModeJSON:
		return renderJSON(w, data, summary, meta, false)
	case ModeAgent:
		return renderJSON(w, data, summary, meta, true)
	case ModeJQ:
		return renderJQ(w, data, opts.JQFilter)
	default:
		return fmt.Errorf("unknown output mode")
	}
}
