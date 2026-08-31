package cmd

import (
	"encoding/json"
	"strconv"

	"github.com/wenmar-pro/wenmar-cli/internal/errors"
)

// setRequest records the HTTP method and path for the current command so the
// error handler can show which request failed.
func setRequest(method, path string) {
	if currentDebugInfo == nil {
		currentDebugInfo = &errors.DebugInfo{}
	}
	currentDebugInfo.Method = method
	currentDebugInfo.Path = path
}

// extractData converts the generated response's JSON200 field to a
// generic map/slice for the output renderer.
func extractData(json200 any) any {
	if json200 == nil {
		return nil
	}
	b, err := json.Marshal(json200)
	if err != nil {
		return json200
	}
	var result any
	if json.Unmarshal(b, &result) != nil {
		return json200
	}
	if m, ok := result.(map[string]any); ok {
		return m
	}
	return result
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func boolPtr(b bool) *bool { return &b }

func intPtr(i int) *int { return &i }

// intSliceToStrPtr converts a []int flag var to a *[]string for SDK params
// that take tag/ID lists as strings. Returns nil when empty.
func intSliceToStrPtr(ids []int) *[]string {
	if len(ids) == 0 {
		return nil
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = strconv.Itoa(id)
	}
	return &out
}
