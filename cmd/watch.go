package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/auth"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	"github.com/wenmar-pro/wenmar-cli/internal/watch"
)

var (
	watchInterval    time.Duration
	watchExitOnFirst bool
	watchEvents      string
	watchRunSync     string
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for changes on a list endpoint (polling-based)",
	Long: `Poll a list endpoint on an interval and emit events when items
are added, changed, or removed. Events are emitted as JSON to stdout.

  wenmar watch --resource work_orders
  wenmar watch --resource work_orders --exit-on-first --json
  wenmar watch --resource work_orders --events new,changed --interval 5s

No Rails SSE endpoint required — this uses polling v1.`,
	RunE: runWatch,
}

func init() {
	watchCmd.Flags().DurationVar(&watchInterval, "interval", 5*time.Second, "Polling interval")
	watchCmd.Flags().BoolVar(&watchExitOnFirst, "exit-on-first", false, "Exit after the first poll (for scripts)")
	watchCmd.Flags().StringVar(&watchEvents, "events", "", "Comma-separated event types (new,changed,removed)")
	watchCmd.Flags().StringVar(&watchRunSync, "run-sync", "", "Script to run for each event (event JSON piped to stdin)")
	rootCmd.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, args []string) error {
	configPath := configPathFlag
	if configPath == "" {
		p, err := config.ConfigPath()
		if err == nil {
			configPath = p
		}
	}

	token, err := auth.ResolveTokenFrom(tokenFlag, configPath)
	if err != nil {
		return err
	}
	baseURL := auth.ResolveBaseURLFrom(baseURLFlag, configPath)

	// Build the URL to poll
	resource := "work_orders" // default; could be a flag in the future
	url := fmt.Sprintf("%s/%s", baseURL, resource)

	// Parse event filter
	var eventTypes map[string]bool
	if watchEvents != "" {
		eventTypes = map[string]bool{}
		for _, e := range splitComma(watchEvents) {
			eventTypes[e] = true
		}
	}

	poller := &watch.Poller{
		URL:         url,
		Token:       token,
		Interval:    watchInterval,
		ExitOnFirst: watchExitOnFirst,
		EventTypes:  eventTypes,
	}

	// Handle Ctrl-C
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")

	return poller.Run(ctx, func(e watch.Event) {
		if watchRunSync != "" {
			runSyncScript(watchRunSync, e)
			return
		}
		enc.Encode(e)
	})
}

func splitComma(s string) []string {
	var parts []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func runSyncScript(script string, e watch.Event) {
	// Pipe the event JSON to the script's stdin
	cmd := exec.Command(script)
	cmd.Stdin = strings.NewReader(fmt.Sprintf("%v", e))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
