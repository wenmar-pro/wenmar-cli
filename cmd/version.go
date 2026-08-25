package cmd

import "fmt"

// Build-time metadata, injected via ldflags:
//   -X github.com/wenmar-pro/wenmar-cli/cmd.version=<semver>
//   -X github.com/wenmar-pro/wenmar-cli/cmd.commit=<sha>
//   -X github.com/wenmar-pro/wenmar-cli/cmd.date=<RFC3339>
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func versionString() string {
	return fmt.Sprintf("wenmar v%s (commit %s, %s)", version, commit, date)
}
