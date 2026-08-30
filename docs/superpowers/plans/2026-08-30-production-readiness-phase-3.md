# Wenmar CLI Production-Readiness — Phase 3 Implementation Plan (Interface & Help Overhaul)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-ruby:subagent-driven-development (recommended) or superpowers-ruby:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Collapse the 11 output-mode flags into a canonical `--output <mode>` (with `--json`/`--agent`/`--quiet`/`--jq` as hidden sugar), organize help with cobra command groups, put examples on every command, and keep all three exit-code renderings (help topic, README, SKILL.md) in sync by test.

**Architecture:** Mode resolution moves from 9 positional booleans to `output.ParseMode(ModeSpec)` returning `(Mode, error)` — unknown modes and flag conflicts fail fast in `PersistentPreRunE` before any API call. Sugar flags stay functional but hidden from leaf help; discovery moves to the root Long, the `wenmar help output` topic, and docs. Groups come from cobra's native `Group`/`GroupID`. Examples flow through the generator's overrides so they regenerate with commands.

**Tech Stack:** Go 1.27, cobra/pflag, committed generated commands (post-Phase-2), GitHub Actions CI.

**Spec:** `docs/superpowers/specs/2026-08-30-production-readiness-refactor-design.md` (§Phase 3, decisions D3 + D5)
**Prerequisites:** Phase 0/1 landed; Phase 2 cutover landed (this plan assumes generated `cmd/gen_*.go` files are the canonical resource commands and `workorders`/`servicecategories` naming with aliases is in place). If Phase 2 has NOT landed, Tasks 1-4 and 6-9 still apply but Task 5's generator pieces and the group/example wiring must target hand-written files instead.

**Conventions:** Run `go build ./... && go test ./...` before every commit. All commands from repo root. Commit style: `feat:`, `fix:`, `test:`, `docs:`, `chore:`.

---

## Verified ground truth (do not re-derive)

- Mode resolution today: `output.ResolveModeStyled(md, json, agent, quiet, idsOnly, count, jq, html, styled bool)` — 9 positional params, precedence chain jq > count > ids > agent > quiet > json > html > md > styled, then pipe auto-switch to `ModeQuiet` (output.go:47-86). Callers: `cmd/runners.go:15` (`resolveMode()`), `cmd/url.go:33` (direct call).
- Sugar/flag var inventory (root.go:15-31): `mdFlag` (bound TWICE: `--md`/`-m` and `--markdown`), `jsonFlag`, `agentFlag`, `quietFlag`, `jqFlag`, `idsOnlyFlag`, `countFlag`, `htmlFlag`, `styledFlag`, `allowPartial`, plus `tokenFlag`/`baseURLFlag`/`locationFlag`/`configPathFlag`/`debugFlag`.
- `output.Mode` constants (output.go:13-23): `ModeDefault, ModeMD, ModeJSON, ModeAgent, ModeJQ, ModeQuiet, ModeIDsOnly, ModeCount, ModeHTML`. There is NO ModeStyled — `styled` returns `ModeDefault` (explicit mode, so no pipe auto-switch).
- Mode-string consumers beyond runners: `cmd/doctor.go:136` checks `jsonFlag` directly (renders its own `{ok, checks}` envelope, NOT the standard envelope — only the mode check needs updating, not the renderer); `cmd/help_topics.go:141,155` checks `agentFlag` for structured help; `cmd/breadcrumbs.go:46` checks `allowPartial` (stays as a behavior flag, untouched).
- Tests referencing dropped flags: `cmd/integration_test.go:327` (`--md`), `:573` (`--count`), `:588` (`--markdown`); `execute()` resets `mdFlag, jsonFlag, agentFlag, jqFlag, idsOnlyFlag, countFlag` (lines ~26-28, may have moved after Phase 1). `internal/output/output_test.go` tests `ResolveModeStyled` directly.
- cobra native support (verified via go doc): `cobra.Group{ID, Title}`, `Command.AddGroup(...)`, `Command.GroupID`, `Command.Example` string field, `Command.SetHelpFunc`, `pflag.Flag.Hidden`. Group titles render as section headers in `Available Commands:` automatically.
- Generated commands come from `cmd/gencli/gen.go` `emitGroup`/`emitCommand` — the parent dict is built in `emitGroup` (gen.go:228-247 post-Phase-2), the command dict in `emitCommand` (gen.go:253-269). `CommandOverride` lives in `cmd/gencli/main.go`.
- Docs claiming dropped flags: `README.md:113-122` (output-mode table), `skills/wenmar/SKILL.md:38,60-62,117-144` (multiple `--md`/`--ids-only`/`--count` references).
- Root command today: `Short: "Wenmar Pro API CLI"`, one-line `Long`, `SilenceUsage`/`SilenceErrors` true, `SuggestionsMinimumDistance: 2` (Phase 1 Task 7), custom `SetHelpFunc` wrapping agent mode (Phase 1 Task 10).

**Design decisions (from spec D3 + review feedback):**

| Choice | Rationale |
|---|---|
| `--output <mode>` is the only visible mode flag | One canonical entry point; 16-flag wall drops to 6 globals at leaf level |
| `--json`/`--agent`/`--quiet`/`--jq` stay functional but `Hidden: true` | The "2-4 most-typed sugar" kept per D3; hidden because cobra cannot show a flag at root but hide it at leaves — root's `Long` lists them explicitly instead (discovery preserved, leaves stay clean) |
| Dropped: `--md`/`-m`/`--markdown`, `--ids-only`, `--count`, `--html`, `--styled` | Rare modes become `--output md` etc.; prerelease break, documented in a migration table |
| Conflicts error instead of silent precedence | `--json --output md` and `--json --agent` fail with a clear message — the old jq-wins-everything chain was a silent-wrongness trap |
| `--allow-partial` untouched | Response-acceptance behavior, not an output mode |
| Mode validation in a new `PersistentPreRunE` | Fail fast BEFORE any API call; runners keep a defensive `resolveMode()` that re-validates and propagates |
| Groups via cobra `GroupID` | Native section headers in help; no template surgery |
| Examples via generator overrides | Regenerate with commands; companions set `Example` directly |

---

## Task 1: `output.ParseMode` — the new mode resolver

**Files:**
- Modify: `internal/output/output.go` (add ModeSpec/ParseMode; deprecate ResolveModeStyled)
- Test: `internal/output/output_test.go` (migrate + new cases)

- [ ] **Step 1: Write the failing tests first**

Add to `internal/output/output_test.go` (keep existing render tests; migrate the `ResolveModeStyled` tests — see Step 3):

```go
func TestParseMode(t *testing.T) {
	cases := []struct {
		name    string
		spec    ModeSpec
		want    Mode
		wantErr bool
	}{
		{name: "output table", spec: ModeSpec{Output: "table"}, want: ModeDefault},
		{name: "output md", spec: ModeSpec{Output: "md"}, want: ModeMD},
		{name: "output json", spec: ModeSpec{Output: "json"}, want: ModeJSON},
		{name: "output agent", spec: ModeSpec{Output: "agent"}, want: ModeAgent},
		{name: "output quiet", spec: ModeSpec{Output: "quiet"}, want: ModeQuiet},
		{name: "output ids-only", spec: ModeSpec{Output: "ids-only"}, want: ModeIDsOnly},
		{name: "output count", spec: ModeSpec{Output: "count"}, want: ModeCount},
		{name: "output html", spec: ModeSpec{Output: "html"}, want: ModeHTML},
		{name: "output styled forces table", spec: ModeSpec{Output: "styled"}, want: ModeDefault},
		{name: "sugar json", spec: ModeSpec{JSON: true}, want: ModeJSON},
		{name: "sugar agent", spec: ModeSpec{Agent: true}, want: ModeAgent},
		{name: "sugar quiet", spec: ModeSpec{Quiet: true}, want: ModeQuiet},
		{name: "jq implies json mode", spec: ModeSpec{JQ: ".[]"}, want: ModeJQ},
		{name: "unknown mode errors", spec: ModeSpec{Output: "yaml"}, wantErr: true},
		{name: "output plus sugar conflicts", spec: ModeSpec{Output: "md", JSON: true}, wantErr: true},
		{name: "two sugars conflict", spec: ModeSpec{JSON: true, Agent: true}, wantErr: true},
		{name: "jq plus output conflicts", spec: ModeSpec{Output: "md", JQ: ".[]"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseMode(tc.spec)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseMode(%+v) expected error, got mode %v", tc.spec, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMode(%+v): %v", tc.spec, err)
			}
			if got != tc.want {
				t.Errorf("ParseMode(%+v) = %v, want %v", tc.spec, got, tc.want)
			}
		})
	}
}

func TestParseModeAutoSwitch(t *testing.T) {
	// ParseMode must not auto-switch here: the pipe check happens inside
	// ParseMode only when nothing explicit is set. This test pins that the
	// explicit table mode survives even when piped (test stdout is not a
	// TTY in CI).
	got, err := ParseMode(ModeSpec{Output: "table"})
	if err != nil {
		t.Fatal(err)
	}
	if got != ModeDefault {
		t.Errorf("explicit table mode should win over pipe auto-switch, got %v", got)
	}
	// With nothing set and piped stdout, quiet is chosen.
	got, err = ParseMode(ModeSpec{})
	if err != nil {
		t.Fatal(err)
	}
	if got != ModeQuiet {
		t.Errorf("piped stdout with no explicit mode should auto-switch to quiet, got %v", got)
	}
}
```

- [ ] **Step 2: Verify they fail**

Run: `go test ./internal/output/ -run TestParseMode -v`
Expected: FAIL — `ModeSpec`/`ParseMode` undefined.

- [ ] **Step 3: Implement in internal/output/output.go**

Add below the Mode constants (keep `ResolveMode`/`ResolveModeStyled` for now — deleted in Task 2 once no callers remain):

```go
// ModeSpec carries the output-mode flags from the command line. --output
// is the canonical selector; --json/--agent/--quiet/--jq are sugar.
type ModeSpec struct {
	Output string
	JSON   bool
	Agent  bool
	Quiet  bool
	JQ     string
}

// modeNames maps --output values to Modes. "styled" and "table" both render
// the default table; "styled" exists to override the pipe auto-switch.
var modeNames = map[string]Mode{
	"table":    ModeDefault,
	"styled":   ModeDefault,
	"md":       ModeMD,
	"json":     ModeJSON,
	"agent":    ModeAgent,
	"quiet":    ModeQuiet,
	"ids-only": ModeIDsOnly,
	"count":    ModeCount,
	"html":     ModeHTML,
}

// ParseMode resolves the output mode from ModeSpec. It errors on unknown
// mode names and on combinations that would silently pick a winner
// (--output plus any sugar, or two sugar flags together). With nothing set
// and a piped stdout it auto-switches to ModeQuiet so piped output is
// machine-readable.
func ParseMode(spec ModeSpec) (Mode, error) {
	if spec.Output != "" {
		if spec.JSON || spec.Agent || spec.Quiet || spec.JQ != "" {
			return 0, fmt.Errorf("--output cannot be combined with --json/--agent/--quiet/--jq — pick one")
		}
		m, ok := modeNames[spec.Output]
		if !ok {
			return 0, fmt.Errorf("unknown output mode %q (valid: table, md, json, agent, quiet, ids-only, count, html, styled)", spec.Output)
		}
		return m, nil
	}

	n := 0
	for _, set := range []bool{spec.JSON, spec.Agent, spec.Quiet, spec.JQ != ""} {
		if set {
			n++
		}
	}
	if n > 1 {
		return 0, fmt.Errorf("conflicting output flags — use --output <mode> to select one")
	}
	switch {
	case spec.JQ != "":
		return ModeJQ, nil
	case spec.Quiet:
		return ModeQuiet, nil
	case spec.Agent:
		return ModeAgent, nil
	case spec.JSON:
		return ModeJSON, nil
	}

	// Auto-switch: piped stdout gets machine-readable raw JSON.
	if !isTerminal(os.Stdout) {
		return ModeQuiet, nil
	}
	return ModeDefault, nil
}
```

- [ ] **Step 4: Migrate the old resolver's tests, then run the package**

Read `internal/output/output_test.go` for the existing `ResolveModeStyled` tests. Convert their cases into `ParseMode` cases (each boolean input becomes either a ModeSpec sugar field or an `--output` string; the old precedence-chain cases like "jq beats count" become conflict-error cases). Delete tests for flag combinations that are now errors. Then:

```bash
go test ./internal/output/ -v
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/output/
git commit -m "feat(output): ParseMode with --output mode names and conflict errors"
```

### Task 2: Root flag migration — `--output` in, wall flags out, fail-fast validation

**Files:**
- Modify: `cmd/root.go` (flag registration, PersistentPreRunE)
- Modify: `cmd/runners.go` (resolveMode signature + all call sites)
- Modify: `cmd/url.go:33` (use resolveMode)
- Modify: `cmd/doctor.go:136` (honor --output json too)
- Modify: `cmd/integration_test.go` (execute() resets, flag tests) — full test migration is Task 3; this task only keeps the build green
- Delete: `ResolveMode`/`ResolveModeStyled` from `internal/output/output.go` when no callers remain

- [ ] **Step 1: Root flag registration (root.go:72-89)**

Replace the output-flag block. New registration:

```go
func init() {
	rootCmd.SetVersionTemplate(versionString() + "\n")
	rootCmd.PersistentFlags().StringVar(&tokenFlag, "token", "", "API bearer token (or set WENMAR_TOKEN env)")
	rootCmd.PersistentFlags().StringVar(&baseURLFlag, "base-url", "", "API base URL (default: https://app.wenmarpro.com, or set WENMAR_URL)")
	rootCmd.PersistentFlags().StringVar(&locationFlag, "location", "", "Location ID to scope requests (or set WENMAR_LOCATION_ID)")
	rootCmd.PersistentFlags().StringVar(&outputFlag, "output", "", "Output mode: table, md, json, agent, quiet, ids-only, count, html, styled (see 'wenmar help output')")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Shorthand for --output json")
	rootCmd.PersistentFlags().BoolVar(&agentFlag, "agent", false, "Shorthand for --output agent (also makes --help emit JSON)")
	rootCmd.PersistentFlags().BoolVar(&quietFlag, "quiet", false, "Shorthand for --output quiet")
	rootCmd.PersistentFlags().StringVar(&jqFlag, "jq", "", "jq filter expression (shorthand for --output json plus a filter)")
	rootCmd.PersistentFlags().BoolVar(&allowPartial, "allow-partial", false, "Accept truncated responses (adds a notice to the envelope)")
	rootCmd.PersistentFlags().StringVar(&configPathFlag, "config-path", "", "Path to config file")
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Print request debug info (token source, base URL, method/path) to stderr")

	// Sugar flags work everywhere but stay out of leaf help; the root
	// Long and `wenmar help output` are their discovery surfaces.
	rootCmd.PersistentFlags().MarkHidden("json")
	rootCmd.PersistentFlags().MarkHidden("agent")
	rootCmd.PersistentFlags().MarkHidden("quiet")
	rootCmd.PersistentFlags().MarkHidden("jq")
	rootCmd.PersistentFlags().MarkHidden("config-path")

	// Fail fast on mode conflicts/typos BEFORE any command runs (and
	// before any API call inside it).
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		_, err := output.ParseMode(modeSpec())
		return err
	}

	// ... existing PersistentPostRunE (debug) and SetHelpFunc unchanged
}
```

And the var block at root.go:15-31: delete `mdFlag, idsOnlyFlag, countFlag, htmlFlag, styledFlag`; add `outputFlag string`.

- [ ] **Step 2: modeSpec + resolveMode (runners.go:13-16)**

```go
// modeSpec snapshots the output-mode flags for ParseMode.
func modeSpec() output.ModeSpec {
	return output.ModeSpec{
		Output: outputFlag,
		JSON:   jsonFlag,
		Agent:  agentFlag,
		Quiet:  quietFlag,
		JQ:     jqFlag,
	}
}

// resolveMode resolves the output mode. ParseMode validated the flags in
// PersistentPreRunE; the error path here is defensive for direct handler
// calls (tests) and still propagates.
func resolveMode() (output.Mode, error) {
	return output.ParseMode(modeSpec())
}
```

- [ ] **Step 3: Sweep resolveMode call sites**

`grep -n "resolveMode()" cmd/*.go` — every site changes from `mode := resolveMode()` to:

```go
	mode, err := resolveMode()
	if err != nil {
		return err
	}
```

Sites (verified inventory): runners.go runShow, runShowStr, runList, runListPaginated, runListPaginatedWithAll, runCreate, runAction, runDelete, runActionNoBody, runSeedAction (post-Phase-2); tags.go runTagsMutation; work_orders_extras.go runWorkOrdersShow, runWorkOrdersTab. Each is the same 4-line replacement — the surrounding code already has `err` in scope and returns error. Mechanical sweep; verify with `go build ./...`.

Also: `cmd/url.go:33` — replace the direct `output.ResolveModeStyled(...)` call with `mode, err := resolveMode(); if err != nil { return err }`.

Also: `cmd/doctor.go:136` — change `if jsonFlag {` to:

```go
	mode, err := resolveMode()
	if err != nil {
		return err
	}
	if mode == output.ModeJSON {
```

(doctor's `{ok, checks}` renderer stays as-is — it is intentionally not the standard envelope.)

- [ ] **Step 4: Delete the old resolver**

```bash
grep -rn "ResolveModeStyled\|ResolveMode(" --include="*.go" cmd/ internal/
```

When only output.go's definitions and their tests remain: delete `ResolveMode` and `ResolveModeStyled` from output.go and remove their direct tests (already migrated in Task 1 Step 4).

- [ ] **Step 5: Verify manually**

```bash
go build -o wenmar ./cmd/wenmar
./wenmar customers list --output bogus; echo "exit=$?"        # error: unknown mode, exit 1, no API call
./wenmar customers list --json --output md; echo "exit=$?"    # error: conflict
./wenmar customers list --json --agent; echo "exit=$?"        # error: conflict
./wenmar --help                                               # Global Flags: --output, --token, --base-url, --location, --debug, --allow-partial only
./wenmar customers list --help                                 # same short globals; local flags first
```

- [ ] **Step 6: Commit**

```bash
go build ./... && go test ./...
git add cmd/ internal/output/
git commit -m "feat: --output <mode> is canonical; sugar flags hidden; conflicts fail fast"
```

### Task 3: Test migration for the flag surface

**Files:**
- Modify: `cmd/integration_test.go`, `cmd/url_test.go`, `cmd/doctor_test.go`, any test using dropped flags
- Modify: `cmd/agent_surface_test.go` (Phase 1's TestAgentSurfacesAgree sets `agentFlag` — still valid)

- [ ] **Step 1: Find every dropped-flag reference in tests**

```bash
grep -rn '"--md"\|"-m"\|"--markdown"\|"--ids-only"\|"--count"\|"--html"\|"--styled"' cmd/*_test.go
```

Verified hits to fix: integration_test.go:327 (`--md`), :573 (`--count`), :588 (`--markdown`). For each, migrate:

- `--md` → `--output`, `md` (two args)
- `--count` → `--output`, `count`
- `--markdown` → repurpose the test: it asserted alias behavior; now assert `--markdown` FAILS as an unknown flag:

```go
func TestMarkdownFlagRemoved(t *testing.T) {
	_, err := execute("customers", "list", "--markdown")
	if err == nil {
		t.Error("--markdown was dropped; it must error (use --output md)")
	}
}
```

Same treatment for `--md`, `-m`, `--ids-only`, `--count`, `--html`, `--styled`: one table-driven removal test covering all six:

```go
func TestDroppedOutputFlagsRemoved(t *testing.T) {
	dropped := []string{"--md", "-m", "--markdown", "--ids-only", "--count", "--html", "--styled"}
	for _, flag := range dropped {
		t.Run(flag, func(t *testing.T) {
			_, err := execute("customers", "list", flag)
			if err == nil {
				t.Errorf("%s was dropped; it must error (see --output)", flag)
			}
		})
	}
}
```

- [ ] **Step 2: Add positive tests for the new surface**

```go
func TestOutputModeFlag(t *testing.T) {
	srv := startFakeAPI(t, "tok-out")
	defer srv.Close()
	t.Setenv("WENMAR_URL", srv.URL)

	out, err := execute("customers", "list", "--output", "md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "|") || !strings.Contains(out, "full_name") {
		t.Errorf("--output md should render a GFM table, got:\n%s", out)
	}

	out, err = execute("customers", "list", "--output", "ids-only")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.FieldsFunc(out, func(r rune) bool { return r == '\n' })
	if len(lines) != 2 || lines[0] != "1" || lines[1] != "2" {
		t.Errorf("--output ids-only should print one ID per line, got %q", out)
	}
}

func TestOutputModeConflictFailsFast(t *testing.T) {
	srv := startFakeAPI(t, "tok-conflict")
	defer srv.Close()
	t.Setenv("WENMAR_URL", srv.URL)

	_, err := execute("customers", "list", "--json", "--output", "md")
	if err == nil {
		t.Fatal("conflicting mode flags must error")
	}
	// Fail-fast means NO API call was made: assert the fake server saw nothing.
	if n := srvRequestCount(); n != 0 {
		t.Errorf("conflict validation should run before any API call; saw %d requests", n)
	}
}
```

`srvRequestCount()` needs a counter in the fake API — add to `startFakeAPI`:

```go
var fakeAPIRequests atomic.Int32 // reset in startFakeAPI; incremented in the auth check
```

Increment at the top of every handler (or in a shared middleware wrapper around the mux):

```go
muxWithCount := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fakeAPIRequests.Add(1)
	mux.ServeHTTP(w, r)
})
return httptest.NewServer(muxWithCount)
```

- [ ] **Step 3: Update execute() reset list**

```go
	outputFlag, jsonFlag, agentFlag, quietFlag, jqFlag = "", false, false, false, ""
	currentDebugInfo = nil
```

(delete the mdFlag/idsOnlyFlag/countFlag/htmlFlag/styledFlag resets — vars gone).

- [ ] **Step 4: Run everything**

```bash
go test ./... -count=1
```

Expected: green. Any test asserting the OLD precedence chain (jq-beats-count etc.) gets the same conflict-error rewrite as Task 1 Step 4.

- [ ] **Step 5: Commit**

```bash
git add cmd/
git commit -m "test: migrate flag surface to --output; dropped flags error"
```

### Task 4: Help groups, root Long, and clean leaf help

**Files:**
- Modify: `cmd/root.go` (AddGroup, GroupID assignments for hand-written commands, new Long)
- Modify: `cmd/gencli/gen.go` (emit `GroupID: "resources"` on generated parents)
- Modify: `cmd/tags.go`, `cmd/customers_extras.go`, `cmd/work_orders_extras.go` (GroupID on companion parents — generated parents carry it already; companions add subcommands to generated parents so only their OWN commands need GroupID where they define new parents — tags.go defines `tagsCmd`)
- Regenerate: `cmd/gen_*.go`
- Test: `cmd/help_groups_test.go`

- [ ] **Step 1: Failing test**

```go
package cmd

import (
	"strings"
	"testing"
)

func TestHelpGroups(t *testing.T) {
	out, err := execute("--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, section := range []string{"Resources:", "Session & Config:", "Agents & Discovery:", "Platform:"} {
		if !strings.Contains(out, section) {
			t.Errorf("root help missing group %q; got:\n%s", section, out)
		}
	}
	// Resources section lists the resource commands.
	if !strings.Contains(out, "workorders") {
		t.Error("workorders should appear under Resources")
	}
	// No ungrouped orphans section with our commands in it.
	for _, orphan := range []string{"setup", "doctor", "completion", "tui", "watch"} {
		if orphanSectionContains(out, orphan) {
			t.Errorf("%s should be grouped, not in Additional Commands", orphan)
		}
	}
}

func orphanSectionContains(out, name string) bool {
	// cobra renders ungrouped commands under "Available Commands:" and
	// grouped ones under their group titles. Simplest robust check: the
	// command must appear BELOW a group title line, not under a bare
	// "Available Commands:" header. (Implementation: split output at the
	// first group title; anything before it must not contain the name.)
	idx := strings.Index(out, "Resources:")
	if idx < 0 {
		return false // no groups at all — the outer test already failed
	}
	return strings.Contains(out[:idx], "  "+name+" ")
}
```

- [ ] **Step 2: Verify it fails**

```bash
go test ./cmd/ -run TestHelpGroups -v
```

Expected: FAIL — flat command list today.

- [ ] **Step 3: Root groups + Long (root.go)**

In root.go init (or at the top of the file with rootCmd's definition):

```go
var (
	groupResources = &cobra.Group{ID: "resources", Title: "Resources"}
	groupSession   = &cobra.Group{ID: "session", Title: "Session & Config"}
	groupAgents    = &cobra.Group{ID: "agents", Title: "Agents & Discovery"}
	groupPlatform  = &cobra.Group{ID: "platform", Title: "Platform"}
)

func init() {
	rootCmd.AddGroup(groupResources, groupSession, groupAgents, groupPlatform)
	// ...
}

var rootCmd = &cobra.Command{
	Use:   "wenmar",
	Short: "Wenmar Pro API CLI",
	Long: `A command-line interface for the Wenmar Pro automotive shop
management API.

Getting started:
  wenmar setup        Configure your API token (or export WENMAR_TOKEN)

Output:
  --output <mode>    table | md | json | agent | quiet | ids-only |
                      count | html | styled  (default: table on a
                      terminal, quiet when piped)
  Quick flags: --json --agent --quiet --jq   (see 'wenmar help output')

Topics:
  wenmar help output      All output modes and the envelope format
  wenmar help exit-codes  The stable 0-10 exit-code contract
  wenmar help auth        Token sources and auth methods
  wenmar help location    Location scoping
  wenmar help environment Environment variables
  wenmar help watch       The watch command
  wenmar help agent-help  Structured --agent --help for AI agents`,
	Version:                  version,
	SilenceUsage:             true,
	SilenceErrors:            true,
	SuggestionsMinimumDistance: 2,
}
```

GroupID assignments (add `GroupID: "..."` to each hand-written command var):

- `session`: authCmd, configCmd, setupCmd, doctorCmd, upgradeCmd
- `agents`: commandsCmd, helpCmd, skillCmd, urlCmd
- `platform`: completionCmd, tuiCmd, watchCmd
- `resources`: tagsCmd (companion; generated parents get it from the generator)

- [ ] **Step 4: Generator emits GroupID**

In `emitGroup`'s parent dict (gen.go, post-Phase-2 shape):

```go
	parentDict := jen.Dict{
		jen.Id("Use"):     jen.Lit(group.Resource),
		jen.Id("GroupID"): jen.Lit("resources"),
	}
```

Regenerate and verify:

```bash
make generate
grep -n 'GroupID: "resources"' cmd/gen_customers.go | head -1
```

- [ ] **Step 5: Verify root help shape**

```bash
go build -o wenmar ./cmd/wenmar && ./wenmar --help
```

Expected: four group sections; leaf help (`./wenmar customers list --help`) shows `Flags:` (local) then `Global Flags:` with only `--output`, `--token`, `--base-url`, `--location`, `--debug`, `--allow-partial` (6 lines — down from 16).

- [ ] **Step 6: Run full suite, commit**

```bash
go test ./... -count=1
git add cmd/ cmd/gen_*.go
git commit -m "feat: cobra help groups; root Long lists quick modes and topics"
```

### Task 5: Examples on every command

**Files:**
- Modify: `cmd/gencli/main.go` (`Example` on CommandOverride), `cmd/gencli/gen.go` (emit Example)
- Modify: `cmd/gen_overrides.yaml` (example text per command)
- Modify: `cmd/tags.go`, `cmd/customers_extras.go`, `cmd/work_orders_extras.go` (Example on companion commands)
- Regenerate: `cmd/gen_*.go`
- Test: `cmd/examples_test.go`

- [ ] **Step 1: Generator plumbing**

`CommandOverride` (main.go) gains:

```go
	Example string `yaml:"example"`
```

`GenCommand` gains `Example string`; `buildCommand` wires `if ov.Example != "" { cmd.Example = ov.Example }`. `emitCommand` adds to the dict:

```go
	if cmd.Example != "" {
		dict[jen.Id("Example")] = jen.Lit(cmd.Example)
	}
```

- [ ] **Step 2: Failing test**

```go
package cmd

import (
	"strings"
	"testing"
)

func TestExamplesOnKeyCommands(t *testing.T) {
	for _, args := range [][]string{
		{"customers", "list", "--help"},
		{"customers", "show", "--help"},
		{"workorders", "list", "--help"},
		{"workorders", "show", "--help"},
		{"vehicles", "list", "--help"},
		{"tags", "list", "--help"},
		{"servicecategories", "list", "--help"},
		{"account", "show", "--help"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			out, err := execute(append(args, "--help")...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(out, "Examples:") {
				t.Errorf("no Examples section in help for %v", args)
			}
		})
	}
}
```

Wait — the `append(args, "--help")` double-adds --help for entries that already end with it. Fix: the table should NOT include `--help`; the loop appends it once:

```go
	for _, args := range [][]string{
		{"customers", "list"},
		{"customers", "show"},
		{"workorders", "list"},
		{"workorders", "show"},
		{"vehicles", "list"},
		{"tags", "list"},
		{"servicecategories", "list"},
		{"account", "show"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			out, err := execute(append(args, "--help")...)
			// ...
		})
	}
```

- [ ] **Step 3: Verify it fails**

```bash
go test ./cmd/ -run TestExamplesOnKeyCommands -v
```

Expected: FAIL — zero Example fields today.

- [ ] **Step 4: Write the examples in gen_overrides.yaml**

House style (spec §3.3): one read, one filtered, one `--agent` pipe per resource. Full block to merge into the `commands:` entries:

```yaml
  list_customers:
    resource: customers
    command: list
    summary: List all customers, paginated via the Link header
    method: ListCustomers
    query_param_struct: ListCustomersParams
    example: |
      wenmar customers list
      wenmar customers list --query "jane" --all
      wenmar customers list --output agent | head -20
  show_customer:
    resource: customers
    command: show
    summary: Show a single customer by ID
    method: ShowCustomer
    example: |
      wenmar customers show 42
  list_work_orders:
    resource: workorders
    command: list
    summary: List all work orders, paginated via the Link header
    method: ListWorkOrders
    example: |
      wenmar workorders list
      wenmar workorders list --jq '.[].number'
      wenmar workorders list --output count
  show_work_order_estimate:
    # (companion-owned post-Phase-2 — examples for show/tabs go in
    # work_orders_extras.go directly, not overrides)
  create_work_order:
    resource: workorders
    command: create
    summary: Create a new work order
    method: CreateWorkOrder
    request_struct: CreateWorkOrderRequest
    example: |
      wenmar workorders create --customer-id 42 --vehicle-id 5
  update_work_order:
    resource: workorders
    command: update
    summary: Update a work order by ID
    method: UpdateWorkOrder
    request_struct: UpdateWorkOrderRequest
    example: |
      wenmar workorders update 100 --intake-method drop_off
  delete_work_order:
    resource: workorders
    command: delete
    summary: Delete a work order by ID
    method: DeleteWorkOrder
    example: |
      wenmar workorders delete 100 --dry-run
  list_vehicles:
    resource: vehicles
    command: list
    summary: List all vehicles
    method: ListVehicles
    example: |
      wenmar vehicles list
      wenmar vehicles list --jq '.[].plate'
  show_vehicle:
    resource: vehicles
    command: show
    summary: Show a single vehicle by ID
    method: ShowVehicle
    example: |
      wenmar vehicles show 5
  decode_vin:
    resource: vehicles
    command: decode-vin
    summary: Decode a VIN into make/model
    method: DecodeVin
    positional_arg: string
    example: |
      wenmar vehicles decode-vin 1HGCM82633A004352
  list_customers_drivers:
    resource: drivers
    command: list
    summary: List drivers for a customer
    method: ListDrivers
    example: |
      wenmar drivers list --customer-id 42
  list_vendors:
    resource: vendors
    command: list
    summary: List all vendors
    method: ListVendors
    example: |
      wenmar vendors list
  show_vendor:
    resource: vendors
    command: show
    summary: Show a single vendor by ID
    method: ShowVendor
    example: |
      wenmar vendors show 7
  list_customers_statements:
    resource: statements
    command: list
    summary: List statements for a customer
    method: ListStatements
    example: |
      wenmar statements list --customer-id 42
  show_statement:
    resource: statements
    command: show
    summary: Show a single statement by ID
    method: ShowStatement
    example: |
      wenmar statements show 9001
  list_service_categories:
    resource: servicecategories
    command: list
    summary: List all service categories
    method: ListServiceCategories
    example: |
      wenmar servicecategories list
  show_location:
    resource: locations
    command: show
    summary: Show a location by ID
    method: ShowLocation
    example: |
      wenmar locations show main
  list_account:
    resource: account
    command: show
    summary: Show account details
    method: ListAccount
    example: |
      wenmar account show
  lookup_customer:
    resource: customers
    command: lookup
    summary: Search customers by name/email/phone
    method: LookupCustomer
    positional_arg: string
    example: |
      wenmar customers lookup "jane doe"
```

NOTE: the exact `summary:` values above must match what's already in the file — when merging, ONLY add the `example:` key to existing entries rather than replacing them (the block above shows context; the merge is additive). The `list_customers_drivers`/`list_customers_statements` summaries drop the stale "paginated via the Link header" claim IF Phase 2's honest-pagination pass didn't already update them — check the file state at implementation time.

- [ ] **Step 5: Companion examples**

In `cmd/work_orders_extras.go` add `Example` to each command var:

```go
var workOrdersShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single work order by ID",
	Example: `  wenmar workorders show 100
  wenmar workorders show 100 --output agent`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkOrdersShow,
}
```

(tabs get one example each: `wenmar workorders estimate 100`, etc.)

In `cmd/customers_extras.go`:

```go
var customersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new customer",
	Example: `  wenmar customers create --full-name "Jane Doe" --email "jane@test.com"`,
	RunE:  runCustomersCreate,
}

var customersUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a customer by ID",
	Example: `  wenmar customers update 42 --company-name "New Corp"`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCustomersUpdate,
}
```

In `cmd/tags.go`:

```go
var tagsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a customer or vehicle tag",
	Example: `  wenmar tags create --name "Fleet A"
  wenmar tags create --type vehicle --name "Priority"`,
	RunE: runTagsCreate,
}
```

(list/delete/rename similarly: one or two lines each.)

- [ ] **Step 6: Regenerate, run, commit**

```bash
make generate
go test ./cmd/ -run TestExamplesOnKeyCommands -v && go test ./... -count=1
git add cmd/ cmd/gen_*.go
git commit -m "feat: examples on every command via generator overrides and companions"
```

### Task 6: Help topics — output topic rewrite + migration table

**Files:**
- Modify: `cmd/help_topics.go` (rewrite "output" topic; verify "exit-codes" is the D5 source of truth)
- Test: covered by Task 7's sync test

- [ ] **Step 1: Replace the output topic content**

```go
	{
		name:  "output",
		title: "Output Modes",
		content: `The canonical flag is --output <mode>:

  table     Human-readable table (default on a terminal)
  md        GitHub-flavored markdown table
  json      Full JSON envelope {ok, data, summary, meta, breadcrumbs}
  agent     Raw JSON data (no envelope)
  quiet     Raw JSON output, no envelope (default when piped)
  ids-only  One ID per line (for shell loops)
  count     Bare integer count
  html      HTML document
  styled    Force the table even when piped

Quick flags (equivalents, hidden from subcommand help):
  --json          Same as --output json
  --agent         Same as --output agent (also makes --help emit JSON)
  --quiet         Same as --output quiet
  --jq <expr>     jq filter over the data (implies --output json)

Combining --output with a quick flag (or two quick flags together) is an
error — pick one.

Auto-switch: when stdout is not a TTY (e.g. piped to another command) and
no explicit mode is set, wenmar emits raw JSON (quiet) so output is
machine-readable. Use --output styled to force tables in a pipe.

Migration from pre-release flags:
  --md / -m / --markdown   -> --output md
  --ids-only               -> --output ids-only
  --count                  -> --output count
  --html                   -> --output html
  --styled                 -> --output styled

Envelope structure:
  {"ok": true, "data": [...], "summary": "5 customers",
   "meta": {"has_next": true}, "breadcrumbs": [{"action": "show", "cmd": "..."}]}`,
	},
```

- [ ] **Step 2: Verify**

```bash
go build -o wenmar ./cmd/wenmar && ./wenmar help output
./wenmar help output --agent   # JSON topic render still works
```

- [ ] **Step 3: Commit**

```bash
git add cmd/help_topics.go
git commit -m "docs(help): output topic rewritten for --output with migration table"
```

### Task 7: Exit-code sync test (D5 enforcement)

**Files:**
- Create: `cmd/exit_code_sync_test.go`

The 0-10 contract is rendered in three places plus code: `wenmar help exit-codes` (help_topics.go), README.md, skills/wenmar/SKILL.md, and `internal/errors` constants. This test keeps all four agreeing — a code added or reworded in one place fails CI.

- [ ] **Step 1: Write the test**

```go
package cmd

import (
	"os"
	"regexp"
	"sort"
	"strconv"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/errors"
)

// codeSet extracts the leading exit-code numbers from a table rendered as
// "N  meaning" lines (help topic) or "| N | meaning |" rows (markdown).
var (
	topicCodeRe  = regexp.MustCompile(`(?m)^\s{2}(\d{1,2})\s{2,}`)
	markdownCodeRe = regexp.MustCompile(`(?m)^\|\s*(\d{1,2})\s*\|`)
)

func codeSet(re *regexp.Regexp, text string) []int {
	seen := map[int]bool{}
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		n, _ := strconv.Atoi(m[1])
		if n >= 0 && n <= 10 {
			seen[n] = true
		}
	}
	out := make([]int, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

func TestExitCodeTableSync(t *testing.T) {
	want := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// 1. The help topic lists all 11 codes.
	var topic string
	for _, ht := range helpTopics {
		if ht.name == "exit-codes" {
			topic = ht.content
		}
	}
	if topic == "" {
		t.Fatal("exit-codes topic missing from helpTopics")
	}
	if got := codeSet(topicCodeRe, topic); !equalInts(got, want) {
		t.Errorf("help exit-codes topic lists %v, want %v", got, want)
	}

	// 2. The internal/errors constants match the contract values.
	consts := map[string]int{
		"success": errors.ExitSuccess, "generic": errors.ExitGeneric,
		"auth": errors.ExitAuth, "notfound": errors.ExitNotFound,
		"validation": errors.ExitValidation, "ratelimit": errors.ExitRateLimit,
		"server": errors.ExitServer, "conflict": errors.ExitConflict,
		"forbidden": errors.ExitForbidden, "partial": errors.ExitPartial,
		"offline": errors.ExitOffline,
	}
	for name, v := range map[string]int{
		"success": 0, "generic": 1, "auth": 2, "notfound": 3, "validation": 4,
		"ratelimit": 5, "server": 6, "conflict": 7, "forbidden": 8,
		"partial": 9, "offline": 10,
	} {
		if consts[name] != v {
			t.Errorf("errors.Exit%s = %d, want %d", name, consts[name], v)
		}
	}

	// 3. README documents the same set.
	readme, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	if got := codeSet(markdownCodeRe, string(readme)); !equalInts(got, want) {
		t.Errorf("README exit-code table lists %v, want %v", got, want)
	}

	// 4. SKILL.md documents the same set.
	skill, err := os.ReadFile("../skills/wenmar/SKILL.md")
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	if got := codeSet(markdownCodeRe, string(skill)); !equalInts(got, want) {
		t.Errorf("SKILL.md exit-code table lists %v, want %v", got, want)
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
```

Note: the `topicCodeRe` must match the topic's actual rendering — the exit-codes topic uses two-space indented lines like `  0  success` (help_topics.go:45-57). Run once; if the regex misses, adjust to `(?m)^\s{2}(\d{1,2})\s` and re-verify by printing the matched set. The markdown regex matches the README's `| 0 | Success |` rows; SKILL.md's table (post-Phase-1 fix) uses the same shape.

- [ ] **Step 2: Run it**

```bash
go test ./cmd/ -run TestExitCodeTableSync -v
```

Expected: PASS if Phase 1 Task 18 completed the SKILL.md table; FAIL with a precise diff otherwise (fix the doc, not the test).

- [ ] **Step 3: Commit**

```bash
git add cmd/exit_code_sync_test.go
git commit -m "test: exit-code contract stays in sync across topic, README, SKILL.md, and code"
```

### Task 8: README + SKILL.md flag migration

**Files:**
- Modify: `README.md`
- Modify: `skills/wenmar/SKILL.md`

(Full SKILL.md rewrite is Phase 4; this task only migrates the flag surface so docs match the binary.)

- [ ] **Step 1: README**

Replace the output-modes table (README.md:108-122) with:

```markdown
## Output modes

The canonical flag is `--output <mode>`:

| Command | Description |
|---------|-------------|
| (default) | Human-readable table — or raw JSON when piped |
| `--output table` | Human-readable table (explicit) |
| `--output md` | GFM table |
| `--output json` | Full JSON envelope `{ok, data, summary, meta}` |
| `--output agent` | Raw JSON data (no envelope) |
| `--output quiet` | Raw JSON output, no envelope |
| `--output ids-only` | One ID per line (for shell loops) |
| `--output count` | Bare integer count |
| `--output html` | HTML document |
| `--output styled` | Force human tables even when piped |

Quick flags (still work, hidden from subcommand help): `--json`, `--agent`,
`--quiet`, `--jq 'filter'` (implies json). Combining quick flags with
`--output` is an error.

When stdout is not a TTY and no explicit mode is set, wenmar emits raw
JSON so piped output is machine-readable. Use `--output styled` to force
tables in a pipe.

### Migration from earlier pre-release flags

| Old | New |
|-----|-----|
| `--md` / `-m` / `--markdown` | `--output md` |
| `--ids-only` | `--output ids-only` |
| `--count` | `--output count` |
| `--html` | `--output html` |
| `--styled` | `--output styled` |

Run `wenmar help output` for the full reference.
```

Sweep the Usage examples: `wenmar customers list --json` stays valid (sugar); `--jq` stays; any `--md`/`--agent`-pipe examples change to `--output` form. Update the "Agent discovery" section: `wenmar customers list --help --agent` still works (sugar).

- [ ] **Step 2: SKILL.md**

Update every flag reference (verified lines 38, 60-62, 117-124, 134-144): `--md` → `--output md`, `--ids-only` → `--output ids-only`, `--count` → `--output count`. Add the conflict rule to the agent invariants: "Never combine output flags; `--output` alone or one quick flag." Keep `--json`/`--agent`/`--jq` references — they're sugar and still work.

- [ ] **Step 3: Verify against the binary**

```bash
go build -o wenmar ./cmd/wenmar
./wenmar help output
./wenmar customers list --output md
./wenmar customers list --jq '.[].full_name'
```

Every documented command in README/SKILL.md must run (grep the docs for `wenmar ` invocations and spot-check).

- [ ] **Step 4: Commit**

```bash
git add README.md skills/wenmar/SKILL.md
git commit -m "docs: --output migration in README and SKILL.md"
```

### Task 9: Surface snapshot + final verification

**Files:**
- Regenerate: `surface-snapshot.json`
- Modify: none (verification only)

- [ ] **Step 1: Regenerate the snapshot**

```bash
go run ./cmd/wenmar surface-snapshot > surface-snapshot.json
git diff surface-snapshot.json | head -80
```

Expected diff: dropped flags gone (`--md`, `--markdown`, `--ids-only`, `--count`, `--html`, `--styled`), `--output` present, sugar flags hidden (surface snapshot skips `f.Hidden` — surface_snapshot.go:61-63 — so `--json`/`--agent`/`--quiet`/`--jq` DISAPPEAR from the snapshot even though they still work; this is correct: the snapshot records the *documented* surface).

- [ ] **Step 2: Full gate**

```bash
go vet ./... && gofmt -l cmd/ internal/ && go test -race ./... -count=1 && make regen-drift
```

Expected: all green. `gofmt -l` prints nothing.

- [ ] **Step 3: Human smoke pass**

```bash
go build -o wenmar ./cmd/wenmar
./wenmar                     # no token: welcome + setup pointer
./wenmar --help              # groups, fits roughly one screen
./wenmar customers --help    # Resources section, local flags, 6 globals
./wenmar help                # topics list
./wenmar help output         # new topic
./wenmar bogus               # error + suggestion (Phase 1)
```

- [ ] **Step 4: Commit**

```bash
git add surface-snapshot.json
git commit -m "chore: surface snapshot for the --output interface"
```

---

## Self-review notes

- **Spec coverage (§Phase 3):** 3.1 `--output` canonical + sugar + conflicts + ResolveMode signature cleanup → Tasks 1-3 (spec's `ResolveMode(out, jq, sugar)` intent implemented as `ParseMode(ModeSpec)` — same shape, struct not positional). 3.2 help groups + leaf-flag collapse + root Long/footer + suggestions → Task 4 (suggestions landed in Phase 1 Task 7; root Long carries the topics footer since cobra help templates are per-command and a custom template was judged more fragile than the Long). 3.3 examples everywhere → Task 5 (generator + companions; house style applied). 3.4 exit-code enforcement → Task 7. Deviation from spec, documented: sugar flags are hidden at ALL levels (cobra can't show-at-root/hide-at-leaf); root Long lists them for discovery — same effect, no template surgery.
- **Deliberate non-goals:** no custom help templates; no `--output` shell completion of mode names (flag completion is Phase 4 polish if wanted); doctor's custom JSON envelope stays custom (mode-check only).
- **Type consistency:** `ModeSpec`/`ParseMode` defined Task 1, consumed Task 2 (`modeSpec()`, `resolveMode()`), tested Task 3. Group IDs defined Task 4 root, consumed by generator + companions in the same task. The fake-API request counter (`fakeAPIRequests`) added in Task 3 Step 2 is used by the conflict fail-fast test only.
- **Risk notes:** (1) Task 2's resolveMode signature change touches ~13 call sites — mechanical, but the build must be green before Task 3's test edits begin. (2) The `topicCodeRe` regex in Task 7 is the one assertion sensitive to exact whitespace in help_topics.go — the test prints the matched set on failure, making a mismatch self-diagnosing. (3) Hidden sugar flags vanish from `surface-snapshot.json` — this is intended (documented surface) but the commit message in Task 9 says so explicitly to avoid a reviewer double-take.
- **Ordering within the phase:** Tasks 1→2→3 are one chain (build must stay green between commits); Task 4 and 5 both regenerate, so land Task 4 first (groups) then Task 5 (examples) to keep regen diffs separated; Tasks 6-8 are doc-only; Task 9 last.