package errors

import (
	"errors"
	"fmt"
	"io"
	"strings"

	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// DebugInfo carries request context that helps diagnose a failed command.
type DebugInfo struct {
	TokenSource string
	TokenMasked string
	BaseURL     string
	Method      string
	Path        string
}

// PrintError writes a human-readable error to w, augmented with debug context
// (token source, base URL, request method/path, and per-field validation
// details) so users can quickly tell what went wrong.
func PrintError(w io.Writer, err error, info *DebugInfo) {
	fmt.Fprintf(w, "ERROR: %s\n", err)

	var apiErr *wenmar.APIError
	if errors.As(err, &apiErr) {
		printDebugBlock(w, info, apiErr)
		printHints(w, apiErr)
		return
	}

	// Non-API error (network, config, etc.) — still show what we know.
	if info != nil {
		fmt.Fprintln(w)
		printDebugLine(w, "token", info.TokenMasked, info.TokenSource)
		printDebugLine(w, "base URL", info.BaseURL, "")
		if info.Method != "" && info.Path != "" {
			printDebugLine(w, "request", info.Method+" "+info.Path, "")
		}
	}
}

func printDebugBlock(w io.Writer, info *DebugInfo, apiErr *wenmar.APIError) {
	fmt.Fprintln(w)
	if info != nil {
		printDebugLine(w, "token", info.TokenMasked, info.TokenSource)
		printDebugLine(w, "base URL", info.BaseURL, "")
	}
	if apiErr.Method != "" && apiErr.Path != "" {
		printDebugLine(w, "request", apiErr.Method+" "+apiErr.Path, "")
	}
	if apiErr.StatusCode != 0 {
		printDebugLine(w, "status", fmt.Sprintf("%d", apiErr.StatusCode), "")
	}
	if len(apiErr.Details) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  field errors:")
		for field, msgs := range apiErr.Details {
			fmt.Fprintf(w, "    %s: %v\n", field, msgs)
		}
	}
}

func printDebugLine(w io.Writer, label, value, note string) {
	if value == "" {
		value = "(unknown)"
	}
	if note != "" {
		fmt.Fprintf(w, "  %-9s %s  (%s)\n", label+":", value, note)
		return
	}
	fmt.Fprintf(w, "  %-9s %s\n", label+":", value)
}

func printHints(w io.Writer, apiErr *wenmar.APIError) {
	fmt.Fprintln(w)
	switch apiErr.Code {
	case "unauthorized":
		fmt.Fprintln(w, "  Hint: the token may be invalid, expired, or missing. Run `wenmar setup` to reconfigure.")
	case "validation_failed":
		fmt.Fprintln(w, "  Hint: fix the field errors above and retry.")
	case "not_found":
		fmt.Fprintln(w, "  Hint: the resource does not exist, or you lack access to it.")
	case "rate_limited":
		fmt.Fprintln(w, "  Hint: you are being rate limited. Wait and retry.")
	default:
		if apiErr.StatusCode >= 500 {
			fmt.Fprintln(w, "  Hint: the server reported an error. Check the server logs or retry later.")
		}
	}
}

// MaskToken returns a masked form of a token for safe display.
func MaskToken(token string) string {
	if len(token) <= 8 {
		return strings.Repeat("*", len(token))
	}
	return token[:4] + "..." + token[len(token)-4:]
}
