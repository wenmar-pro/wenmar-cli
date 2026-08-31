package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// renderCount prints a bare integer: the number of items in a collection,
// 1 for a single object, or 0 for nil. Designed for monitoring scripts.
func renderCount(w io.Writer, data any) error {
	if data == nil {
		fmt.Fprintln(w, 0)
		return nil
	}

	switch v := data.(type) {
	case []any:
		fmt.Fprintln(w, len(v))
		return nil
	case map[string]any:
		fmt.Fprintln(w, 1)
		return nil
	default:
		// JSON round-trip to normalize
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		var normalized any
		if err := json.Unmarshal(b, &normalized); err != nil {
			return err
		}
		return renderCount(w, normalized)
	}
}
