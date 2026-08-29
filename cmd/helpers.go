package cmd

import (
	"encoding/json"

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
// generic map/slice for the output renderer. The generated types have
// pointer fields and nested structs — we marshal to JSON and back to
// get a clean map[string]any or []map[string]any.
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