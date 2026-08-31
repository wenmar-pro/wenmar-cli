# Wenmar CLI

A command-line interface for the Wenmar Pro automotive shop management API.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/Wenmar-Pro/wenmar-cli/main/install-cli | bash
```

Installs `wenmar` to `~/bin` or `~/.local/bin` and verifies the SHA-256
checksum against the release. See `install-cli` in the repo root for
`WENMAR_BIN_DIR` / `WENMAR_VERSION` / `WENMAR_RELEASES_BASE` options.

## Auth

Set your API token as an environment variable:

```bash
export WENMAR_TOKEN="your-api-token"
```

Or pass it on each command:

```bash
wenmar --token "your-api-token" customers list
```

To get a token, generate one from the Wenmar Pro settings page or via
the Rails console: `User.find(1).generate_api_token!`.

## Preflight checklist

Before starting an agent workflow:

1. `wenmar doctor` — Verify auth, connectivity, and config
2. `wenmar commands` — Discover the full command surface
3. Set output mode explicitly (`--output <mode>`, or a quick flag like
   `--json`/`--jq`/`--agent`)

## Agent Invariants

These rules MUST be followed without exception:

1. **Parse VINs first** with `wenmar vehicles decode-vin <vin>` — never
   assume a VIN format or try to decode it yourself.
2. **Work order and resource IDs are positional**, not flags:
   `wenmar work_orders show 12345`, not `--id 12345`.
3. **Destructive operations require `--dry-run` first.** Run
   `wenmar vehicles delete 42 --dry-run` to preview before executing.
4. **`--base-url` selects the shop instance**; omitting it uses the
   config default (`~/.config/wenmar/config`).
5. **Check `wenmar commands`** for the full command catalog when unsure
   what's available.
6. **Choose the right output mode:**
   - `--jq` to filter/extract specific fields
   - `--json` for full envelope `{ok, data, summary, meta}`
   - `--output md` for human-readable GFM tables
   - `--output ids-only` for shell loops (`| xargs`)
   - `--output count` for bare integer counts (monitoring)
   - `--agent` for headless agent workflows (raw JSON, no envelope)
   - Never combine output flags: use `--output <mode>` alone, or one quick
     flag (`--json`/`--agent`/`--quiet`/`--jq`). Combining them is an error.
7. **Never pipe to external `jq`** — use `--jq` instead (built-in,
   no external dependency).
8. **Parse URLs first** with `wenmar url parse "<url>"` to extract the
   resource type and ID before calling `show`/`update`/`delete`.

## Decision trees

### Finding a work order

```
Need to find a work order?
├── Have the WO number? → wenmar work_orders show <number>
├── Have the VIN? → wenmar vehicles decode-vin <vin> → find vehicle → wenmar work_orders list --jq '.[] | select(.vehicle.id == <id>)'
├── My active jobs? → wenmar work_orders list --jq '.[] | select(.status == "in_progress")'
├── Overdue jobs? → wenmar work_orders list --jq '.[] | select(.status == "overdue")'
└── Have a URL? → wenmar url parse "<url>" → use extracted id
```

### Finding a vehicle

```
Need to find a vehicle?
├── Have the VIN? → wenmar vehicles decode-vin <vin>
├── Have a plate? → wenmar vehicles lookup "<plate>"
├── Know the customer? → wenmar customers show <id> → follow vehicles_url
└── Have a URL? → wenmar url parse "<url>" → use extracted id
```

### Finding a customer

```
Need to find a customer?
├── Have the ID? → wenmar customers show <id>
├── Know the name? → wenmar customers list --jq '.[] | select(.full_name | test("Jane"; "i"))'
├── Have the email? → wenmar customers list --jq '.[] | select(.emails[]?.address == "jane@example.com")'
└── Have a URL? → wenmar url parse "<url>" → use extracted id
```

### Modifying resources

```
Want to change something?
├── Have a URL? → wenmar url parse "<url>" → use extracted id
├── Create? → wenmar <type> create --field value --json
├── Update? → wenmar <type> update <id> --field value --json
└── Delete?
    ├── Preview first: wenmar <type> delete <id> --dry-run --json
    └── Execute: wenmar <type> delete <id> --json
```

## Output modes

The canonical flag is `--output <mode>`:

- `--output table`: Human-readable table (default on a terminal)
- `--output md`: GFM table
- `--output json`: Full envelope `{ok, data, summary, meta, breadcrumbs}`
- `--output agent`: Raw JSON data (no envelope) — for AI agents
- `--output quiet`: Raw JSON data (no envelope). Unlike `--agent`, does not
  hijack `--help` to emit CommandInfo JSON.
- `--output ids-only`: One ID per line — for shell loops
- `--output count`: Bare integer count — for monitoring
- `--output styled`: Force human tables even when piped

Quick flags (equivalents, hidden from subcommand help): `--json`, `--agent`,
`--quiet`, `--jq '.filter'` (implies json). Combining `--output` with a quick
flag (or two quick flags together) is an error — pick one.

Always pass an explicit mode for scripts and agents. When stdout is not a
TTY (e.g. piped) and no mode flag is set, wenmar emits raw JSON so the
output is machine-readable.

## Common workflows

### List customers

```bash
wenmar customers list --output md
wenmar customers list --json
wenmar customers list --jq '.[].full_name'
wenmar customers list --output ids-only | xargs -I{} wenmar customers show {}
wenmar customers list --output count
```

### Show a customer's details

```bash
wenmar customers show 42 --output md
wenmar customers show 42 --jq '.emails[]?.address'
```

### Create a customer

```bash
wenmar customers create --full-name "Jane Doe" --email "jane@test.com" --json
```

### Work orders

```bash
wenmar work_orders list --output md
wenmar work_orders list --json
wenmar work_orders show 100 --output md
wenmar work_orders show 100 --jq '.vehicle.make'
wenmar work_orders create --customer-id 1 --vehicle-id 1 --json
```

### Delete with dry-run

```bash
wenmar vehicles delete 42 --dry-run --json   # Preview
wenmar vehicles delete 42 --json             # Execute
```

## Diagnostics

```bash
wenmar doctor          # Auth, connectivity, config, completion check
wenmar doctor --json   # Structured for agents
wenmar config path     # Show config file location
wenmar config list     # Show all config values
```

## Shell completion

```bash
wenmar completion bash > ~/.local/share/bash-completion/completions/wenmar
wenmar completion zsh > "${fpath[1]}/_wenmar"
wenmar completion fish > ~/.config/fish/completions/wenmar.fish
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Generic error |
| 2 | Auth failure |
| 3 | Not found |
| 4 | Validation error |
| 5 | Rate limited |
| 6 | Server error |
| 7 | Conflict (e.g. duplicate VIN) |
| 8 | Forbidden (403) |
| 9 | Truncated response without `--allow-partial` |
| 10 | Network unreachable |

When the server sends an unrecognized error `code`, the exit code falls
back to the HTTP status class (401→2, 404→3, 422→4, 429→5, 403→8,
409→7, 5xx→6), so the contract holds even for new error codes.

Scripts can branch on failure class without parsing stderr.

## Gotchas

- **Pagination is via the Link header**, not a body field. The SDK
  follows `rel="next"` automatically. `customers list` supports `--page N`;
  `work_orders list` does not (it follows the Link header only).
- **Work orders have nested customer/vehicle** — the `--output md` table
  truncates nested objects. Use `--json` or `--jq` for full detail.
- **The API is additive-only** — no versioned URLs. New fields may
  appear but existing fields keep their meaning.
- **Destructive ops without `--dry-run` execute immediately** — always
  preview first in agent workflows.

## Full capability discovery

```bash
wenmar commands
```

Dumps every command, its flags, args, and descriptions as JSON. An agent
can learn the entire tool surface from this one command.

## Decomposing URLs

```bash
wenmar url parse "https://app.wenmarpro.com/work_orders/42.json"
# { "host": "app.wenmarpro.com", "resource_type": "work_orders", "id": "42", "format": "json" }
```

Given a user-pasted URL, `url parse` extracts `resource_type`, `id`,
`format`, and `query_params` so an agent can immediately call the
matching `show`/`update`/`delete`. Unknown paths return `{host, path,
format}` with no resource type.
