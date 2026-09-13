package cmd

import (
	"fmt"
	"strings"

	"github.com/wenmar-pro/wenmar-cli/internal/auth"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	"github.com/wenmar-pro/wenmar-cli/internal/errors"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// newClient builds a Wenmar SDK client from the resolved token and base URL.
// It records the token source and base URL for error diagnostics.
func newClient() (*wenmar.Client, error) {
	configPath := configPathFlag
	if configPath == "" {
		p, err := config.ConfigPath()
		if err == nil {
			configPath = p
		}
	}
	manager, err := auth.ResolveAuthManager(tokenFlag, configPath)
	if err != nil {
		return nil, err
	}
	baseURL := auth.ResolveBaseURLFrom(baseURLFlag, configPath)

	// Resolve the token for diagnostics.
	rt, _ := auth.ResolveTokenWithSource(tokenFlag, configPath)
	currentDebugInfo = &errors.DebugInfo{
		TokenSource: string(rt.Source),
		TokenMasked: errors.MaskToken(rt.Token),
		BaseURL:     baseURL,
	}

	cfg := wenmar.DefaultConfig()
	cfg.BaseURL = baseURL
	return wenmar.NewClient(cfg, manager)
}

// newClientForLocation builds a client scoped to a location via the
// X-Wenmar-Location header. If locationID is empty, it returns the bare client.
func newClientForLocation(locationID string) (*wenmar.Client, error) {
	client, err := newClient()
	if err != nil {
		return nil, err
	}
	if locationID == "" {
		return client, nil
	}
	return client.ForLocation(locationID), nil
}

// newScopedClient resolves the location from flag/env/config and builds a
// client scoped to it.
func newScopedClient() (*wenmar.Client, error) {
	configPath := configPathFlag
	if configPath == "" {
		p, err := config.ConfigPath()
		if err == nil {
			configPath = p
		}
	}
	locationID := auth.ResolveLocationID(locationFlag, configPath)
	return newClientForLocation(locationID)
}

// formatMissingLocationError wraps an API error that indicates the
// X-Wenmar-Location header is required, adding an actionable hint about how
// to configure a default location. Other errors pass through unchanged.
func formatMissingLocationError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "X-Wenmar-Location header is required for this token") {
		return fmt.Errorf("%w\n\nNo default location configured for this API token\nRun `wenmar location use` or set WENMAR_LOCATION_ID", err)
	}
	return err
}
