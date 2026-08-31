# Basecamp-Pure Output Flags — Kill `--output` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-ruby:subagent-driven-development (recommended) or superpowers-ruby:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `--output <mode>` umbrella with five self-describing boolean flags (`--json`, `--agent`, `--jq`, `--ids-only`, `--styled`), delete the redundant modes (`md`, `quiet`, `count`, `html`), error on conflicting flag combinations instead of resolving precedence, and update the help topic, generated examples, README, and SKILL.md to match.

**Architecture:** `output.ParseMode` loses the `Output string` field and the `modeNames` map — it becomes validation-over-booleans: count the set flags, error listing them if more than one, else resolve to one of five Modes (`ModeDefault`, `ModeJSON`, `ModeAgent`, `ModeJQ`, `ModeIDsOnly`). The pipe auto-switch targets `ModeAgent` directly (the `ModeQuiet` constant dies — same renderer, one fewer concept). Renderers `count.go` and `html.go` are deleted along with their modes; counts become `--jq 'length'`. All five flags are visible at every help level — the hidden-sugar mechanism is gone. This supersedes design decision D3 in `docs/superpowers/specs/2026-08-30-production-readiness-refactor-design.md` (user-approved revision: the 9-mode problem was fake; `--output` was a premature abstraction).

**Tech Stack:** Go 1.27, cobra/pflag, committed generated commands (`cmd/gen_*.go` canonical), wenmar-sdk 0.4.1.

**Supersedes:** Tasks 1–3, 6, 8 of `docs/superpowers/plans/2026-08-30-production-readiness-phase-3.md` (those landed as commits `386613c`..`65fb45c` and are now reworked). Phase 3 Tasks 4 (groups), 5 (examples), 7 (exit-code sync), 9 (conformance gate) are untouched and keep passing.

**Conventions:** Run `go build ./... && go test ./...` before every commit. All commands from repo root. Commit style: `feat:`, `fix:`, `test:`, `docs:`, `chore:`.

---

## Verified ground truth (current state on `main` — do not re-derive)

- **Mode system** (`internal/output/output.go`): 9 constants (`ModeDefault, ModeMD, ModeJSON, ModeAgent, ModeJQ, ModeQuiet, ModeIDsOnly, ModeCount, ModeHTML`, lines 13-23). `modeNames` maps 9 `--output` strings (55-65). `ModeSpec{Output, JSON, Agent, Quiet, JQ}` (45-51) — **`IDsOnly` and `Styled` are reachable ONLY via `--output` today**; the boolean flags were dropped in the Phase 3 cutover. `ParseMode` (72-109): `--output`+sugar conflicts, then sugar precedence jq > quiet > agent > json, then pipe auto-switch → `ModeQuiet`.
- **Renderers** (`Render`, output.go:115-134): `ModeMD, ModeDefault → renderMarkdown` (md IS the default renderer); `ModeAgent, ModeQuiet → renderJSONRaw` (identical); `ModeCount → renderCount` (count.go); `ModeHTML → renderHTML` (html.go). `--jq 'length'` is a drop-in for count: `renderJQ` (jq.go:41-46) encodes a single scalar result bare — `--jq 'length'` on the extracted collection emits `2`, exactly what `renderCount` emits (asserted by TestCustomersList_Count, integration_test.go:705-716).
- **Flags** (`cmd/root.go`): `outputFlag` registered at :101 with the 9-mode help string; sugar `--json`/`--agent`/`--quiet`/`--jq` at :102-105, all `MarkHidden` (:112-115); `--config-path` hidden (:116). `PersistentPreRunE` (:120-123) validates via `output.ParseMode(modeSpec())`. Vars at :15-27 include `outputFlag`, `quietFlag`.
- **Consumers**: `cmd/runners.go:13-28` (`modeSpec()` reads `outputFlag`; `resolveMode()`); `cmd/url.go:34`, `cmd/doctor.go:141` (both call `resolveMode()`, no direct flag reads). `cmd/runners.go:155,208` check `mode == output.ModeIDsOnly || mode == output.ModeCount` for pagination notices. Root Long's output block at root.go:50-54.
- **`--agent` help hijack** (root.go:142-154): `SetHelpFunc` emits `agent.BuildCommandInfo` JSON when `agentFlag` is set. **This is wenmar's deliberate agent-discovery surface** (Phase 1 Task 10, documented in the `agent-help` help topic and SKILL.md) — it stays. Divergence from basecamp noted: basecamp's `--agent` doesn't hijack help because basecamp has no structured-help feature; wenmar does.
- **Tests referencing the old surface**: `cmd/integration_test.go` — `execute()` resets `outputFlag…quietFlag` (:49-50); `TestCustomersList_Markdown` uses `--output md` (:461-471, asserts GFM header `| id |`); `TestCustomersList_Count` uses `--output count` (:705-716, asserts `"2"`); `TestDroppedOutputFlagsRemoved` drops list `{"--md","-m","--markdown","--ids-only","--count","--html","--styled"}` (:721-731); `TestOutputModeFlag` uses `--output md` and `--output ids-only` (:733-755); `TestOutputModeConflictFailsFast` uses `--json --output md` + asserts zero API calls via `srvRequestCount()` (:757-769). `internal/output/output_test.go` `TestParseMode` has `Output: "table"/"md"/…` cases (:141-160) and `TestParseModeAutoSwitch`.
- **Generated examples** (`cmd/gen_overrides.yaml`): line 174 `wenmar customers list --output agent | head -20`; line 304 `wenmar workorders list --output count`. These flow into committed `cmd/gen_*.go` via `make generate`.
- **Docs**: `cmd/help_topics.go:20-55` — the `output` topic is `--output`-canonical with a migration table; the `agent-help` topic says "Global output flags are omitted from leaf help" (stale once flags are visible). `README.md:108-145` — `--output` table + migration table. `skills/wenmar/SKILL.md` — `--output` references at :38-39, :58-63, output section at :114+, `--output agent`/`--output count` in examples. Root help footer lists topics (root.go:56-63).
- **Surface snapshot**: `surface-snapshot.json` reflects the `--output` interface (regenerated at commit `65fb45c`); hidden flags are excluded from the snapshot by `buildSurfaceSnapshot` (surface_snapshot.go skips `f.Hidden`), so today's snapshot shows ONLY `--output` among mode flags — after this plan it shows all five booleans.
- **What does NOT change**: `--token`, `--base-url`, `--location`, `--allow-partial`, `--debug` (visible); `--config-path` (stays hidden, "for testing"); the pipe auto-switch itself (piped + no flags → raw JSON); exit codes; groups; examples tests; exit-code sync test; freshness gates from Phase 4 (not yet landed — this plan must land BEFORE the Phase 4 freshness test so the test locks the final surface).

**Why `--md` dies (get the reason right in every doc):** `--md` and the default share `renderMarkdown` — `--md` was always redundant with the default renderer; its only utility was overriding the pipe auto-switch, which is exactly what `--styled` does. It is NOT redundant with `--agent` (raw JSON — a different renderer entirely).

**Why `--count` dies:** `--jq 'length'` emits the identical bare integer through the existing jq path — verified against `renderJQ`'s single-result encoding. `--ids-only` stays because `--jq '.[].id'` emits a JSON array, not one-ID-per-line, which breaks `xargs` workflows.

---

## Task 1: `output.ParseMode` goes boolean — modes, map, renderers

**Files:**
- Modify: `internal/output/output.go` (ModeSpec, ParseMode, modeNames deletion, Render, Mode constants)
- Delete: `internal/output/count.go`, `internal/output/count_test.go`, `internal/output/html.go`, `internal/output/html_test.go`
- Modify: `internal/output/output_test.go` (TestParseMode rewrite; TestParseModeAutoSwitch target change)

- [ ] **Step 1: Rewrite the failing tests**

Replace `TestParseMode` in `internal/output/output_test.go` (keep the file's other tests — render tests etc.):

```go
func TestParseMode(t *testing.T) {
	cases := []struct {
		name    string
		spec    ModeSpec
		want    Mode
		wantErr bool
	}{
		{name: "nothing set resolves default", spec: ModeSpec{}, want: ModeDefault},
		{name: "json", spec: ModeSpec{JSON: true}, want: ModeJSON},
		{name: "agent", spec: ModeSpec{Agent: true}, want: ModeAgent},
		{name: "jq", spec: ModeSpec{JQ: ".[]"}, want: ModeJQ},
		{name: "ids-only", spec: ModeSpec{IDsOnly: true}, want: ModeIDsOnly},
		{name: "styled forces default", spec: ModeSpec{Styled: true}, want: ModeDefault},
		{name: "json plus agent conflicts", spec: ModeSpec{JSON: true, Agent: true}, wantErr: true},
		{name: "json plus jq conflicts", spec: ModeSpec{JSON: true, JQ: ".[]"}, wantErr: true},
		{name: "agent plus ids-only conflicts", spec: ModeSpec{Agent: true, IDsOnly: true}, wantErr: true},
		{name: "styled plus json conflicts", spec: ModeSpec{Styled: true, JSON: true}, wantErr: true},
		{name: "styled plus ids-only conflicts", spec: ModeSpec{Styled: true, IDsOnly: true}, wantErr: true},
		{name: "jq plus ids-only conflicts", spec: ModeSpec{JQ: ".[]", IDsOnly: true}, wantErr: true},
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
```

Update `TestParseModeAutoSwitch`: the piped auto-switch now targets `ModeAgent` (same raw-JSON renderer the old `ModeQuiet` used):

```go
	// With nothing set and piped stdout, raw JSON is chosen.
	got, err = ParseMode(ModeSpec{})
	if err != nil {
		t.Fatal(err)
	}
	if got != ModeAgent {
		t.Errorf("piped stdout with no explicit mode should auto-switch to raw JSON (ModeAgent), got %v", got)
	}
```

Delete any `output_test.go` cases that fed `Output: "..."` strings (they compile against a field that's about to disappear).

- [ ] **Step 2: Verify it fails**

```bash
go test ./internal/output/ -run TestParseMode -v
```

Expected: FAIL — `ModeSpec.IDsOnly`/`Styled` undefined; conflict cases don't error as specced.

- [ ] **Step 3: Implement the boolean resolver**

In `internal/output/output.go`, replace the Mode constants block (lines 13-23), `ModeSpec`, `modeNames`, and `ParseMode` (lines 43-109) with:

```go
const (
	ModeDefault Mode = iota // human table (also forced by --styled when piped)
	ModeJSON                // --json: full envelope {ok, data, summary, meta, breadcrumbs}
	ModeAgent               // --agent: raw JSON, no envelope (also the piped default)
	ModeJQ                  // --jq <expr>
	ModeIDsOnly             // --ids-only: one ID per line
)

// ModeSpec carries the output-mode flags from the command line. All five
// are peers — there is no umbrella flag. More than one set is an error.
type ModeSpec struct {
	JSON    bool
	Agent   bool
	JQ      string
	IDsOnly bool
	Styled  bool
}

// ParseMode resolves the output mode from ModeSpec. Exactly one mode flag
// may be set — combining them errors with the offending flag names rather
// than silently picking a winner. With nothing set: table on a terminal,
// raw JSON when piped.
func ParseMode(spec ModeSpec) (Mode, error) {
	active := []string{}
	if spec.JSON {
		active = append(active, "--json")
	}
	if spec.Agent {
		active = append(active, "--agent")
	}
	if spec.JQ != "" {
		active = append(active, "--jq")
	}
	if spec.IDsOnly {
		active = append(active, "--ids-only")
	}
	if spec.Styled {
		active = append(active, "--styled")
	}

	if len(active) > 1 {
		return 0, fmt.Errorf("conflicting output flags: %s (use only one)", strings.Join(active, ", "))
	}

	switch {
	case spec.JQ != "":
		return ModeJQ, nil
	case spec.IDsOnly:
		return ModeIDsOnly, nil
	case spec.Styled:
		return ModeDefault, nil // forces the human table even when piped
	case spec.Agent:
		return ModeAgent, nil
	case spec.JSON:
		return ModeJSON, nil
	}

	// Auto-switch: piped stdout gets machine-readable raw JSON.
	if !isTerminal(os.Stdout) {
		return ModeAgent, nil
	}
	return ModeDefault, nil
}
```

Add `"strings"` to the imports. Update `Render` (lines 115-134) — remove the dead arms:

```go
func Render(w io.Writer, data any, summary string, meta *Meta, opts Options) error {
	switch opts.Mode {
	case ModeDefault:
		return renderMarkdown(w, data, summary)
	case ModeJSON:
		return renderJSON(w, data, summary, meta, opts.Breadcrumbs, opts.Notice)
	case ModeAgent:
		return renderJSONRaw(w, data)
	case ModeJQ:
		return renderJQ(w, data, opts.JQFilter)
	case ModeIDsOnly:
		return renderIDsOnly(w, data)
	default:
		return fmt.Errorf("unknown output mode")
	}
}
```

Delete `count.go`, `count_test.go`, `html.go`, `html_test.go`:

```bash
git rm internal/output/count.go internal/output/count_test.go internal/output/html.go internal/output/html_test.go
```

- [ ] **Step 4: Run the package**

```bash
go test ./internal/output/ -v
```

Expected: all pass. (`go build ./...` still fails here — `cmd` references `outputFlag`/`ModeCount`; fixed in Task 2. That's expected mid-chain; do not commit yet if the repo must stay green per-commit — in that case fold Task 1+2 into one commit.)

- [ ] **Step 5: Commit**

```bash
git add internal/output/
git commit -m "feat(output): five boolean mode flags; conflicts error; count/html modes removed"
```

### Task 2: Root flags, modeSpec, and consumers

**Files:**
- Modify: `cmd/root.go` (flag registration, var block, Long, PreRunE)
- Modify: `cmd/runners.go` (modeSpec, pagination-notice checks)
- Modify: `cmd/help_topics.go` (agent-help topic stale line) — the output topic is Task 4

- [ ] **Step 1: Root var block (root.go:15-27)**

Delete `outputFlag`; add `idsOnlyFlag`, `styledFlag`:

```go
var (
	tokenFlag      string
	baseURLFlag    string
	locationFlag   string
	jsonFlag       bool
	agentFlag      bool
	jqFlag         string
	idsOnlyFlag    bool
	styledFlag     bool
	allowPartial   bool
	configPathFlag string
	debugFlag      bool
)
```

(`quietFlag` is deleted — no flag binds to it anymore.)

- [ ] **Step 2: Flag registration (root.go:101-116)**

Replace the `--output` + hidden-sugar block:

```go
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output as full JSON envelope {ok, data, summary, meta}")
	rootCmd.PersistentFlags().BoolVar(&agentFlag, "agent", false, "Output raw JSON data, no envelope (with --help: structured JSON)")
	rootCmd.PersistentFlags().StringVar(&jqFlag, "jq", "", "Filter output with a jq expression")
	rootCmd.PersistentFlags().BoolVar(&idsOnlyFlag, "ids-only", false, "Print one ID per line (for shell loops)")
	rootCmd.PersistentFlags().BoolVar(&styledFlag, "styled", false, "Force the human table even when piped")
```

Delete the four `MarkHidden` calls for json/agent/quiet/jq (keep `--config-path` hidden). `PersistentPreRunE` stays exactly as is — it already validates via `output.ParseMode(modeSpec())`, and ParseMode now rejects conflicts among the booleans.

- [ ] **Step 3: modeSpec (runners.go:13-21)**

```go
// modeSpec snapshots the output-mode flags for ParseMode.
func modeSpec() output.ModeSpec {
	return output.ModeSpec{
		JSON:    jsonFlag,
		Agent:   agentFlag,
		JQ:      jqFlag,
		IDsOnly: idsOnlyFlag,
		Styled:  styledFlag,
	}
}
```

- [ ] **Step 4: Pagination-notice checks (runners.go:155, 208)**

Both sites read `if mode == output.ModeIDsOnly || mode == output.ModeCount {` — `ModeCount` is gone:

```go
	if mode == output.ModeIDsOnly {
```

- [ ] **Step 5: Root Long (root.go:50-54)**

Replace the `Output:` block:

```
Output:
  --json              Full JSON envelope {ok, data, summary, meta}
  --agent             Raw JSON (no envelope); --agent --help emits JSON
  --jq <expr>         Filter output with a jq expression
  --ids-only          One ID per line; --styled forces tables when piped
  (default: table on a terminal, raw JSON when piped)
```

- [ ] **Step 6: Fix the stale agent-help topic line**

In `cmd/help_topics.go`, the `agent-help` topic's flags field doc says "Global output flags are omitted from leaf help" — now false (flags are visible). Change that line to:

```
  flags       Flags [{name, short, type, required, default, description}]
```

(delete the omission sentence entirely — nothing is omitted anymore).

- [ ] **Step 7: Build and manual verify**

```bash
go build -o wenmar ./cmd/wenmar
./wenmar customers list --json --agent; echo "exit=$?"
# ERROR: conflicting output flags: --json, --agent (use only one)   exit=1
./wenmar --help | grep -E "json|agent|jq|ids-only|styled"
# all five visible under Global Flags
./wenmar customers list --help | grep -c "output"
# 0 — --output is gone
```

- [ ] **Step 8: Commit**

```bash
go build ./... && go test ./internal/... && go test ./cmd/ 2>&1 | tail -5
```

(cmd tests will fail on the old flag invocations — Task 3 fixes them. If the repo must be green per-commit, land Tasks 2+3 as one commit.)

```bash
git add cmd/root.go cmd/runners.go cmd/help_topics.go
git commit -m "feat: basecamp-pure output flags — --output removed, five visible booleans, conflicts error"
```

### Task 3: Integration-test migration

**Files:**
- Modify: `cmd/integration_test.go`

- [ ] **Step 1: execute() reset (lines 49-50)**

```go
	jsonFlag, agentFlag, jqFlag, idsOnlyFlag, styledFlag = false, false, "", false, false
```

(delete `outputFlag`/`quietFlag` resets; keep the `resetHelpFlag(rootCmd)` call and auth resets that follow.)

- [ ] **Step 2: Migrate each flagged test**

`TestCustomersList_Markdown` (:461-471) — was `--output md`, asserting the GFM header through a pipe; the pipe-override flag is now `--styled`:

```go
func TestCustomersList_Styled(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	defer srv.Close()
	out, err := execute(
		"customers", "list", "--styled",
		"--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "| id |") {
		t.Errorf("expected GFM table header, got: %s", out)
	}
}
```

`TestCustomersList_Count` (:705-716) — `--jq 'length'` is the replacement; the assertion (`"2"`) is unchanged because `renderJQ` encodes the single scalar bare:

```go
func TestCustomersList_Count(t *testing.T) {
	srv := startFakeAPI(t, "secret-token")
	out, err := execute(
		"customers", "list", "--jq", "length",
		"--base-url", srv.URL, "--token", "secret-token",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	trimmed := strings.TrimSpace(out)
	if trimmed != "2" {
		t.Errorf("expected count '2', got %q", trimmed)
	}
}
```

`TestDroppedOutputFlagsRemoved` (:721-731) — the dropped list grows `--output` and `--quiet`; `--ids-only`/`--styled` move OUT of the dropped list (they're back):

```go
func TestDroppedOutputFlagsRemoved(t *testing.T) {
	dropped := []string{"--output", "--md", "-m", "--markdown", "--quiet", "--count", "--html"}
	for _, flag := range dropped {
		t.Run(flag, func(t *testing.T) {
			args := []string{"customers", "list", flag}
			// --output takes a value; give it one so the error is the
			// unknown-flag error, not a missing-argument error.
			if flag == "--output" {
				args = append(args, "md")
			}
			_, err := execute(args...)
			if err == nil {
				t.Errorf("%s was dropped; it must error", flag)
			}
		})
	}
}
```

`TestOutputModeFlag` (:733-755) — becomes a positive test of the five booleans:

```go
func TestOutputFlags(t *testing.T) {
	srv := startFakeAPI(t, "tok-out")
	defer srv.Close()
	t.Setenv("WENMAR_URL", srv.URL)
	t.Setenv("WENMAR_TOKEN", "tok-out")

	out, err := execute("customers", "list", "--styled")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "| id |") {
		t.Errorf("--styled should render the human table, got:\n%s", out)
	}

	out, err = execute("customers", "list", "--ids-only")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.FieldsFunc(out, func(r rune) bool { return r == '\n' })
	if len(lines) != 2 || lines[0] != "1" || lines[1] != "2" {
		t.Errorf("--ids-only should print one ID per line, got %q", out)
	}
}
```

`TestOutputModeConflictFailsFast` (:757-769) — the conflict is now two booleans; the zero-API-calls assertion is the heart of the test and stays:

```go
func TestOutputFlagConflictFailsFast(t *testing.T) {
	srv := startFakeAPI(t, "tok-conflict")
	defer srv.Close()
	t.Setenv("WENMAR_URL", srv.URL)

	_, err := execute("customers", "list", "--json", "--agent")
	if err == nil {
		t.Fatal("conflicting output flags must error")
	}
	if n := srvRequestCount(); n != 0 {
		t.Errorf("conflict validation should run before any API call; saw %d requests", n)
	}
}
```

- [ ] **Step 3: Sweep for stragglers**

```bash
grep -rn "outputFlag\|quietFlag\|--output\|ModeQuiet\|ModeCount\|ModeHTML\|ModeMD" cmd/ internal/ --include="*.go"
```

Expected: zero hits outside this plan's docs. Fix any remaining test/helper references the same way.

- [ ] **Step 4: Run everything**

```bash
go test ./... -count=1
```

Expected: green.

- [ ] **Step 5: Commit**

```bash
git add cmd/
git commit -m "test: migrate to boolean output flags; --output/--quiet/--count/--html removed"
```

### Task 4: Help output topic rewrite

**Files:**
- Modify: `cmd/help_topics.go` (the `output` topic, lines 20-55)

- [ ] **Step 1: Replace the topic content**

```go
	{
		name:  "output",
		title: "Output Modes",
		content: `One flag, one mode — combining them is an error:

  --json           Full JSON envelope {ok, data, summary, meta, breadcrumbs}
  --agent          Raw JSON data, no envelope
  --jq <expr>      Filter output with a jq expression (e.g. --jq 'length'
                   for a bare count, --jq '.[].full_name' for one field)
  --ids-only       One ID per line (for shell loops: | xargs)
  --styled         Force the human table even when piped

Default: a human-readable table on a terminal; raw JSON (--agent shape)
when piped, so piped output is always machine-readable. --styled overrides
the pipe default.

Counts: there is no --count flag; use --jq 'length'.
GFM tables: there is no --md flag; the default table IS markdown. Use
--styled to force it when piped.

Migration from the pre-release --output interface:
  --output json     -> --json
  --output agent    -> --agent
  --output quiet    -> --agent (or nothing: piping already does this)
  --output md       -> default renderer, or --styled when piped
  --output table    -> default renderer
  --output styled   -> --styled
  --output ids-only -> --ids-only
  --output count    -> --jq 'length'
  --output html     -> (removed)

Envelope structure:
  {"ok": true, "data": [...], "summary": "5 customers",
   "meta": {"has_next": true}, "breadcrumbs": [{"action": "show", "cmd": "..."}]}`,
	},
```

- [ ] **Step 2: Verify**

```bash
go build -o wenmar ./cmd/wenmar && ./wenmar help output
./wenmar help output --agent | head -5   # JSON topic render still works
```

- [ ] **Step 3: Commit**

```bash
git add cmd/help_topics.go
git commit -m "docs(help): output topic rewritten for boolean flags"
```

### Task 5: Generated examples + regenerate

**Files:**
- Modify: `cmd/gen_overrides.yaml` (lines 174, 304)
- Regenerate: `cmd/gen_*.go`

- [ ] **Step 1: Fix the two example strings**

Line 174:

```yaml
      wenmar customers list --agent | head -20
```

Line 304:

```yaml
      wenmar workorders list --jq 'length'
```

- [ ] **Step 2: Regenerate and verify**

```bash
make generate
git diff cmd/gen_*.go | head -20    # only the two example strings change
```

If the diff shows anything else, STOP — the regen-drift gate means generated output must be byte-stable; investigate before proceeding.

- [ ] **Step 3: Run the examples + drift gates**

```bash
go test ./cmd/ -run TestExamplesOnKeyCommands -v
make regen-drift   # or the in-process equivalent from the Phase 2 rework
go test ./... -count=1
```

- [ ] **Step 4: Commit**

```bash
git add cmd/gen_overrides.yaml cmd/gen_*.go
git commit -m "docs(gen): examples use the boolean output flags"
```

### Task 6: README + SKILL.md

**Files:**
- Modify: `README.md` (output section, lines 108-145)
- Modify: `skills/wenmar/SKILL.md` (preflight :38-39, invariants :55-63, output section :114+, examples throughout)

- [ ] **Step 1: README output section**

```markdown
## Output modes

One flag, one mode — combining them is an error:

| Flag | Description |
|------|-------------|
| (default) | Human-readable table — or raw JSON when piped |
| `--json` | Full JSON envelope `{ok, data, summary, meta}` |
| `--agent` | Raw JSON data (no envelope) |
| `--jq 'expr'` | Filter output with a jq expression (`--jq 'length'` for counts) |
| `--ids-only` | One ID per line (for shell loops) |
| `--styled` | Force human tables even when piped |

When stdout is not a TTY and no flag is set, wenmar emits raw JSON so piped
output is machine-readable. Use `--styled` to force tables in a pipe.

### Migration from the pre-release `--output` interface

| Old | New |
|-----|-----|
| `--output json` / `--output agent` | `--json` / `--agent` |
| `--output quiet` | `--agent` (or nothing — piping already does it) |
| `--output md` / `--output table` | default renderer; `--styled` when piped |
| `--output ids-only` | `--ids-only` |
| `--output count` | `--jq 'length'` |
| `--output html` | (removed) |

Run `wenmar help output` for the full reference.
```

- [ ] **Step 2: SKILL.md changes**

Preflight (line 38-39):

```markdown
3. Set output mode explicitly (`--json`, `--jq`, `--agent` — pick one;
   combining them is an error)
```

Invariants #6 (lines 55-63):

```markdown
6. **Choose the right output mode — exactly one flag:**
   - `--jq 'expr'` to filter/extract specific fields
   - `--json` for full envelope `{ok, data, summary, meta}`
   - `--agent` for headless agent workflows (raw JSON, no envelope)
   - `--jq 'length'` for bare counts, `--ids-only` for shell loops
   - `--styled` to force human tables when piped
   - Combining any of these is an error — never pass two.
```

Output modes section (line ~114) — replace with a table matching the README's (same rows), plus the pipe-default sentence. Examples sweep — replace every `--output agent` → `--agent`, `--output md` → `--styled` (in pipe contexts) or drop (TTY contexts where the default renders anyway), `--output count` → `--jq 'length'`, `--output ids-only` → `--ids-only`. Verify the sweep:

```bash
grep -n "\-\-output" skills/wenmar/SKILL.md README.md
```

Expected: zero hits after the sweep.

- [ ] **Step 3: Verify every documented invocation against the binary**

```bash
go build -o wenmar ./cmd/wenmar
./wenmar customers list --jq 'length'
./wenmar customers list --ids-only
./wenmar customers list --styled
./wenmar workorders list --jq 'length'
./wenmar customers list --agent | head -3
```

All five must run (against a token-less fake state they'll error with exit 2 auth — that's fine; the flag parsing must not error with "unknown flag").

- [ ] **Step 4: Commit**

```bash
git add README.md skills/wenmar/SKILL.md
git commit -m "docs: README and SKILL.md on the boolean output flags"
```

### Task 7: Surface snapshot + final gate

**Files:**
- Regenerate: `surface-snapshot.json`

- [ ] **Step 1: Regenerate the snapshot**

```bash
go run ./cmd/wenmar surface-snapshot > surface-snapshot.json
git diff surface-snapshot.json | head -40
```

Expected diff: `--output` gone; `--json`, `--agent`, `--jq`, `--ids-only`, `--styled` present (they were hidden before, and hidden flags are excluded from the snapshot). No other changes.

- [ ] **Step 2: Full gate**

```bash
go vet ./... && gofmt -l cmd/ internal/ && go test -race ./... -count=1
make regen-drift
```

Expected: all green, `gofmt -l` prints nothing.

- [ ] **Step 3: Human smoke pass**

```bash
go build -o wenmar ./cmd/wenmar
./wenmar --help                          # five mode flags visible, no --output
./wenmar help output                     # new topic
./wenmar customers list --json --jq 'x'; echo "exit=$?"   # conflict error, exit 1
./wenmar customers list --output md; echo "exit=$?"        # unknown flag, exit 1
./wenmar help exit-codes                 # unchanged (0-10)
```

- [ ] **Step 4: Commit**

```bash
git add surface-snapshot.json
git commit -m "chore: surface snapshot for the boolean output flags"
```

---

## Self-review notes

- **Feedback coverage:** Kill `--output` (Tasks 1-3), kill `--md`/`--quiet`/`--count`/`--html` (Task 1 renderers + Task 3 dropped-flag assertions), keep `--json`/`--agent`/`--jq`/`--ids-only`/`--styled` visible (Task 2), validation-not-precedence conflict error listing the set flags (Task 1 ParseMode — implements the feedback's resolveOutputMode sketch, adapted to the existing ModeSpec plumbing so url.go/doctor.go need zero changes), `--jq 'length'` for counts (Tasks 3, 4, 5, 6), help-topic/README/SKILL.md updates (Tasks 4, 6). Feedback's per-task table for the Phase 3 plan is honored: Tasks 4 (groups), 5 (examples), 7 (exit-code sync), 9 (conformance) untouched.
- **Corrections to the feedback, folded in with evidence:** (1) `--md` shares `renderMarkdown` with the DEFAULT (output.go:117-118), not with `--agent` — the docs say "redundant with the default renderer," never "same as agent." (2) The feedback's table described basecamp's `--agent` as "no help hijack"; wenmar's `--agent` hijacks `--help` deliberately (Phase 1 agent discovery, the `agent-help` topic, SKILL.md's structured-help invariant) — kept, and the `--agent` flag description now documents it inline. (3) The feedback's `cmd.Flags().Changed()` sketch reads flags off the command; the landed `ModeSpec` + `PersistentPreRunE` plumbing achieves the same fail-fast with less churn — `url.go:34` and `doctor.go:141` don't change at all.
- **Deliberate decisions:** `--styled` participates in the conflict set (asking for tables AND json is contradictory — fail loud, matching the no-silent-wrongness principle). The piped auto-switch targets `ModeAgent` so `ModeQuiet` dies entirely — one raw-JSON concept. `count.go`/`html.go` are deleted, not orphaned; YAGNI on html until asked.
- **Ordering:** Tasks 1→2→3 are one compile chain (the plan notes where intermediate states don't build — collapse commits if the repo must stay green per-commit). Task 4/5/6 are docs+examples (any order after 3). Task 7 last. This plan must land BEFORE Phase 4's freshness test (Task 3 of the Phase 4 plan) so the gate locks the final surface.
- **Risk notes:** (1) `TestParseModeAutoSwitch`'s piped-target change (`ModeAgent`) could break any test asserting `ModeQuiet` by name — the Step 3 sweep in Task 3 catches stragglers. (2) `make generate` must be byte-stable apart from the two example strings; any other diff means the working tree and committed generation drifted — stop and reconcile via the regen-drift gate before continuing. (3) SKILL.md example changes here are surgical (flag swaps); the Phase 4 full rewrite then validates them — the two plans must not both rewrite the same lines in flight.