package auth

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/wenmar-pro/wenmar-cli/internal/config"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// ResolveAndSaveLocationID discovers the account's locations and resolves a
// default location for the CLI. The defaultLocationID is normally the API
// token's own location_id. If flag/env overrides are present they are passed
// in as defaultLocationID. If nonInteractive is true, the default is used
// silently. Otherwise the user is prompted to confirm or pick another.
// The chosen value is saved to config and returned.
func ResolveAndSaveLocationID(ctx context.Context, client *wenmar.Client, configPath, defaultLocationID string, nonInteractive bool, out io.Writer, in io.Reader) (string, error) {
	accountResp, err := client.ListAccount(ctx)
	if err != nil {
		return "", fmt.Errorf("could not fetch account locations: %w", err)
	}
	account := accountResp.GetJSON200()
	if account == nil {
		return "", fmt.Errorf("unexpected empty account response")
	}
	if len(account.Locations) == 0 {
		return "", fmt.Errorf("account has no locations")
	}

	locationsByID := make(map[string]string, len(account.Locations))
	for _, loc := range account.Locations {
		locationsByID[strconv.Itoa(loc.Id)] = loc.Name
	}

	selectedID := defaultLocationID
	if selectedID == "" {
		selectedID = strconv.Itoa(account.Locations[0].Id)
	}

	// Validate the selected/default ID against the account's locations.
	selectedName, ok := locationsByID[selectedID]
	if !ok {
		return "", fmt.Errorf("location %s is not valid for this account", selectedID)
	}

	if !nonInteractive {
		reader := bufio.NewReader(in)
		fmt.Fprintf(out, "\nDefault location: %s (id: %s)\n", selectedName, selectedID)
		fmt.Fprint(out, "Use this location? [Y/n]: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("could not read confirmation: %w", err)
		}
		if strings.TrimSpace(strings.ToLower(line)) == "n" {
			fmt.Fprintln(out, "Select a default location:")
			for i, loc := range account.Locations {
				idStr := strconv.Itoa(loc.Id)
				fmt.Fprintf(out, "  %d. %s (%s)\n", i+1, loc.Name, idStr)
			}
			fmt.Fprint(out, "Enter number: ")
			numLine, err := reader.ReadString('\n')
			if err != nil {
				return "", fmt.Errorf("could not read selection: %w", err)
			}
			num, err := strconv.Atoi(strings.TrimSpace(numLine))
			if err != nil || num < 1 || num > len(account.Locations) {
				return "", fmt.Errorf("invalid selection")
			}
			selectedLoc := account.Locations[num-1]
			selectedID = strconv.Itoa(selectedLoc.Id)
			selectedName = selectedLoc.Name
		}

		if selectedID != defaultLocationID && defaultLocationID != "" {
			fmt.Fprintf(out, "Warning: this token was created for %q. Requests to %q will be rejected unless you create a new API token for that location.\n",
				locationsByID[defaultLocationID], selectedName)
		}
	}

	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		cfg = &config.Config{}
	}
	cfg.LocationID = selectedID
	if err := config.SaveTo(configPath, cfg); err != nil {
		return "", fmt.Errorf("could not save location: %w", err)
	}
	return selectedID, nil
}
