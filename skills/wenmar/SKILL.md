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
3. Set output mode explicitly (`--json`, `--jq`, `--agent`, `--md`)

## Agent Invariants

These rules MUST be followed without exception:

1. **Parse VINs first** with `wenmar vehicles decode-vin <vin>` — never
   assume a VIN format or try to decode it yourself.
2. **Work order and resource IDs are positional**, not flags:
   `wenmar work_orders show 12345`, not `--id 12345`.
3. **Destructive operations require `--dry-run` first** unless `--force`
   is passed. Run `wenmar vehicles delete 42 --dry-run` to preview
   before executing.
4. **`--base-url` selects the shop instance**; omitting it uses the
   config default (`~/.config/wenmar/config`).
5. **Bulk operations need explicit `--force`** — no implicit batch
   deletes or updates.
6. **Check `wenmar commands`** for the full command catalog when unsure
   what's available.
7. **Choose the right output mode:**
   - `--jq` to filter/extract specific fields
   - `--json` for full envelope `{ok, data, summary, meta}`
   - `--md` for human-readable GFM tables
   - `--ids-only` for shell loops (`| xargs`)
   - `--count` for bare integer counts (monitoring)
   - `--agent` for headless agent workflows (raw JSON, no envelope)
8. **Never pipe to external `jq`** — use `--jq` instead (built-in,
   no external dependency).
9. **Parse URLs first** with `wenmar url parse "<url>"` to extract the
   resource type and ID before calling `show`/`update`/`delete`.

## Decision trees

### Finding a work order

```
Need to find a work order?
├── Have the WO number? → wenmar work_orders show <number>
├── Have the VIN? → wenmar vehicles decode-vin <vin> → find vehicle → wenmar work_orders list --vehicle-id=<id>
├── My active jobs? → wenmar work_orders list --status active
├── Overdue jobs? → wenmar work_orders list --overdue
└── Have a URL? → wenmar url parse "<url>" → use extracted id
```

### Finding a vehicle

```
Need to find a vehicle?
├── Have the VIN? → wenmar vehicles decode-vin <vin>
├── Have a plate? → wenmar vehicles list --plate <plate> --state <state>
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

- Default (no flag): GFM table — human-readable
- `--md` / `-m` / `--markdown`: GFM table (explicit)
- `--json`: Full envelope `{ok, data, summary, meta, breadcrumbs}`
- `--agent`: Raw JSON data (no envelope) — for AI agents
- `--quiet`: Raw JSON data (no envelope). Unlike `--agent`, does not
  hijack `--help` to emit CommandInfo JSON.
- `--jq '.filter'`: jq-filtered JSON — implies `--json`
- `--ids-only`: One ID per line — for shell loops
- `--count`: Bare integer count — for monitoring

Always pass an explicit flag for scripts and agents. Auto-detection is
not used — explicit flags are predictable.

## Common workflows

### List customers

```bash
wenmar customers list --md
wenmar customers list --json
wenmar customers list --jq '.[].full_name'
wenmar customers list --ids-only | xargs -I{} wenmar customers show {}
wenmar customers list --count
```

### Show a customer's details

```bash
wenmar customers show 42 --md
wenmar customers show 42 --jq '.emails[]?.address'
```

### Create a customer

```bash
wenmar customers create --full-name "Jane Doe" --email "jane@test.com" --json
```

### Work orders

```bash
wenmar work_orders list --md
wenmar work_orders list --page 2 --json
wenmar work_orders show 100 --md
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

Scripts can branch on failure class without parsing stderr.

## Gotchas

- **Pagination is via the Link header**, not a body field. The SDK
  follows `rel="next"` automatically. Use `--page N` to jump to a page.
- **Work orders have nested customer/vehicle** — the `--md` table
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
