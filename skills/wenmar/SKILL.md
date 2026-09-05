# Wenmar CLI

A command-line interface for the Wenmar Pro automotive shop management API.
Single static binary; all examples below are real, tested commands.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/Wenmar-Pro/wenmar-cli/main/install-cli | bash
```

Installs `wenmar` to `~/bin` or `~/.local/bin`; verifies SHA-256 checksum and
the cosign signature when available. Env overrides: `WENMAR_BIN_DIR`,
`WENMAR_VERSION`, `WENMAR_RELEASES_BASE`.

## Authentication

Pick ONE, in this precedence order:

1. `--token` flag (per command): `wenmar --token <tok> customers list`
2. `WENMAR_TOKEN` env var: `export WENMAR_TOKEN="<tok>"` (best for agents/CI)
3. Stored credentials: `wenmar auth login --token <tok>` — persists to the
   system keyring (file fallback `~/.config/wenmar/credentials.json`), then
   works without env vars: `wenmar auth status` verifies.
4. OAuth browser flow (humans): `wenmar auth login`

Get a token from the Wenmar Pro settings page, or via the Rails console:
`User.find(1).generate_api_token!`.

For agents: `WENMAR_TOKEN` or `auth login --token` are the two non-interactive
paths. Check state with `wenmar auth status`, and `wenmar auth token` to
print the token in scripts.

## Preflight checklist

1. `wenmar doctor` — auth, connectivity, config, completion, skill checks
2. `wenmar commands` — full command catalog as JSON (paths, flags, args,
   aliases, required flags). Pipe it (or add `--agent`) for raw JSON.
3. `wenmar <command> --help --agent` — structured JSON for ONE command
4. Read the help topics for the contracts:
   - `wenmar help output` — output modes and the JSON envelope format
   - `wenmar help exit-codes` — the stable 0-10 exit-code contract
   - `wenmar help auth` — token sources and auth methods
   - `wenmar help location` — location scoping
   - `wenmar help environment` — environment variables
   - `wenmar help watch` — the watch command
   - `wenmar help agent-help` — structured `--agent --help` for AI agents

## Agent invariants

These rules MUST be followed without exception:

1. **Parse VINs first** with `wenmar vehicles decode-vin <vin>` — never
   assume a VIN format.
2. **Resource IDs are positional**, never flags: `wenmar customers show 42`.
3. **Preview destructive ops** with `--dry-run` where offered
   (vehicles/drivers/workorders/servicecategories delete). There is no
   `--force` flag; a delete without `--dry-run` executes immediately.
4. **Set an explicit output mode** for anything parsed: `--agent` (raw JSON,
   no envelope). Default piped output is raw JSON, but explicit beats
   implicit.
5. **Never combine output flags**: exactly one of `--json`/`--agent`/
   `--jq`/`--ids-only`/`--count`/`--styled`. Mixing them errors.
6. **Never pipe to external `jq`** — use `--jq 'expr'` (built-in).
7. **Use `wenmar commands`** when unsure what exists — it lists canonical
   paths plus aliases (`canonical: false` entries).
8. **Parse pasted URLs first**: `wenmar url parse "<url>"` extracts
   `resource_type` and `id` before any show/update/delete.
9. **Canonical names**: `workorders` (aliases `work_orders`, `wo`),
   `servicecategories` (aliases `service-categories`, `sc`). Old spellings
   keep working; prefer canonical in new scripts.
10. **Branch on exit codes** (see the table below) — never parse stderr.

## Output modes

One flag, one mode — combining them is an error:

| Flag | What you get |
|------|--------------|
| (default) | Human table on a terminal; raw JSON when piped |
| `--json` | Full envelope `{ok, data, summary, meta, breadcrumbs}` |
| `--agent` | Raw JSON data, no envelope |
| `--jq 'expr'` | Filter output with a jq expression (implies raw JSON) |
| `--ids-only` | One ID per line (shell loops) |
| `--count` | Bare integer count (monitoring) |
| `--styled` | Force the human table even when piped |

`--agent` also makes `--help` emit structured JSON. Piped stdout with no
explicit mode → raw JSON. Conflicts error out.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Generic error |
| 2 | Auth failure / not logged in |
| 3 | Not found |
| 4 | Validation error |
| 5 | Rate limited |
| 6 | Server error |
| 7 | Conflict (e.g. duplicate VIN) |
| 8 | Forbidden |
| 9 | Truncated response — pass `--allow-partial` to accept |
| 10 | Network unreachable |

## Common workflows

### Customers

```bash
wenmar customers list
wenmar customers list --q "jane" --all          # full-text + all pages
wenmar customers list --ids-only
wenmar customers show 42
wenmar customers show 42 --jq '.emails[]?.address'
wenmar customers create --full-name "Jane Doe" --email "jane@test.com"
wenmar customers update 42 --company-name "New Corp"
wenmar customers lookup "jane doe"                  # name/email/phone search
wenmar customers duplicates --first-name Jane --last-name Doe --email j@x.com
wenmar customers vehicles 42
wenmar customers workorders 42
```

### Work orders

```bash
wenmar workorders list
wenmar workorders list --count
wenmar workorders show 100
wenmar workorders show 100 --jq '.vehicle.make'
wenmar workorders create --customer-id 42 --vehicle-id 5
wenmar workorders update 100 --intake-method drop_off
wenmar workorders delete 100 --dry-run
wenmar workorders estimate 100      # also: wip, inspection, parts, payments
```

### Vehicles

```bash
wenmar vehicles list
wenmar vehicles list --jq '.[].plate'
wenmar vehicles show 5
wenmar vehicles decode-vin 1HGCM82633A004352
wenmar vehicles lookup "honda civic"                # make/model/plate/vin
wenmar vehicles prefill --vin 1HGCM82633A004352
wenmar vehicles duplicates 1HGCM82633A004352       # VIN as positional arg
wenmar vehicles create --customer-id 42 --make Honda --model Civic --year 2020
wenmar vehicles transfer 5 --customer-id 43
wenmar vehicles trash 42
```

### Drivers, vendors, statements, locations, service categories, tags

```bash
wenmar drivers list --customer-id 42
wenmar drivers create --customer-id 42 --full-name "Jane Doe"
wenmar vendors list
wenmar vendors show 7
wenmar statements list --customer-id 42
wenmar statements show 9001
wenmar locations show main
wenmar account show
wenmar servicecategories list
wenmar servicecategories create --name "Oil change" --service-type maintenance
wenmar servicecategories seed-defaults
wenmar customertags list
wenmar customertags create --name "Fleet A"
wenmar workordertags list
wenmar workordertags create --name "Priority" --color "#ff0000"
```

### Location scoping

```bash
wenmar --location loc_abc workorders list     # or WENMAR_LOCATION_ID env,
                                             # or location_id in config
wenmar help location                         # full topic
```

### Pagination

Lists paginate via the Link header. Any list command that accepts filter
flags also supports `--all` (follow every page), `--page N`, and
`--per-page N`. When `meta.has_next` is true in `--json`, more pages
exist. Use `--all` to fetch the complete result set in one call:

```bash
wenmar customers list --all
wenmar customers list --q "jane" --all
```

### Truncated responses

If a response is truncated, wenmar exits 9. Re-run with `--allow-partial` to
accept the data plus a truncation notice in the envelope.

## Diagnostics

```bash
wenmar doctor                 # auth, connectivity, config checks
wenmar doctor --json          # structured {ok, checks:[...]}
wenmar config path
wenmar config list
wenmar watch --resource workorders --interval 5s   # poll + JSON events
```

## URL decomposition

```bash
wenmar url parse "https://app.wenmarpro.com/work_orders/42.json"
# {"resource_type": "work_orders", "id": "42", "format": "json", ...}
```

Recognized resources: customers, vehicles, workorders/work_orders, account,
locations, servicecategories (+ spellings), vendors, drivers, statements,
tags. Unknown paths return `{host, path, format}` only.

## Full capability discovery

```bash
wenmar commands
```

Every command with path, description, aliases, args, and flags (with
`required`). Alias entries carry `canonical: false` and `compatibility_for`.

## Gotchas

- **Pagination is Link-header based** — no page numbers in response bodies.
- **Nested objects truncate in tables** — use `--json` or `--jq`
  for full detail.
- **`customers update` cannot change emails/addresses/tags** — the update
  API only accepts phone changes (with `--remove-phone` by ID) and scalar
  fields. Nested emails/addresses/tags are set at CREATE time only.
- **The API is additive-only** — new fields may appear; existing meanings hold.
