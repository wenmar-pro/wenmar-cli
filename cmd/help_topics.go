package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// helpTopic is a named help document.
type helpTopic struct {
	name    string
	title   string
	content string
}

var helpTopics = []helpTopic{
	{
		name:  "output",
		title: "Output Formats",
		content: `Wenmar CLI supports several output modes:

  --json       Full JSON envelope: {ok, data, summary, meta, breadcrumbs}
  --agent      Raw JSON data (no envelope)
  --quiet      Raw JSON output, no envelope, no agent discovery
  --md / -m    GitHub-flavored markdown table
  --html       HTML document
  --jq <expr>  Apply a jq filter (implies --json)
  --ids-only   Print one ID per line (for shell loops)
  --count      Print the count of results (bare integer)
  --styled     Force human tables even when piped

Auto-switch: when stdout is not a TTY (e.g. piped to another command) and no
explicit output mode is set, wenmar emits raw JSON so the output is
machine-readable. Use --styled to force human tables in a pipe.

Envelope structure:
  {"ok": true, "data": [...], "summary": "5 customers",
   "meta": {"has_next": true}, "breadcrumbs": [{"action": "show", "cmd": "..."}]}`,
	},
	{
		name:  "exit-codes",
		title: "Exit Codes",
		content: `Wenmar CLI uses stable process exit codes:

  0  success
  1  unclassified error
  2  unauthorized / token expired
  3  resource not found
  4  validation failed
  5  rate limited
  6  server error (5xx)
  7  conflict (e.g. duplicate VIN)
  8  forbidden (403)
  9  truncated response without --allow-partial
  10 network unreachable`,
	},
	{
		name:  "environment",
		title: "Environment Variables",
		content: `Wenmar CLI reads these environment variables:

  WENMAR_TOKEN         API bearer token
  WENMAR_URL           API base URL (default: https://app.wenmarpro.com)
  WENMAR_LOCATION_ID   Location ID to scope requests`,
	},
	{
		name:  "auth",
		title: "Authentication",
		content: `Wenmar CLI supports two auth methods:

  static   A fixed API token (default). Stored in the system keyring with a
           file fallback at ~/.config/wenmar/credentials.json.
  oauth    Not yet implemented. The auth interfaces are designed so OAuth
           can be added without breaking callers.

Token sources, in precedence order:
  1. --token flag
  2. WENMAR_TOKEN env var
  3. keyring (via credential store)
  4. config file token (legacy)

Commands:
  wenmar auth login     Store your API token
  wenmar auth status    Show auth status and test the connection
  wenmar auth token     Print the bearer token (for scripts)
  wenmar auth refresh   Refresh the token (OAuth only)
  wenmar auth logout    Clear stored credentials`,
	},
	{
		name:  "location",
		title: "Location Scoping",
		content: `Wenmar CLI can scope all requests to a specific location.

  --location <id>       Location ID (flag)
  WENMAR_LOCATION_ID    Location ID (env var)
  location_id           Location ID (config file)

When set, the CLI sends the X-Wenmar-Location header on every request. The
SDK's ForLocation method verifies access to the location before returning a
scoped client.`,
	},
	{
		name:  "watch",
		title: "Watch",
		content: `The watch command polls a list endpoint and emits events when items
are added, changed, or removed.

  wenmar watch --resource work_orders
  wenmar watch --resource customers --location loc_abc
  wenmar watch --resource work_orders --events new,changed --interval 5s
  wenmar watch --resource work_orders --run-sync ./script.sh

Flags:
  --resource <name>   Resource to watch (work_orders, customers, vehicles)
  --location <id>     Scope to a location
  --interval <dur>    Polling interval (default 5s)
  --events <types>    Comma-separated event types (new,changed,removed)
  --exit-on-first     Exit after the first poll
  --run-sync <script> Run a script for each event (event JSON on stdin)
  --run-async <script> Run a script without blocking the poll loop`,
	},
}

var helpCmd = &cobra.Command{
	Use:   "help [command|topic]",
	Short: "Help about any command or topic",
	Args:  cobra.ArbitraryArgs,
	RunE:  runHelp,
}

func init() {
	// Replace cobra's built-in help command so `wenmar help <topic>` works.
	rootCmd.SetHelpCommand(helpCmd)
}

func runHelp(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// List available topics.
		if agentFlag {
			return emitHelpTopicsJSON(cmd)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Available help topics:")
		for _, t := range helpTopics {
			fmt.Fprintf(cmd.OutOrStdout(), "  %-14s %s\n", t.name, t.title)
		}
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "Run `wenmar help <topic>` for details.")
		return nil
	}

	for _, t := range helpTopics {
		if t.name == args[0] {
			if agentFlag {
				return emitHelpTopicJSON(cmd, t)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n%s\n", t.title, t.content)
			return nil
		}
	}

	// Fall back to cobra's built-in help for commands. Resolve the target
	// command from the remaining args (topics were checked above).
	target, _, err := cmd.Root().Find(args)
	if err != nil || target == cmd.Root() {
		return fmt.Errorf("unknown help topic or command %q — run `wenmar help` for topics or `wenmar --help` for commands", args[0])
	}
	return target.Help()
}

func emitHelpTopicsJSON(cmd *cobra.Command) error {
	topics := make([]map[string]string, 0, len(helpTopics))
	for _, t := range helpTopics {
		topics = append(topics, map[string]string{"name": t.name, "title": t.title})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(topics)
}

func emitHelpTopicJSON(cmd *cobra.Command, t helpTopic) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]string{
		"name":    t.name,
		"title":   t.title,
		"content": t.content,
	})
}
