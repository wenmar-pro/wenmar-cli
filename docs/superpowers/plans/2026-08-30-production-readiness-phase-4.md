# Wenmar CLI Production-Readiness — Phase 4 Implementation Plan (Agent Skill Rewrite & Release Hardening)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-ruby:subagent-driven-development (recommended) or superpowers-ruby:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the release: SKILL.md rewritten against verified CLI reality and guarded by a CI freshness test that parses every documented command; goreleaser hardened with SBOMs and attestations; release workflow gated on tests; the package-manager promises in the README made real or removed; `url parse` tracks the canonical resource names.

**Architecture:** The freshness test is the keystone — it parses SKILL.md (and README) code fences, extracts every `wenmar ...` invocation, and resolves each against the live cobra root, so fabricated commands/flags can never reach an agent again. Release hardening is config-only: goreleaser v2 native `sboms`/`attestations`, a test job before goreleaser in the workflow, and brew/scoop publishing wired to match the README (or the README claims deleted — decided in Task 6 per what's maintainable).

**Tech Stack:** Go 1.27, cobra, committed generated commands (post-Phase-2), goreleaser v2, cosign, syft, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-08-30-production-readiness-refactor-design.md` (§Phase 4, decisions D4 + D5)
**Prerequisites:** Phases 0-3 landed. This plan assumes: `--output <mode>` interface with hidden sugar flags; `workorders`/`servicecategories` canonical names with aliases; help groups; examples on commands; the exit-code sync test exists (Phase 3 Task 7); CI already gates fmt/vet/lint/vulncheck/race/surface-diff/regen-drift.

**Conventions:** Run `go build ./... && go test ./...` before every commit. All commands from repo root. Commit style: `feat:`, `fix:`, `test:`, `ci:`, `docs:`, `chore:`.

---

## Verified ground truth (do not re-derive)

- **SKILL.md today** (232 lines) still contains, post-Phase-1: the fabricated `--force` invariant (#3/#5), `work_orders list --status active`/`--overdue` and `--vehicle-id=<id>` (decision tree, lines 75-79 — none of these flags exist), `vehicles list --plate <plate> --state <state>` (line 87 — no such flags), "Auto-detection is not used" (line 126-127 — contradicts the pipe auto-switch), and the exit-code table truncated at 6 of 11 (lines 190-198). Phase 3 Task 8 migrated the flag surface (`--md` → `--output md`) but did NOT touch these — they're Phase 4's targets.
- **The bundled-skill mechanism:** `cmd/skill.go` installs SKILL.md from `<binary-dir>/../skills/wenmar` (a repo-layout fallback, skill.go:50-61). This means the INSTALLED binary serves whatever SKILL.md shipped NEXT TO it — for released binaries, `skills/wenmar/SKILL.md` must be bundled into the archive (goreleaser `extra_files` — NOT in .goreleaser.yml today) or the skill commands break on installed copies. This is a latent release bug nobody caught: Task 5 fixes it.
- **`url parse` known resources** (url.go:48-55): `customers, vehicles, work_orders, account, locations` — hardcoded list that must track Phase 2 renames (`workorders` canonical, `work_orders` still valid as an alias AND as an API route segment — the API itself serves `/work_orders` paths, so BOTH belong in the parser's known set, plus `servicecategories`/`service_categories`, `vendors`, `drivers`, `statements`, `tags`).
- **Release workflow** (release.yml): runs goreleaser directly on tag push — no build/test gate, no snapshot, no SBOM, no attestations, draft release. `.goreleaser.yml` has cosign keyless checksum signing only; `before:` hook runs `go mod tidy` (mutates tree during release — must go).
- **README promises** (lines 38-55): brew tap, scoop, AUR (`yay`), mise — none published by .goreleaser.yml (no `brews:`/`scoops:`/`aurs:` blocks). All four are dead instructions today.
- **`wenmar commands` catalog** (`internal/agent.BuildCatalog`): includes aliases as separate entries with `canonical: false, compatibility_for: <path>` (discovery.go:98-110) — the freshness test must accept BOTH canonical paths and alias paths.
- **cobra root for resolution:** `cmd.RootCmd()` is exported (root.go:70). `rootCmd.Find([]string)` resolves a path + leftover args, returning an error for unknown commands (Phase 1 Task 8 uses it in `help`). Flag checking per command: `cmd.Flags().Lookup(name)` — but generated/hand-written commands bind flags in `init()` at package load, so a test in `package cmd` sees the fully-built tree.
- **Help topics exist:** `output`, `exit-codes`, `environment`, `auth`, `location`, `watch`, `agent-help` (Phase 1 Task 10). The freshness test should also verify the topic names SKILL.md references exist in `helpTopics`.
- **Exit codes** (internal/errors/exit.go:10-22): 0-10 with status fallbacks (Phase 1 Task 15). The Phase 3 sync test already binds topic+README+SKILL.md to the constants — SKILL.md's table MUST therefore have 11 rows before the freshness work begins, or that test fails. Phase 1 Task 18 was supposed to do this; verify state at implementation time.

**Design decisions:**

| Choice | Rationale |
|---|---|
| Freshness test lives in `package cmd` | Needs the built cobra tree; reuses `RootCmd()` |
| Parses SKILL.md + README code fences | Both are agent/user-facing promises; same harness |
| Resolves command paths AND flags; tolerates `<placeholder>` tokens | Fabricated flags are exactly the failure mode we're closing |
| SKILL.md rewrite comes BEFORE the freshness test goes live | The test's first run must pass on the rewritten file — write docs, then lock them |
| Skill bundling via goreleaser `extra_files` | Fixes the installed-binary skill bug with 5 lines of config |
| `go mod tidy` before-hook removed | Release must never mutate the tree; tidy is a dev action enforced by CI |
| Package managers: wire brew + scoop via goreleaser, DROP AUR + mise claims | goreleaser supports `brews:`/`scoops:` natively; AUR needs a separate repo/workflow and mise needs a manifest+registrations — not worth prerelease maintenance; delete those README lines instead of shipping dead promises |

---

## Task 1: `url parse` tracks the canonical resource names

**Files:**
- Modify: `cmd/url.go:48-55` (knownResources)
- Modify: `cmd/url_test.go` (cases for new resources and aliases)

Small, self-contained, and unblocks SKILL.md's URL section (which documents what `url parse` recognizes). Done first so the freshness test can include `url parse` examples.

- [ ] **Step 1: Failing tests**

Add to `cmd/url_test.go` (read its existing table shape first and extend it):

```go
func TestParseWenmarURL_CanonicalAndLegacyResources(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want string // resource_type ("" = unknown)
	}{
		{"workorders canonical", "https://app.wenmarpro.com/workorders/42.json", "workorders"},
		{"work_orders legacy still parses", "https://app.wenmarpro.com/work_orders/42.json", "work_orders"},
		{"servicecategories", "https://app.wenmarpro.com/servicecategories/7.json", "servicecategories"},
		{"service-categories legacy", "https://app.wenmarpro.com/service-categories/7.json", "service-categories"},
		{"vendors", "https://app.wenmarpro.com/vendors/3.json", "vendors"},
		{"drivers", "https://app.wenmarpro.com/drivers/9.json", "drivers"},
		{"statements", "https://app.wenmarpro.com/statements/9001.json", "statements"},
		{"tags", "https://app.wenmarpro.com/customer_tags/5.json", "customer_tags"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseWenmarURL(tc.url)
			if got.ResourceType != tc.want {
				t.Errorf("parseWenmarURL(%q).ResourceType = %q, want %q", tc.url, got.ResourceType, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Verify it fails**

```bash
go test ./cmd/ -run TestParseWenmarURL_CanonicalAndLegacy -v
```

Expected: FAIL — the known set is `customers, vehicles, work_orders, account, locations` only.

- [ ] **Step 3: Extend knownResources**

```go
// knownResources are the API route segments the parser recognizes. Both
// canonical CLI names and legacy/alias spellings appear because URLs come
// from the API/web app, which may use either over time.
var knownResources = map[string]bool{
	"customers":           true,
	"vehicles":            true,
	"workorders":          true,
	"work_orders":         true,
	"account":             true,
	"locations":           true,
	"servicecategories":   true,
	"service_categories":  true,
	"service-categories":  true,
	"vendors":             true,
	"drivers":             true,
	"statements":          true,
	"customer_tags":       true,
	"vehicle_tags":        true,
	"settings":            true, // /settings/tags → tags mutation surface
}
```

Note: read `parseWenmarURL`'s segment-matching logic first (url.go:60-130) — it takes the FIRST path segment; `/settings/tags` would report `settings` with the rest in `Path`. If two-segment recognition for tags is out of scope, drop the `settings` entry and the tags test cases — one-segment recognition only for the tags routes is a documented limitation in SKILL.md instead. Decide by reading the parser; keep the change minimal and honest.

- [ ] **Step 4: Run + commit**

```bash
go test ./cmd/ -run TestParseWenmarURL -v && go test ./...
git add cmd/url.go cmd/url_test.go
git commit -m "fix(url): parse recognizes canonical and legacy resource names"
```

### Task 2: SKILL.md full rewrite

**Files:**
- Rewrite: `skills/wenmar/SKILL.md`

The complete file. Every command below is verified against the post-Phase-3 surface (canonical names, `--output`, sugar flags, real pagination). Written for an agent that has NEVER seen the tool.

- [ ] **Step 1: Write the new SKILL.md**

```markdown
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
   aliases, required flags)
3. `wenmar <command> --help --agent` — structured JSON for ONE command
4. Read `wenmar help output` and `wenmar help exit-codes` for the contracts

## Agent invariants

These rules MUST be followed without exception:

1. **Parse VINs first** with `wenmar vehicles decode-vin <vin>` — never
   assume a VIN format.
2. **Resource IDs are positional**, never flags: `wenmar customers show 42`.
3. **Preview destructive ops** with `--dry-run` where offered
   (vehicles/drivers/workorders/servicecategories delete). There is no
   `--force` flag; a delete without `--dry-run` executes immediately.
4. **Set an explicit output mode** for anything parsed: `--output agent`
   (or the `--agent` shorthand). Default piped output is raw JSON, but
   explicit beats implicit.
5. **Never combine output flags**: `--output` alone, or exactly one of
   `--json`/`--agent`/`--quiet`/`--jq`. Mixing them errors.
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

`--output <mode>` is canonical:

| Mode | What you get |
|------|--------------|
| `table` | Human table (terminal default) |
| `md` | GFM table |
| `json` | Full envelope `{ok, data, summary, meta, breadcrumbs}` |
| `agent` | Raw JSON data, no envelope |
| `quiet` | Raw JSON; the default when piped |
| `ids-only` | One ID per line (shell loops) |
| `count` | Bare integer count |
| `html` | HTML document |
| `styled` | Force tables even when piped |

Shorthands (hidden from subcommand help, still work): `--json`, `--agent`
(also makes `--help` emit JSON), `--quiet`, `--jq 'expr'` (implies json).
Piped stdout with no explicit mode → `quiet`. Conflicts error out.

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
wenmar customers list --query "jane" --all          # full-text + all pages
wenmar customers list --output ids-only
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
wenmar workorders list --output count
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
wenmar vehicles delete 42 --dry-run
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
wenmar tags list
wenmar tags create --name "Fleet A"
wenmar tags create --type vehicle --name "Priority"
```

### Location scoping

```bash
wenmar --location loc_abc workorders list     # or WENMAR_LOCATION_ID env,
                                             # or location_id in config
wenmar help location                         # full topic
```

### Pagination

Lists paginate via the Link header. `customers list` supports `--page N`,
`--per-page N`, and `--all` (follow every page). When `meta.has_next` is
true in `--output json`, more pages exist.

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
- **Nested objects truncate in tables** — use `--output json` or `--jq`
  for full detail.
- **`customers update` cannot change emails/addresses/tags** — the update
  API only accepts phone changes (with `--remove-phone` by ID) and scalar
  fields. Nested emails/addresses/tags are set at CREATE time only.
- **The API is additive-only** — new fields may appear; existing meanings hold.
```

- [ ] **Step 2: Verify EVERY command in the file against the binary**

This is the manual pre-check before the automated gate (Task 3) exists:

```bash
go build -o wenmar ./cmd/wenmar
# Extract every wenmar invocation from code fences and run --help on each:
grep -oE 'wenmar [a-z_ -]+' skills/wenmar/SKILL.md | sort -u | while read -r cmd; do
  ./wenmar $(echo "$cmd" | sed 's/^wenmar //' | tr -d '`' | sed 's/[<"].*//') --help > /dev/null 2>&1 || echo "MISSING: $cmd"
done
```

Expected: no MISSING lines. Investigate any hit — it's either a doc typo or a real surface gap.

- [ ] **Step 3: Cross-check the exit-code table**

```bash
./wenmar help exit-codes
```

All 11 rows must match SKILL.md's table (the Phase 3 sync test enforces this — run it):

```bash
go test ./cmd/ -run TestExitCodeTableSync -v
```

- [ ] **Step 4: Commit**

```bash
git add skills/wenmar/SKILL.md
git commit -m "docs(skill): rewrite SKILL.md against the verified CLI surface"
```

### Task 3: The freshness test

**Files:**
- Create: `cmd/docs_freshness_test.go`

Parses SKILL.md and README code fences, extracts `wenmar` invocations, and resolves each against the cobra tree. Any fabricated command or flag fails CI forever.

- [ ] **Step 1: Write the harness**

```go
package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// docInvocation is one wenmar command line extracted from documentation.
type docInvocation struct {
	Source  string // "SKILL.md" | "README.md"
	Line    int
	Raw     string
	Path    []string // command path tokens (canonical or alias)
	Flags   []string // flag names used (without --)
}

var wenmarLineRe = regexp.MustCompile(`(?m)^\s*(?:\$ )?wenmar\s+([^` + "`" + `\n]+)`)

// extractInvocations parses every `wenmar ...` line from doc code fences.
func extractInvocations(t *testing.T, path string) []docInvocation {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	src := string(data)

	// Extract fenced code blocks only — prose mentions of wenmar flags
	// (like this test's own comments) must not be validated.
	var invocations []docInvocation
	inFence := false
	line := 0
	for _, l := range strings.Split(src, "\n") {
		line++
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inFence = !inFence
			continue
        }
		if !inFence {
			continue
		}
		m := wenmarLineRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		inv := docInvocation{Source: path, Line: line, Raw: m[0]}
		tokens := strings.Fields(m[1])
		for _, tok := range tokens {
			tok = strings.Trim(tok, `"`)
			if strings.HasPrefix(tok, "--") {
				inv.Flags = append(inv.Flags, strings.TrimLeft(tok, "-"))
				continue
			}
			if strings.HasPrefix(tok, "<") || strings.HasPrefix(tok, "$") {
				break // placeholder or env-var token — stop path building
			}
			if strings.ContainsAny(tok, "<>|") {
				break // placeholder syntax — stop path building
			}
			inv.Path = append(inv.Path, tok)
		}
		if len(inv.Path) > 0 {
			invocations = append(invocations, inv)
		}
	}
	return invocations
}

// resolveCommand walks the cobra tree along path (resolving aliases).
// Returns the deepest command found and any leftover path tokens.
func resolveCommand(root *cobra.Command, path []string) (*cobra.Command, []string, error) {
	cmd := root
	remaining := path
	for len(remaining) > 0 {
		next := cmd.Find([]string{remaining[0]})
		if next == nil || next == cmd {
			return cmd, remaining, fmt.Errorf("unknown command %q", remaining[0])
		}
		cmd = next
		remaining = remaining[1:]
	}
	return cmd, remaining, nil
}

func TestDocsFreshness(t *testing.T) {
	docs := []string{
		"../skills/wenmar/SKILL.md",
		"../README.md",
	}
	for _, doc := range docs {
		t.Run(doc, func(t *testing.T) {
			invocations := extractInvocations(t, doc)
			if len(invocations) < 10 {
				t.Fatalf("suspiciously few invocations parsed (%d) — parser regression?", len(invocations))
			}
			for _, inv := range invocations {
				inv := inv
				t.Run(fmt.Sprintf("L%d:%s", inv.Line, strings.Join(inv.Path, " ")), func(t *testing.T) {
					cmd, leftover, err := resolveCommand(rootCmd, inv.Path)
					if err != nil {
						t.Errorf("%s:%d: %q — %v (docs drifted from the CLI)", inv.Source, inv.Line, inv.Raw, err)
						return
					}
					// Leftover tokens are positional args — allowed only if
					// the command expects them (a non-nil Args validator).
					if len(leftover) > 0 && cmd.Args == nil {
						t.Errorf("%s:%d: %q passes %d positional args to %q which accepts none",
							inv.Source, inv.Line, inv.Raw, len(leftover), cmd.CommandPath())
					}
					// Every flag must exist on the command (or be a global).
					for _, flag := range inv.Flags {
						if cmd.Flags().Lookup(flag) != nil {
							continue
						}
						if rootCmd.PersistentFlags().Lookup(flag) != nil {
							continue
						}
						t.Errorf("%s:%d: %q — flag --%s does not exist on %q",
							inv.Source, inv.Line, inv.Raw, flag, cmd.CommandPath())
					}
				})
			}
		})
	}
}
```

(Note: `leftover` is scoped inside the subtest closure — no stray references outside it. Quoted example values like `"jane doe"` become leftover positional tokens; commands with `Args` validators such as `ExactArgs(1)` accept them, which is the desired freshness behavior. Placeholder-bearing lines are covered by the Task 2 Step 2 manual sweep instead.)

- [ ] **Step 2: Handle the README's install-script and completion examples**

Some README blocks contain `wenmar completion bash > ...` (flags fine, redirect fine) and multi-line curl pipes — the regex only matches lines STARTING with `wenmar`, so curl lines are skipped naturally. Lines like `wenmar completion bash > ~/.local/share/...` parse as path `[completion, bash]` — `bash` is a leftover positional; `completion` has `Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs)` post-Phase-1 — `cmd.Args != nil`, so leftover passes. Good.

Edge case to verify when first run: `wenmar --location loc_abc workorders list` — the FIRST token after `wenmar` is a flag, so path-building never starts and Path stays empty → skipped by `len(inv.Path) > 0`. The freshness test intentionally skips flag-first invocations (cobra parses flags anywhere, but doc examples should lead with the command for readability — enforce that in the doc style instead).

- [ ] **Step 3: Run it — expect failures, fix docs not the test**

```bash
go test ./cmd/ -run TestDocsFreshness -v
```

Expected: FAIL on any stale doc line. Fix each by correcting the DOC (the CLI is post-Phase-3 truth). Iterate until green. The subtest names (`L<line>:<path>`) pinpoint exactly what to fix.

- [ ] **Step 4: Commit**

```bash
git add cmd/docs_freshness_test.go skills/wenmar/SKILL.md README.md
git commit -m "test: docs freshness — every documented wenmar command resolves and flags exist"
```

### Task 4: Release workflow gates + goreleaser hardening

**Files:**
- Modify: `.github/workflows/release.yml` (test gate before goreleaser)
- Modify: `.goreleaser.yml` (drop go mod tidy hook; SBOMs; attestations; extra_files for the skill)

- [ ] **Step 1: release.yml — tests before goreleaser**

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.27"

      - name: Configure Go module auth for private Wenmar repos
        run: |
          git config --global url."https://x-access-token:${{ secrets.GITHUB_TOKEN }}@github.com/Wenmar-Pro/".insteadOf "https://github.com/Wenmar-Pro/"

      - name: Build
        run: go build ./...

      - name: Test
        run: go test -race ./...

  release:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.27"

      - name: Configure Go module auth for private Wenmar repos
        run: |
          git config --global url."https://x-access-token:${{ secrets.GITHUB_TOKEN }}@github.com/Wenmar-Pro/".insteadOf "https://github.com/Wenmar-Pro/"

      - name: Install cosign
        uses: sigstore/cosign-installer@v3

      - name: Run goreleaser
        uses: goreleaser/goreleaser-action@v7
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          COSIGN_EXPERIMENTAL: "1"
```

- [ ] **Step 2: .goreleaser.yml — hardening**

```yaml
version: 2

project_name: wenmar

builds:
  - id: wenmar
    main: ./cmd/wenmar
    binary: wenmar
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X github.com/wenmar-pro/wenmar-cli/cmd.version={{.Version}} -X github.com/wenmar-pro/wenmar-cli/cmd.commit={{.Commit}} -X github.com/wenmar-pro/wenmar-cli/cmd.date={{.Date}}

archives:
  - id: default
    formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    # Bundle the agent skill with every archive so `wenmar skill install`
    # works from installed binaries (was broken: the binary looked for
    # skills/wenmar/SKILL.md next to itself and released archives never
    # shipped it).
    files:
      - skills/wenmar/SKILL.md

sboms:
  - id: default
    artifacts: archive

nfpms:
  - id: packages
    package_name: wenmar
    file_name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    formats:
      - deb
      - rpm
      - apk
    homepage: https://github.com/Wenmar-Pro/wenmar-cli
    description: Wenmar Pro automotive shop management CLI
    maintainer: Wenmar Pro <support@wenmarpro.com>
    license: MIT
    contents:
      - src: skills/wenmar/SKILL.md
        dst: /usr/share/wenmar/skills/wenmar/SKILL.md

checksum:
  name_template: "checksums.txt"

signs:
  - id: cosign
    cmd: cosign
    signature: "${artifact}.bundle"
    args:
      - sign-blob
      - --bundle
      - "${signature}"
      - "${artifact}"
    artifacts: checksum
    output: true

# Provenance attestations for the checksums (keyless, matches the existing
# cosign flow). Requires the workflow to run with the default permissions
# plus id-token: write — added in release.yml below.
attestations:
  - id: default
    artifacts: checksum
    cmd: cosign
    args:
      - attest-blob
      - --bundle
      - "${artifact}"
      - "${artifact}"

release:
  draft: true

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^chore:"
```

Changes: `before: go mod tidy` DELETED (release never mutates the tree; tidy is CI-enforced); `files: skills/wenmar/SKILL.md` added to archives (fixes the installed-skill bug — Task 5 covers the path-resolution side); `contents:` ships the skill in deb/rpm/apk to `/usr/share/wenmar/...`; `sboms:` + `attestations:` added.

- [ ] **Step 3: Add id-token permission for attestations**

In release.yml's `release` job permissions:

```yaml
permissions:
  contents: write
  id-token: write   # cosign keyless attestations
```

(The top-level `permissions: contents: write` moves to job level — the test job needs none.)

- [ ] **Step 4: Validate the config locally**

```bash
go run github.com/goreleaser/goreleaser/v2@latest check    # or: goreleaser check if installed
```

Expected: no config errors. If the `attestations` block's args syntax differs for goreleaser v2's current release (it has changed across versions — verify against the installed version's docs with `goreleaser help attestations` or the JSON schema), adjust to the validated form. Config must pass `check` before commit.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/release.yml .goreleaser.yml
git commit -m "ci: release gated on tests; SBOMs, attestations, skill bundled in archives"
```

### Task 5: Skill path resolution for installed binaries

**Files:**
- Modify: `cmd/skill.go:45-62` (bundledSkillDir candidates)
- Test: `cmd/skill_test.go` (new cases)

The `skills/wenmar/SKILL.md` bundle fixes archives (Task 4), but nfpms install the skill to `/usr/share/wenmar/skills/wenmar/SKILL.md` while the binary lands in `/usr/bin` — `bundledSkillDir`'s relative candidates (`<exe-dir>/skills/wenmar`, `<exe-dir>/../skills/wenmar`) never find it. Add absolute fallbacks.

- [ ] **Step 1: Failing test**

```go
func TestBundledSkillDirFindsSharePath(t *testing.T) {
	// Simulate an installed layout: binary in a temp "bin", skill in a
	// sibling "share/wenmar/skills/wenmar".
	root := t.TempDir()
	binDir := filepath.Join(root, "usr", "bin")
	shareSkill := filepath.Join(root, "usr", "share", "wenmar", "skills", "wenmar")
	if err := os.MkdirAll(shareSkill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shareSkill, "SKILL.md"), []byte("# x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := bundledSkillDirFrom(binDir)
	if err != nil {
		t.Fatalf("bundledSkillDirFrom: %v", err)
	}
	if got != shareSkill {
		t.Errorf("bundledSkillDirFrom = %q, want %q", got, shareSkill)
	}
}
```

- [ ] **Step 2: Refactor for testability**

Extract the candidate resolution from `bundledSkillDir` into a testable form:

```go
// bundledSkillDirFrom returns the skill directory for the given binary
// directory. Candidate locations cover the repo layout, the goreleaser
// archive layout (binary next to skills/), and package installs
// (/usr/bin + /usr/share/wenmar/skills).
func bundledSkillDirFrom(binDir string) (string, error) {
	candidates := []string{
		filepath.Join(binDir, "skills", "wenmar"),
		filepath.Join(binDir, "..", "skills", "wenmar"),
		filepath.Join(binDir, "..", "share", "wenmar", "skills", "wenmar"),
		filepath.Join(binDir, "..", "..", "share", "wenmar", "skills", "wenmar"),
		"/usr/share/wenmar/skills/wenmar",
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "SKILL.md")); err == nil {
			return filepath.Clean(c), nil
		}
	}
	return "", fmt.Errorf("bundled skill not found near the wenmar binary (looked in %v)", candidates)
}

func bundledSkillDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return bundledSkillDirFrom(filepath.Dir(exe))
}
```

Note the `/usr/share` absolute candidate: on a package install the binary is at `/usr/bin/wenmar`, so `..` climbs to `/usr` — the relative `../share/...` candidates cover it. The absolute path is a belt-and-suspenders fallback for `/usr/local/bin` style layouts (climbing hits `/usr/local/share/wenmar/...` — also covered by relative candidates). Keep both.

- [ ] **Step 3: Run + commit**

```bash
go test ./cmd/ -run TestBundledSkillDir -v && go test ./...
git add cmd/skill.go cmd/skill_test.go
git commit -m "fix(skill): resolve bundled SKILL.md from archive and package layouts"
```

### Task 6: Package-manager promises — publish brew + scoop, drop AUR + mise

**Files:**
- Modify: `.goreleaser.yml` (brews, scoops)
- Modify: `README.md` (drop AUR/mise lines; note brew/scoop)
- Create: tap repo setup notes (documented in the task, not a repo file)

- [ ] **Step 1: goreleaser blocks**

```yaml
brews:
  - id: default
    name: wenmar
    repository:
      owner: wenmar-pro
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    homepage: https://github.com/wenmar-pro/wenmar-cli
    description: Wenmar Pro automotive shop management CLI
    license: MIT
    install: |
      bin.install "wenmar"
    commit_author:
      name: wenmar-release-bot
      email: support@wenmarpro.com

scoops:
  - id: default
    name: wenmar
    repository:
      owner: wenmar-pro
      name: scoop-bucket
      token: "{{ .Env.SCOOP_BUCKET_TOKEN }}"
    homepage: https://github.com/wenmar-pro/wenmar-cli
    description: Wenmar Pro automotive shop management CLI
    license: MIT
    commit_author:
      name: wenmar-release-bot
      email: support@wenmarpro.com
```

Prerequisites (document in the release process — a short `docs/RELEASE.md`, see Step 3):
1. Create `wenmar-pro/homebrew-tap` and `wenmar-pro/scoop-bucket` repos (empty).
2. Create a PAT with `repo` scope for the tap commits; add as the `HOMEBREW_TAP_TOKEN` and `SCOOP_BUCKET_TOKEN` secrets on the wenmar-cli repo.

- [ ] **Step 2: README install section**

```markdown
### Package managers

```bash
# Homebrew (macOS / Linux)
brew install wenmar-pro/tap/wenmar

# Scoop (Windows)
scoop bucket add wenmar https://github.com/wenmar-pro/scoop-bucket
scoop install wenmar

# deb / rpm / apk
# Available as release assets (wenmar_<version>_<os>_<arch>.deb / .rpm / .apk)
```
```

Delete the AUR (`yay`) and mise lines — dead promises until someone owns those repositories.

- [ ] **Step 3: docs/RELEASE.md**

```markdown
# Releasing

1. Ensure CI is green on main (fmt/vet/lint/vulncheck/test/surface-diff/regen-drift).
2. Bump nothing by hand — goreleaser derives the version from the tag.
3. Tag: `git tag v0.x.0 && git push origin v0.x.0`.
4. The Release workflow runs: test job → goreleaser (builds, SBOMs,
   cosign-signs checksums, attests, drafts the release, updates the
   Homebrew tap and Scoop bucket).
5. Review the draft release, then publish.
6. The install-cli script picks up the new version immediately (it resolves
   latest from the releases page).
```

- [ ] **Step 4: Validate + commit**

```bash
goreleaser check   # or the go-run equivalent from Task 4 Step 4
git add .goreleaser.yml README.md docs/RELEASE.md
git commit -m "ci: publish homebrew tap and scoop bucket; drop AUR/mise claims; document release process"
```

### Task 7: Skill content test — the installed copy matches the repo copy

**Files:**
- Create: `cmd/skill_content_test.go`

The freshness test (Task 3) validates the REPO's SKILL.md. This test guards the SHIPPING path: `skill install` must copy the same bytes, and the bundled-candidate lookup must find THIS repo's file during tests.

- [ ] **Step 1: Write the test**

Create `cmd/skill_content_test.go`:

```go
package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/agent"
)

func TestSkillInstallCopiesRepoSkillVerbatim(t *testing.T) {
	// The repo's skills/wenmar/SKILL.md is the source of truth. Locate it
	// through the same candidates the binary uses (repo layout), then
	// install and byte-compare.
	source, err := bundledSkillDir()
	if err != nil {
		t.Skipf("skill source not found in this layout: %v", err)
	}
	want, err := os.ReadFile(filepath.Join(source, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(t.TempDir(), "skill")
	if err := agent.InstallSkill(source, target, true); err != nil {
		t.Fatalf("install: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(target, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Error("installed SKILL.md differs from the repo copy — shipping path corrupted")
	}
}
```

- [ ] **Step 2: Run + commit**

```bash
go test ./cmd/ -run TestSkillInstallCopiesRepoSkill -v
git add cmd/skill_content_test.go
git commit -m "test(skill): install copies the repo SKILL.md verbatim"
```

### Task 8: Final release-readiness gate

**Files:**
- Verification only; fixes as found.

- [ ] **Step 1: Full local gate**

```bash
go vet ./... && gofmt -l cmd/ internal/ && go test -race ./... -count=1
make regen-drift
make surface-diff
go test ./cmd/ -run "TestDocsFreshness|TestExitCodeTableSync|TestSkillInstall" -v
```

Expected: all green.

- [ ] **Step 2: End-to-end smoke on a fresh build**

```bash
go build -o wenmar ./cmd/wenmar
./wenmar                                    # welcome + setup
./wenmar --help                            # groups, topics footer
./wenmar commands | head -20               # catalog
./wenmar help output exit-codes auth location watch agent-help environment
./wenmar doctor                            # against a fake or real instance
./wenmar skill install && cat ~/.agents/skills/wenmar/SKILL.md | head -5
./wenmar skill uninstall
./wenmar url parse "https://app.wenmarpro.com/workorders/42.json"
```

- [ ] **Step 3: Release dry-run**

```bash
goreleaser release --clean --snapshot   # builds all 6 targets + packages locally
ls dist/ | head -20
```

Expected: 6 archives with `skills/wenmar/SKILL.md` inside (verify: `tar -tzf dist/wenmar_*_linux_amd64.tar.gz | grep SKILL`), checksums, SBOMs (`*.sbom.json` or `.spdx.json` depending on version), `.bundle` signatures.

- [ ] **Step 4: The v0.1.0 release**

Follow `docs/RELEASE.md`: tag, watch the workflow (test job → release job), review the draft, publish. The freshness/sync/surface gates mean what's documented is what shipped.

- [ ] **Step 5: Commit any fixes found + close out**

```bash
git add -A && git commit -m "chore: release-readiness fixes from the final gate"
```

---

## Self-review notes

- **Spec coverage (§Phase 4):** 4.1 SKILL.md rewrite → Task 2 (all deleted fabrications enumerated; auth storage, location scoping, --allow-partial/exit 9, --help --agent schema via `agent-help` topic reference, help topics, watch, per-resource examples, error guidance via exit-code table + invariants). 4.2 freshness gate → Task 3 (SKILL.md + README; command paths AND flags; alias-tolerant via cobra's Find). 4.3 release hardening → Tasks 4-6 (test gate, SBOMs, attestations, skill bundling, skill path resolution, brew/scoop published, AUR/mise claims deleted, RELEASE.md, `go mod tidy` hook removed). Also caught beyond spec: the bundled-skill path bug for installed binaries (Task 5) and nfpms `contents` for the skill — the skill system was broken for every non-repo install since inception.
- **Deliberate non-goals:** no `.opencode` project-skill update for SKILL.md content (the skill file itself is the artifact); no watch/TUI doc depth beyond one-liners (commands' own help + examples cover them); no changelog automation beyond goreleaser's default.
- **Type consistency:** `bundledSkillDirFrom(binDir)` defined Task 5, used in Task 7. `agent.InstallSkill(source, target, force)` signature matches internal/agent/install.go:40. The freshness test's `resolveCommand` uses `cobra.Find` per token — alias resolution is native to Find, matching the catalog's alias entries.
- **Risk notes:** (1) Task 3's first run will surface every stale doc line — budget an iteration loop fixing DOCS, never the test (unless the parser itself has a bug: distinguish by whether the flagged CLI feature actually exists — check `./wenmar <cmd> --help`). (2) goreleaser v2's `attestations` config syntax varies by version — Task 4 Step 4 makes `goreleaser check` the gate before commit; adjust args to the validated form rather than trusting this plan's transcription. (3) The brew/scoop taps need PATs (`HOMEBREW_TAP_TOKEN`, `SCOOP_BUCKET_TOKEN`) created BEFORE the first tagged release after Task 6 lands — otherwise that release's publish step fails; RELEASE.md documents this. (4) Task 2's verification sweep (Step 2) uses a shell loop that can't handle placeholders — the freshness test deliberately skips them too; placeholder-bearing lines are verified by the Task 8 Step 2 manual smoke.
- **Ordering:** Task 1 first (url.go feeds SKILL.md's URL section); Task 2 before Task 3 (test locks the rewritten doc); Tasks 4-6 are config/doc (any order, but 4 before 6 since brews/scoops join the same config file); Task 7 anytime after 2; Task 8 last. Never mix Task 3's doc fixes with parser changes in one commit — reviewer must see doc-only diffs when the test drives them.