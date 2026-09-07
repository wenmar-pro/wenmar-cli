package cmd

import (
	"fmt"
	"strings"
	"testing"
)

func TestNewScopedClient_MissingLocationErrorIsActionable(t *testing.T) {
	// We only test the helper here; the full newScopedClient path requires a
	// token and server. The helper should wrap the raw API message.
	err := fmt.Errorf("API error: GET /account -> bad_request: X-Wenmar-Location header is required for this token (HTTP 400)")
	msg := formatMissingLocationError(err)
	if !strings.Contains(msg.Error(), "wenmar location use") {
		t.Errorf("expected actionable hint, got %q", msg)
	}
}
