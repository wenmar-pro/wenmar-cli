package cmd

import (
	"context"
	"encoding/json"
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
	watchRunAsync    string
	watchLocation    string
	watchResource    string
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for changes on a list endpoint (polling-based)",
	Long: `Poll a list endpoint on an interval and emit events when items
are added, changed, or removed. Events are emitted as JSON to stdout.

  wenmar watch --resource work_orders
  wenmar watch --resource work_orders --exit-on-first --json
  wenmar watch --resource work_orders --events new,changed --interval 5s
  wenmar watch --resource customers --location loc_abc

No Rails SSE endpoint required — this uses polling v1.`,
	RunE: runWatch,
}

func init() {
	watchCmd.Flags().DurationVar(&watchInterval, "interval", 5*time.Second, "Polling interval")
	watchCmd.Flags().BoolVar(&watchExitOnFirst, "exit-on-first", false, "Exit after the first poll (for scripts)")
	watchCmd.Flags().StringVar(&watchEvents, "events", "", "Comma-separated event types (new,changed,removed)")
	watchCmd.Flags().StringVar(&watchRunSync, "run-sync", "", "Script to run for each event (event JSON piped to stdin)")
	watchCmd.Flags().StringVar(&watchRunAsync, "run-async", "", "Script to run for each event without blocking the poll loop")
	watchCmd.Flags().StringVar(&watchLocation, "location", "", "Location ID to scope the polled endpoint")
	watchCmd.Flags().StringVar(&watchResource, "resource", "work_orders", "Resource to watch (work_orders, customers, vehicles)")
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

	client, err := newClient()
	if err != nil {
		return err
	}

	locationID := watchLocation
	if locationID == "" {
		locationID = auth.ResolveLocationID(locationFlag, configPath)
	}

	// Parse event filter
	var eventTypes map[string]bool
	if watchEvents != "" {
		eventTypes = map[string]bool{}
		for _, e := range splitComma(watchEvents) {
			eventTypes[e] = true
		}
	}

	poller := &watch.Poller{
		Client:      client,
		Resource:    watchResource,
		LocationID:  locationID,
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
		if watchRunAsync != "" {
			go runSyncScript(watchRunAsync, e)
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
	data, _ := json.Marshal(e)
	cmd := exec.Command(script)
	cmd.Stdin = strings.NewReader(string(data))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
