# Wenmar CLI

A command-line interface for the Wenmar Pro automotive shop management API.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/Wenmar-Pro/wenmar-cli/master/install-cli | bash
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
