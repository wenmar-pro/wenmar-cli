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

## Output modes

- Default (no flag): GFM table — human-readable
- `--md` / `-m`: GFM table (explicit)
- `--json`: Full envelope `{ok, data, summary, meta}`
- `--agent`: Raw JSON data (no envelope) — for AI agents
- `--jq '.filter'`: jq-filtered JSON — implies `--json`

Always pass an explicit flag for scripts and agents. Auto-detection is
not used — explicit flags are predictable.

- `--quiet`: Raw JSON data (no envelope). Unlike `--agent`, it does not
  hijack `--help` to emit CommandInfo JSON. Use `--agent` in autonomous
  workflows that may need help; use `--quiet` in scripts that never do.

## Decision trees

### Finding resources

```
Need to find something?
├── Know the type + id? → wenmar <type> show <id>
├── List all? → wenmar <type> list
├── Sub-collection? → wenmar work_orders list --vehicle-id=<id>
├── Decode a VIN? → wenmar vehicles decode-vin <vin>
├── Check duplicates? → wenmar vehicles duplicates --vin=<vin>
└── Have a URL? → wenmar url parse "<url>"
```

### Modifying resources

```
Want to change something?
├── Have a URL? → wenmar url parse "<url>" → use extracted id
├── Create? → wenmar <type> create --field value
├── Update? → wenmar <type> update <id> --field value
└── Delete? → wenmar <type> delete <id>
```

## MUST follow these rules

1. **Choose the right output mode** — `--jq` to filter/extract, `--json`
   for full envelope, `--md` for human presentation. Never pipe to
   external `jq` — use `--jq` instead.
2. **Parse URLs first** with `wenmar url parse "<url>"` to extract the
   resource type and ID before calling `show`/`update`/`delete`.
3. **Sub-collection listing uses query-param filters** —
   `wenmar work_orders list --vehicle-id=13`, not nested paths.
4. **Check `wenmar commands`** for the full command catalog when
   unsure what's available.
5. **Always pass an output flag explicitly in scripts** — don't rely on
   TTY auto-detection.

## Common workflows

### List customers

```bash
wenmar customers list --md
wenmar customers list --json
wenmar customers list --jq '.[].full_name'
```

### Show a customer's details

```bash
wenmar customers show 42 --md
```

### Create a customer

```bash
wenmar customers create --full-name "Jane Doe" --email "jane@test.com" --json
```

### List work orders

```bash
wenmar work_orders list --md
wenmar work_orders list --page 2 --json
```

### Show a work order with nested customer/vehicle

```bash
wenmar work_orders show 100 --md
wenmar work_orders show 100 --jq '.vehicle.make'
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

## Full capability discovery

```bash
wenmar commands
```

Dumps every command, its flags, and descriptions as JSON. An agent
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
