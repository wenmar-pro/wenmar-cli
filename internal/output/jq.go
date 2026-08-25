package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/itchyny/gojq"
)

func renderJQ(w io.Writer, data any, filter string) error {
	// Marshal data to JSON, then run jq query
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data for jq: %w", err)
	}

	var input any
	if err := json.Unmarshal(b, &input); err != nil {
		return fmt.Errorf("failed to unmarshal for jq: %w", err)
	}

	q, err := gojq.Parse(filter)
	if err != nil {
		return fmt.Errorf("invalid jq filter: %w", err)
	}

	iter := q.Run(input)
	results := []any{}
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return fmt.Errorf("jq execution error: %w", err)
		}
		results = append(results, v)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if len(results) == 1 {
		return enc.Encode(results[0])
	}
	return enc.Encode(results)
}
