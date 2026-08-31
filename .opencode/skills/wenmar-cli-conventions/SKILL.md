---
name: wenmar-cli-conventions
description: Conventions for the wenmar-cli codebase — how commands are generated, how to add a command, and the exclusion taxonomy. Use when editing commands, the generator, or gen_overrides.yaml.
---

# Wenmar CLI Conventions

## Architecture

The resource commands are **generated**, not hand-written:

```
spec/openapi.enriched.yaml + cmd/gen_overrides.yaml
        → cmd/gencli (jennifer code emission)
        → committed cmd/gen_*.go (one file per resource)
        → shared runners in cmd/runners.go
```

Four companion files hold commands whose flag/parse logic is not
per-operation derivable:

- `cmd/tags.go` — all tag commands (type-branching customer vs vehicle)
- `cmd/customers_extras.go` — customer create/update (splitName, label|value
  email/phone/address parsing, `mergeInto` JSON round-trip)
- `cmd/vehicles_extras.go` — vehicle create/update (pointer-typed wrapper
  bodies with `omitempty`)
- `cmd/work_orders_extras.go` — work-order show + 5 tabs (truncation check,
  tab-fetch switch)

## Golden rule

**Never hand-edit `cmd/gen_*.go`.** Change `cmd/gen_overrides.yaml` or the
generator, then:

```bash
make generate && make golden-update
```

## Adding a command

1. Add a `commands:` entry in `cmd/gen_overrides.yaml` (resource, command,
   method, request_struct, query_param_struct, id_param, aliases, …).
2. `make generate` — emits the command into the resource's `gen_*.go`.
3. `make golden-update` — refresh fixtures.
4. `make surface-snapshot` — refresh the surface contract.

## Exclusion taxonomy

Ops in `exclude:` fall into three buckets:

- **companion** — non-derivable per-op logic (tags type-branching, customer
  email/phone parsing, work-order tab truncation, vehicle pointer bodies).
- **no CLI surface** — work-order lifecycle/auth/history ops the CLI has
  never exposed (e.g. `create_work_order_authorization`, whose
  `ServiceDecisions` has numeric-key fields unrepresentable as flags).
- **not exposed** — `delete_customer`, `list_team`, vehicle history.

## Naming

- kebab-case flags (`--full-name`), snake_case override keys
  (`source_customer_id`).
- Squashed-compound resources with aliases (D2): `workorders` (aliases
  `work_orders`, `wo`), `servicecategories` (aliases `service-categories`,
  `sc`).
- Nested collections are positional show commands via `id_param:`
  (`customers vehicles <id>`).

## Drift gates

- Golden test (`cmd/gencli/golden_test.go`) — emission drift.
- Surface snapshot (`surface-snapshot.json`) — command-surface drift.
- CI runs both; `make golden-update` / `make surface-snapshot` refresh.
