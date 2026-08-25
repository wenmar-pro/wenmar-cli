package auth

import (
	"fmt"
	"os"
)

const defaultBaseURL = "https://app.wenmarpro.com"

func ResolveToken(flagToken, configToken string) (string, error) {
	if flagToken != "" {
		return flagToken, nil
	}

	if envToken := os.Getenv("WENMAR_TOKEN"); envToken != "" {
		return envToken, nil
	}

	if configToken != "" {
		return configToken, nil
	}

	return "", fmt.Errorf("API token required. Set --token flag, WENMAR_TOKEN env var, or ~/.wenmar/config")
}

func ResolveBaseURL(flagURL string) string {
	if flagURL != "" {
		return flagURL
	}

	if envURL := os.Getenv("WENMAR_BASE_URL"); envURL != "" {
		return envURL
	}

	return defaultBaseURL
}
