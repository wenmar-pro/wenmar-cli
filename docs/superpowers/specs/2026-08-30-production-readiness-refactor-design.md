# Wenmar CLI Production-Readiness Refactor — Design

**Date:** 2026-08-30
**Status:** Approved in brainstorming; pending user spec review
**Scope:** Prerelease hardening of wenmar-cli — correctness fixes, codegen adoption, interface and help overhaul, agent-skill rewrite, CI/release gates.

---

## Problem statement

A four-pass review (cmd/, internal/, UX/agent-surface, tests/CI/release) found the CLI is functionally close to production quality but has:

1. **Three live bugs** — main doesn't build against the published SDK (`ListCustomersParams` missing from `go/v0.3.0`), `wenmar help <command>` prints root help instead of the command's help, and the gencli generator stack-overflows on any operation without an explicit `method:` override (`cmd/gencli/gen.go:1098`).
2. **Two broken safety gates** — CI's private-module auth step is dead code (`ci.yml:23` uses `env.GITHUB_TOKEN` where it needs `secrets.GITHUB_TOKEN`), and the surface-snapshot `Required` detection is a range-copy bug that makes the required-flag CI guard vacuous.
3. **A dual implementation** — ten hand-written resource files (~1,785 lines) mirror what the generator emits; generated output is gitignored, the generator is untested (1,362 lines, zero tests), its generated build ships broken action commands, and `.gitignore:5` silently ignores new files in `cmd/gencli/`.
4. **Silent wrongness** — flags accepted then dropped (`customers update --remove-email/--remove-address`, `work_orders list --page`, `customers duplicates --phone`); typo'd subcommands exit 0; TUI refresh loop stops after one tick and drops fetch results; `watch` dies on one transient error; config migration has a data-loss window; exit codes 401/404/422 collapse to generic 1.
5. **UX debt** — 23 commands in a flat help list, 16 global flags repeated at every help level, zero examples, three resource-naming conventions, two agent-discovery surfaces that disagree with each other.
6. **A skill file with fabricated claims** — SKILL.md documents flags and behaviors that don't exist (`--status`, `--overdue`, `--plate`, `--state`, delete `--force`, bulk ops) and truncates the exit-code table at 6 of 11 entries.
7. **Enforcement gap** — every drift-detection tool (`make check-published`, `make surface-diff`, cosign+checksum chain) exists but almost none run in CI; ~45 files fail gofmt; release runs zero tests.

With the API heading to 250+ operations, hand-written commands don't scale. This design makes the generator the source of truth and hardens everything around it — now, while prerelease aliasing makes renames free.

## Goals

- `main` is green, releasable, and CI-enforced against the **published** SDK tag.
- No user input is ever silently accepted and ignored; no typo'd command exits 0.
- One implementation of the command surface: generated, committed, tested, drift-gated.
- Help is scannable: grouped commands, examples everywhere, output flags collapsed.
- Both agent surfaces (`wenmar commands`, `--help --agent`) share one schema and one builder.
- SKILL.md is verified-accurate and stays accurate by CI.
- Scripts can rely on the documented exit-code contract (0–10) in all paths.
- Tests never touch the developer's real keyring, config, or credentials.

## Non-goals

- No new API-facing features (no `--status`/`--overdue`/`--plate` flags — SKILL.md is fixed to match reality instead).
- No TUI redesign; only its broken loop, routing, and data bugs are fixed.
- No MCP server, no SDK changes beyond tagging/pinning what the CLI already consumes.
- No public deprecation windows: this is prerelease; breaking changes ship behind permanent aliases.

## Locked decisions

| # | Decision |
|---|----------|
| D1 | **Generator is the source of truth.** Generated files are committed; hand-written resource twins are deleted; the `generated` build tag goes away; the generator gets golden tests and a CI regen-drift gate. Hand-written files remain only for non-derivable commands (setup, auth, config, doctor, tui, watch, url, skill, upgrade, completion, help topics). |
| D2 | **Naming follows basecamp-cli conventions** (see "CLI naming rules"): squashed compound resource nouns, kebab-case flags, space-separated verb/noun subcommands, no snake_case CLI tokens. Renames ship with permanent backward-compatible aliases. |
| D3 | **`--output <mode>` is canonical.** `--json`, `--agent`, `--quiet`, `--jq` survive as thin sugar; `--md`/`-m`/`--markdown`, `--ids-only`, `--count`, `--html`, `--styled` drop as standalone flags and become `--output` modes. `--allow-partial` stays (behavior flag, not a mode). |
| D4 | **SKILL.md is rewritten to verified reality** and guarded by a CI freshness test that parses every example command in the skill and asserts it exists in `wenmar commands` output. |
| D5 | **Exit-code contract 0–10 is closed and enforced** with status-code fallbacks so scripts always see documented codes. |

---

## Phase 0 — Unblock main

Everything else gates on a green main. Do this first, alone, as one PR.

1. **SDK repin.** Tag wenmar-sdk `go/v0.4.0` (contains `ListCustomersParams`/`ListCustomersWithParams`), bump `go.mod`, verify `GOWORK=off go build ./... && GOWORK=off go test ./...` passes.
2. **CI auth fix.** `ci.yml:23` — `if: secrets.GITHUB_TOKEN != ''` (correct context), so the `git config` rewrite for the private SDK actually runs.
3. **Formatting.** `gofmt -w` all ~45 dirty files; add `gofmt -l` + `go vet ./...` gates to CI in the same PR.

**Exit criteria:** CI green on main with `GOWORK=off`; vet and fmt gates active.

## Phase 1 — Correctness fixes

Behavior-preserving bug fixes, each independently shippable. Order within the phase is by blast radius.

### 1.1 Generator crashers and dead config
- `sdkMethodNameFor` recursion (`gen.go:1094-1099`): fall through to `sdkMethodName(cmd.OperationID)`.
- Wire `flag_overrides` from `gen_overrides.yaml` into `parseBodyFields` (currently parsed in `main.go:141-145` and silently ignored), or delete the config block. **Wire it** — Phase 2 depends on it.
- Delete contradictory YAML entries (operations both `exclude`d and in `commands:`: `delete_customer`, `update_tags`, `create_customer_tag`, `create_vehicle_tag`, `create_customer`) — exclusion wins today; the dead entries mislead.
- Delete dead generator helpers (`pathPrefixForRunner`, `pathPrefixString`).
- Fix `.gitignore:5`: `gencli` → `/gencli` so new files under `cmd/gencli/` aren't silently ignored.

### 1.2 Silently-ignored user input
- `customers update`: wire `--remove-email` / `--remove-address` (IntSlice) into the PATCH body, mirroring the existing phones handling, or remove the flags. **Wire them** — removal is a real capability.
- `work_orders list`: bind `--page` and pass it to `ListWorkOrdersWithPagination`.
- `customers duplicates`: pass `Phone` into `CheckCustomerDuplicateParams`.
- `drivers.go` `--customer-id` bound five times, `tui --remote` never read: keep one binding pattern; delete `--remote`.

### 1.3 Unknown subcommands must fail
- All parent/resource commands get `Args` validation (cobra `OnlyValidArgs`/`NoArgs` + `ValidArgs` of child names) so `wenmar customers delete 42` exits non-zero with "unknown command" + "did you mean" suggestions — never an exit-0 help dump.
- Enable cobra `SuggestionsMinimumDistance` and ensure the custom error printer surfaces cobra's suggestion text.

### 1.4 `wenmar help` correctness
- `runHelp` fallback (`help_topics.go:163-164`): resolve `args[0]` via `cmd.Root().Find(args)`; print that command's help; only fall back to root help/topics listing when not found.
- Bare `wenmar help` lists topics **and** points to `wenmar --help` for the command list.
- Fix stale `auth` help topic: OAuth browser+PKCE **is implemented**; align topic text with current command Shorts.

### 1.5 Surface snapshot and introspection truth
- Fix range-copy bug (`surface_snapshot.go:71-75`): iterate `for i := range surf.Flags { ... }`; regenerate `surface-snapshot.json` (it will gain `required: true` entries — expected diff).
- Mark `surface-snapshot` command `Hidden: true` (CI tool, not user-facing).
- Hide `--config-path` from help (keep functional).

### 1.6 One agent-discovery schema
- Move all catalog building into `internal/agent`: `--help --agent` and `wenmar commands` both call the same `CommandInfo` builder. Fields: `path`, `description`, `aliases`, `args` (positional specs), `flags[]` (`name`, `short`, `type`, `required`, `default`, `description`), `children[]`, `group`.
- `required` reads cobra's required-flag annotation (`cobra_annotation_bash_completion_one_required_flag`), fixing both surfaces (help mode currently reports false for required flags; catalog mode reports `Annotations != nil`, which is true for any annotated flag).
- Global output flags are excluded from per-command flag lists; the schema documents them once (revisited in Phase 3 when they collapse).
- Document the JSON schema in `wenmar help agent-help` (new topic) and SKILL.md.

### 1.7 Test credential isolation
- Introduce `WENMAR_CONFIG_HOME` env var honored by `internal/config` (path resolution) and the SDK credential store (constructor injection in `internal/auth`); all tests set it to `t.TempDir()`.
- Add a `CredentialStore` interface in `internal/auth` (keyring impl + file impl for tests). `TestDoctor_NoToken` must stop deleting the developer's real token; `TestAuthLogin_StoresToken` must stop writing the real keyring.
- CI asserts no test run touches `$HOME/.config/wenmar` (a pre/post checksum in the CI job).

### 1.8 TUI core-loop fixes
- Re-arm the tick: `case tickMsg: return m, tea.Batch(m.tabs[m.active].Init(), tick(m.interval))` — periodic refresh actually refreshes (`internal/tui/app.go:121-123`).
- Route only `tea.KeyMsg` through the topbar/sidebar early-return paths; list-result messages and ticks always reach the active tab (`app.go:128-140`).
- Real debounce: capture query at emit; skip stale `searchFilterMsg` if it matches the current query and no newer keystroke intervened (or cancel/rearm a timer). No refetch per keystroke.
- Thread a cancellable `context.Context` through fetches (created in `NewApp`, cancelled on `tea.Quit`); replace the 24 `context.Background()` uses.
- Rune-safe `truncate` (bounds-checked `[]rune` slicing); pad raw status before styling (fixes column drift); render `wo.UpdatedAt` in the "Updated" column instead of poll time; delete the dead `contentWidth` block and `tuiRemote` flag.
- Add tests: tick re-arm cycle, result-msg delivery while search focused, truncate with multibyte input.

### 1.9 Config safety
- Atomic writes: write `path.tmp`, fsync, `os.Rename` — for config, trusted-repos, and migration (`internal/config/config.go:87-103`, `repo.go:99`, `xdg.go`).
- Migration (`xdg.go:56-66`): write the **real** data atomically to the new path first; only then remove the old file, checking that error. No window where an empty config is the only readable one.
- `os.MkdirAll(dir, 0700)` for config dirs; move the migration notice out of `internal/config` (return `Migrated bool`, cmd prints).
- Validate on load: `base_url` parses with scheme http(s); `auth_method ∈ {static, oauth}`; canonicalize trusted-repo paths via `filepath.Abs`+`EvalSymlinks` before compare/store.

### 1.10 Watch resilience
- Transient poll errors (network, 5xx, rate-limit) retry with capped exponential backoff (e.g. base 1s, ×2, max 60s, reset on success); only auth errors and N consecutive failures (default 3) are fatal (`internal/watch/poller.go:60-63`).
- Ctrl-C returns clean: `runWatch` maps `context.Canceled` → nil exit; no "ERROR: context canceled".
- `decodeList` returns an error on JSON round-trip failure instead of silently dropping items (phantom new/removed events).
- Build the scoped client once in `Run`; drop the single-goroutine mutex or document it.
- `watch --run-sync`: surface script exit status (event field + stderr notice); `--run-async`: bounded concurrency, checked `Encode` errors.

### 1.11 Exit codes and errors
- Status fallbacks in `internal/errors/exit.go`: 401→2 (auth), 404→3 (not found), 422→4 (validation) when the error `Code` is unrecognized; keep code-based mapping primary.
- Offline detection: match `net.Error`, `*url.Error`, and `syscall.ECONNREFUSED`/`EHOSTUNREACH` via `errors.Is` → exit 10.
- Add hints for conflict (7: "resource already exists (e.g. duplicate VIN)") and forbidden (8) to `printHints`.
- `auth status` returns a typed error instead of `os.Exit(2)` mid-function (`cmd/auth.go:159`); stop double-printing (print-and-return).
- OAuth hardening: `ExchangeCode` uses the bounded callback context (`flow.go:65-75`); `GenerateVerifier`/`GenerateState` return errors instead of panicking; state-mismatch error no longer echoes the secret state.
- Error-hygiene sweep: check every `json.Encoder.Encode`; check `EvalSymlinks`, `UserHomeDir`, `Chmod` in upgrade path; `id must be an integer` errors include the offending value.

### 1.12 Auth and output consolidation
- `internal/auth`: one resolution chain — `ResolveAuthManager*` delegates to `ResolveTokenWithSourceFrom`; collapse the resolver family into `Resolve(opts)`; distinguish keyring-infrastructure errors from "no token" ("keyring unavailable: …" with a store-source fallback).
- Store construction and save/delete move behind `internal/auth` (`auth.Store(token)`, `auth.Clear()`); logout surfaces delete errors instead of `_ =`.
- Delete `cmd/setup.go`'s duplicate `maskToken`; use `errors.MaskToken` everywhere (security-relevant dedup).
- Delete dead `CaptureBreadcrumbs` (`internal/output/output.go:94-99` — latent token-leak vector via `os.Args`); one shared `normalize()` for `toMaps`/`toSlice`/`renderIDsOnly`/`renderCount`; typed envelope struct with stable field order.

**Phase 1 exit criteria:** every listed bug has a regression test; `go test -race ./...` green; fake-API integration tests cover the previously-ignored flags (e.g. `customers update --remove-email` asserts the PATCH body contains the id).

---

## Phase 2 — Generator becomes the source of truth

The structural refactor. Requires Phase 1.1 (generator no longer crashes).

### 2.1 Generator completion
- **Action handlers**: emit real handlers for body-less POST/PUT operations (`service-categories deactivate/reactivate/move-up/move-down/seed-defaults`, `tags` mutations) — parse id, call SDK method, render. No more "action %s not yet generated" stubs.
- **`request_struct` auto-derivation** where the SDK names follow the pattern; keep explicit override as escape hatch.
- **Pagination honesty**: operations whose SDK wrapper has a `WithPagination` variant generate `runListPaginatedWithAll`-style handlers; `paginated: false` and non-paginated ops use plain list. Fixes drivers/vendors/statements claiming "paginated via Link header" while dropping pagination.
- **Example fields**: generator emits `Example:` per command from spec `summary`/`examples` where present (see 3.3).
- **Breadcrumbs**: create commands emit the created ID from the response (no more `show 0` suggestions); each resource supplies its own breadcrumb verbs (only `show`/`list` pairs that exist).

### 2.2 Generic runners
Replace the 50× getter-closure boilerplate and `any`-typed body builders with Go generics in `cmd/runners.go` (shared by generated and hand-written commands):

```go
func Show[T any](ctx *RunCtx, fn func(context.Context, *wenmar.Client, int) (T, error)) error
func List[T any](ctx *RunCtx, fn ListFunc[T]) error            // + paginated variant
func Create[B any, R any](ctx *RunCtx, build func(*cobra.Command) B, send SendFunc[B, R]) error
```

- `RunCtx` carries the per-invocation state that is currently global: resolved output mode, debug info (returned, not stored in `currentDebugInfo`), location, base URL, token source. Built from `cmd.Flags()` inside `RunE`.
- Generated `RunE` bodies become one-liners: `return runners.Show(ctx, client.ShowCustomer)`.
- `runShow`/`runShowStr`, `runListPaginated`/`runListPaginatedWithAll`, `runAction`/`runServiceCategoryAction` twins collapse into the generic set.
- Package-level flag vars and `currentDebugInfo` are deleted; integration tests stop resetting globals (removes the `integration_test.go:26-28` whitelist fragility).

### 2.3 Cutover
1. Fix generator (2.1), add golden tests (2.4) — generated build passes the **full** existing suite, not the vacuous whitelist. Delete `make test-generated`'s stale `-run` patterns (`TestVendors`, `TestStatements` etc. match nothing).
2. Extend `gen_overrides.yaml` to cover the currently-excluded operations (`create_customer` with `--full-name` via now-wired `flag_overrides`, `delete_customer`, `update_tags`, `create_customer_tag`, `create_vehicle_tag`).
3. Generate all resource files; replace the hand-written twins file-by-file, porting any behavior only present in the hand-written copy (e.g. `--dry-run` on deletes) into generator/shared-runner support.
4. Delete hand-written resource files and the `generated` build tag; commit generated output with the `DO NOT EDIT` header.
5. `gen_overrides.yaml` stays as the single place humans shape ergonomics (summaries, flag names, aliases).

### 2.4 Generator tests and drift gate
- **Golden tests**: fixture spec (small YAML checked in) → generate → compare against golden files. Also unit-test `sdkMethodName`, `goType`, overrides parsing, exclusion handling.
- **CI regen-drift job**: fetches the pinned SDK's spec (or vendors a spec snapshot under `cmd/gencli/testdata/`), runs `make generate`, fails on `git diff --exit-code` in `cmd/gen_*.go`. The spec can never drift from the CLI surface silently.
- `make check-published` and `make surface-diff` become required CI steps on every PR to main.

### 2.5 Naming migration (D2)

Generator owns resource names; overrides declare canonical + aliases.

| Canonical | Permanent aliases |
|---|---|
| `workorders` | `work_orders`, `wo` |
| `servicecategories` | `service-categories`, `sc` |
| `customers workorders` | `customers work-orders` (nested) |
| `vehicles workorders` | `vehicles work-orders` (nested) |

- Flags stay kebab-case (`--full-name`, `--decode-vin`). No snake_case CLI tokens anywhere; env vars stay SCREAMING_SNAKE (`WENMAR_LOCATION_ID`).
- `locations show <id>` and `account show` keep their shapes; parent `Short` strings become noun-y ("Manage work orders").
- Update `surface-snapshot.json` (aliases included), README, SKILL.md together in the same commit.

### 2.6 CLI conventions skill
Create `.opencode/skill/cli-conventions/SKILL.md` capturing the naming rules for future contributors and agents:

> Resource nouns are squashed compounds (`todolist`, `workorders`, `servicecategories`) — never hyphenated or snake_cased. Multi-word **flags** are kebab-case (`--message-board`). Subcommands are space-separated verb/noun tokens (`customers workorders list`); hyphens inside a single token only for flags. snake_case appears only in API/JSON field names, never as CLI-facing tokens. Renames ship with permanent backward-compatible aliases. Add command `Example` fields whenever adding commands.

**Phase 2 exit criteria:** `grep -r "go:build generated"` returns nothing; generated files compile as the default build; full suite green with no whitelist; CI regen-drift job active; naming canonical with aliases; conventions skill committed.

---

## Phase 3 — Interface and help overhaul

### 3.1 `--output` (D3)
- Modes: `table` (default human), `md`, `json` (envelope), `agent` (raw), `quiet` (raw, no discovery), `ids-only`, `count`, `html`, `styled` (forces table when piped).
- `--output <mode>` registered as a persistent flag on root, shown at every help level.
- Sugar aliases kept as thin persistent flags that only set the mode: `--json`, `--agent`, `--quiet`, `--jq <expr>` (implies `json` + filter). Conflicts (`--json --output md`) error with a clear message.
- Dropped standalone flags: `--md`, `-m`, `--markdown`, `--ids-only`, `--count`, `--html`, `--styled`. **Prerelease break, documented in README migration note.** `--allow-partial` remains a behavior flag.
- `internal/output.ResolveModeStyled`'s 9-boolean signature becomes `ResolveMode(out string, jq string, sugar Sugar) Mode` — boolean-blindness removed.

### 3.2 Help structure
- Cobra command groups: **Resources** (account, customers, drivers, locations, servicecategories, statements, tags, vehicles, vendors, workorders), **Session** (auth, config, setup, doctor, upgrade), **Agents** (commands, help, skill, url), **Platform** (completion, tui, watch). Root `Long` gains: 3-line overview, output modes summary, pointer to `wenmar setup` and `wenmar help <topic>`.
- Leaf help: local flags first; output flags collapsed to one line (`--output <mode>` + sugar) with pointer to `wenmar help output`; globals (`--token`, `--base-url`, `--location`, `--debug`, `--allow-partial`) grouped separately.
- Root footer: `Run 'wenmar help' for topics: output, exit-codes, auth, environment, location, watch, agent-help`.
- Enable cobra suggestions for near-miss commands and flags ("Did you mean 'workorders'?").

### 3.3 Examples everywhere
- Every resource command gets an `Example` field (generator emits from overrides; a `examples:` section in `gen_overrides.yaml` supplies copy). House style: one read, one filtered, one `--agent` pipe per resource.
- `help_topics` gain inline examples; `wenmar help output` documents every mode + `--output` migration table.

### 3.4 Exit-code enforcement (D5)
- Table (below) is the single source of truth, rendered identically in `wenmar help exit-codes`, README, and SKILL.md: 0 success · 1 generic · 2 auth · 3 not found · 4 validation · 5 rate limited · 6 server · 7 conflict · 8 forbidden · 9 truncated (use `--allow-partial`) · 10 offline.
- A Go test asserts the three renderings stay in sync (parse topic text and SKILL.md; compare to `internal/errors` constants).

**Phase 3 exit criteria:** `wenmar --help` fits one screen with groups; `wenmar customers show 42 --help` shows local flags first and one output line; every command has an Example; exit-code renderings match by test.

---

## Phase 4 — Agent skill rewrite and release hardening

### 4.1 SKILL.md rewrite (D4)
Written for agents, verified against the binary, covering:
- Auth: `WENMAR_TOKEN`, `wenmar setup`, `wenmar auth login --token <t>` (non-interactive persistence — the agent-recommended path), `auth status`, `auth token`, `auth logout`, keyring notes.
- Preflight invariants: `wenmar doctor`, `wenmar commands`, explicit `--output`/`--agent` always (document the pipe-auto-JSON behavior honestly, `--output styled` to override).
- Location scoping: `--location`, `WENMAR_LOCATION_ID`, when it's needed.
- Output modes incl. `--allow-partial` and exit 9; full exit-code table 0–10; error-output shape and `--debug`.
- Pagination: Link header, `--page`, `--all`.
- `--help --agent` + `wenmar commands` schema (pointing at `wenmar help agent-help`).
- Per-resource examples: every resource, at least list/show/create-with-flags.
- Deleted fabricated claims: no `--force` on deletes, no bulk ops, no `--status`/`--overdue`/`--plate`/`--state` flags, no "auto-detection is not used".

### 4.2 Skill freshness gate
- Test parses SKILL.md code blocks; for every `wenmar …` invocation line, asserts the command path + flags exist in the `wenmar commands` catalog (run via the cobra root against the fresh build). Any fabricated claim fails CI. Same test covers README's usage blocks.

### 4.3 Release hardening
- `release.yml` runs the full test suite (and `make surface-diff`) before goreleaser.
- goreleaser: add `sboms:` (syft) and `attestations:`; keep cosign keyless + checksum chain.
- Publish the package managers README promises (brew tap, scoop) or delete those README sections — decide at implementation; goreleaser supports both via `brews:`/`scoops:`.
- CI gains a cross-compile smoke job (`GOOS=darwin,windows GOARCH=amd64,arm64 go build ./...`) so all six goreleaser targets compile before tag day.
- shellcheck job on `install-cli`.
- Commit `docs/`; tick completed plan boxes.

**Phase 4 exit criteria:** SKILL.md freshness test green; release publishes signed+SBOM'd artifacts; cross-compile matrix green; README install instructions all real.

---

## Testing strategy

| Layer | Approach |
|---|---|
| Generator | Golden-file tests (fixture spec → generated code), unit tests for name/type mapping and overrides parsing |
| Commands | Existing `startFakeAPI` httptest harness (extended to five untested resources: vendors, drivers, statements, tags, servicecategories); regression test per Phase 1 fix |
| Agent surfaces | One test asserts `--help --agent` and `wenmar commands` agree for sampled commands (same builder → same output) |
| Contract | Exit-code table sync test; SKILL.md/README freshness test; surface-snapshot diff in CI |
| Config/auth | `WENMAR_CONFIG_HOME`-isolated, fake credential store; migration-failure injection (make second write fail, assert old data intact) |
| Watch | Three-poll test: new → changed → removed; transient-failure backoff test; filter tests |
| TUI | tick re-arm cycle; result-delivery while search focused; multibyte truncate; debounce |
| Race/coverage | `go test -race -cover` in CI; no coverage floor yet, report only |
| Install script | shellcheck + a smoke test against a fake release via `WENMAR_RELEASES_BASE` |

## Compatibility and migration

- Renames: canonical + permanent aliases (cobra `Aliases`); `surface-snapshot.json` records aliases so CI catches accidental removals.
- Flag removals (`--md` etc.): README migration table (`--md` → `--output md`); sugar flags `--json`/`--agent`/`--quiet`/`--jq` unchanged so the most common scripts never break.
- Exit codes: no values change; only unmapped paths gain the documented value (strictly script-friendly).
- Help topics: no removals; `agent-help` topic added.

## Execution order and milestones

1. **Phase 0 PR** — green main. (Half day.)
2. **Phase 1** — sequence 1.1 → 1.2/1.3/1.5/1.6 (interface truth) → 1.4 → 1.7 → 1.11 → 1.9/1.10 → 1.8 → 1.12. Ship as ~8 focused PRs; each leaves main green. (1–2 weeks.)
3. **Phase 2** — generator completion + golden tests, then cutover PR, then naming PR. (1 week.)
4. **Phase 3** — `--output` PR, then help-groups PR, then examples PR. (2–3 days.)
5. **Phase 4** — SKILL.md + freshness gate, release hardening. (2–3 days.)

Hard rule: never mix phases in one PR; never start Phase 2 before Phase 0/1 land (the cutover is irreversible once committed generated code replaces the twins).

## Risks and mitigations

| Risk | Mitigation |
|---|---|
| Generated code can't express a hand-written nuance (e.g. `--dry-run`, merge flows) | Port such behaviors into shared runners/generator features **before** deleting twins; cutover PR is reviewed against a full surface-diff |
| SDK spec shape changes break golden files | Golden tests use a small checked-in fixture spec, not the live spec; live-spec drift is the regen-drift job's concern |
| `--output` breaks someone's prerelease script | Keep `--json`/`--agent`/`--quiet`/`--jq` sugar; migration table in README; it's prerelease |
| Naming aliases mask typos (agent types `work_orders` intending `workorders`) | Aliases resolve to the same command; cobra suggestions steer to canonical; SKILL.md teaches canonical only |
| TUI behavioral fixes change snapshot-y tests | Update tests in same PR; TUI has no golden output, only structural assertions |
| OAuth/security fixes regress login | Existing oauth flow/pkce/callback/exchange test files cover the paths; extend for the new bounded exchange context |