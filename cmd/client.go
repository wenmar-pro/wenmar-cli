package cmd

import (
	"context"

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
func newClientForLocation(ctx context.Context, locationID string) (*wenmar.Client, error) {
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
func newScopedClient(ctx context.Context) (*wenmar.Client, error) {
	configPath := configPathFlag
	if configPath == "" {
		p, err := config.ConfigPath()
		if err == nil {
			configPath = p
		}
	}
	locationID := auth.ResolveLocationID(locationFlag, configPath)
	return newClientForLocation(ctx, locationID)
}
