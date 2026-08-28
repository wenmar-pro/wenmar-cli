package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// renderIDsOnly extracts the "id" field from each item in a collection
// and prints one ID per line. For a single object, prints its ID.
// Designed for shell loops: `wenmar customers list --ids-only | xargs ...`
func renderIDsOnly(w io.Writer, data any) error {
	if data == nil {
		return nil
	}

	switch v := data.(type) {
	case []any:
		for _, item := range v {
			id, ok := extractID(item)
			if !ok {
				continue
			}
			fmt.Fprintln(w, id)
		}
		return nil
	case map[string]any:
		id, ok := extractID(v)
		if !ok {
			return nil
		}
		fmt.Fprintln(w, id)
		return nil
	default:
		// Fallback: JSON round-trip to normalize
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		var normalized any
		if err := json.Unmarshal(b, &normalized); err != nil {
			return err
		}
		return renderIDsOnly(w, normalized)
	}
}

func extractID(item any) (string, bool) {
	m, ok := item.(map[string]any)
	if !ok {
		return "", false
	}
	id, exists := m["id"]
	if !exists {
		return "", false
	}
	switch idVal := id.(type) {
	case float64:
		return fmt.Sprintf("%v", int64(idVal)), true
	case string:
		return idVal, true
	case int:
		return fmt.Sprintf("%d", idVal), true
	default:
		return fmt.Sprintf("%v", idVal), true
	}
}
