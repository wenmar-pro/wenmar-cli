# Wenmar CLI Production-Readiness — Phase 0 + Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-ruby:subagent-driven-development (recommended) or superpowers-ruby:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stabilize wenmar-cli — green CI against the published SDK, no silent wrongness (every listed bug has a regression test), and credential-safe tests — so the Phase 2 generator cutover can proceed from a trusted base.

**Architecture:** Behavior-preserving bug fixes land as focused PRs against the existing hand-written command surface. No structural changes (those are Phase 2). The one architectural addition is a `WENMAR_CONFIG_HOME` env override so tests never touch real credentials.

**Tech Stack:** Go 1.27, cobra/pflag, wenmar-sdk v0.4.0 (to be tagged), httptest fake API, bubbletea TUI, GitHub Actions CI.

**Spec:** `docs/superpowers/specs/2026-08-30-production-readiness-refactor-design.md`

**Conventions:** Run `go build ./... && go test ./...` before every commit. All commands run from repo root. Commit style: `feat:`, `fix:`, `test:`, `ci:`, `chore:`, `docs:`.

**Verified SDK facts used in this plan** (from `../wenmar-sdk/go` at current HEAD, which becomes `go/v0.4.0`):
- `UpdateCustomerRequest` (wenmar/requests.go:173-188) has ONLY `Emails *[]EmailUpdateAttribute` and `Phones *[]PhoneUpdateAttribute` for nested attributes — no Addresses, no TagIds.
- The enriched spec's `update_customer` PATCH: `emails_attributes` has id/email/label but **no `_destroy`**; `addresses_attributes` has **no id and no `_destroy`**; **no `tag_ids`**. So email/address/tag changes via `customers update` are API-unsupportable → those flags must be **removed**, not wired.
- `CheckCustomerDuplicateParams` (pkg/generated/client.gen.go): `Phone *int` — the `--phone` flag must be parsed to int.
- `ListWorkOrders` exists only as `ListWorkOrders(ctx)` / `ListWorkOrdersWithPagination(ctx)` — there is **no** `ListWorkOrdersParams` type, so `work_orders list --page` cannot be wired → **remove the flag**.
- `ListCustomersParams` has `Page *int` — customers list `--page` already works and stays.

---

## Phase 0 — Unblock main

### Task 1: Repin SDK and verify published build

**Files:**
- Modify: `go.mod`, `go.sum`

The CLI already uses `ListCustomersParams` (SDK commit f575112, unpublished). `GOWORK=off go build` fails today; CI checks out without `go.work`.

- [ ] **Step 1: Tag the SDK**

```bash
cd ../wenmar-sdk/go
git status --short          # MUST be clean — coordinate with SDK owner if not
git log --oneline -5
grep -rn "ListCustomersParams" wenmar/requests.go | head -2   # confirm HEAD has it
git tag go/v0.4.0
git push origin go/v0.4.0
```

- [ ] **Step 2: Bump go.mod and verify with workspace off (what CI sees)**

```bash
GOWORK=off go get github.com/wenmar-pro/wenmar-sdk/go@go/v0.4.0
GOWORK=off go mod tidy
GOWORK=off go build ./... && GOWORK=off go test ./...
```

Expected: build + all tests pass.

- [ ] **Step 3: Verify workspace build too**

```bash
go build ./... && go test ./...
```

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: repin wenmar-sdk to go/v0.4.0"
```

### Task 2: Fix CI auth and add quality gates

**Files:**
- Modify: `.github/workflows/ci.yml` (full rewrite)
- Create: `.golangci.yml`
- Modify: all gofmt-dirty files (via `gofmt -w`)

- [ ] **Step 1: Rewrite ci.yml**

```yaml
name: CI

on:
  pull_request:
  push:
    branches: [main]

env:
  GOPRIVATE: github.com/Wenmar-Pro/*

jobs:
  build-and-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.27"

      - name: Configure Go module auth for private Wenmar repos
        run: |
          git config --global url."https://x-access-token:${{ secrets.GITHUB_TOKEN }}@github.com/Wenmar-Pro/".insteadOf "https://github.com/Wenmar-Pro/"

      - name: Check formatting
        run: |
          unformatted=$(gofmt -l cmd/ internal/)
          if [ -n "$unformatted" ]; then
            echo "These files need gofmt:"
            echo "$unformatted"
            exit 1
          fi

      - name: Vet
        run: go vet ./...

      - name: Download dependencies
        run: go mod download

      - name: Build
        run: go build -o wenmar ./cmd/wenmar/

      - name: Test
        run: go test -race ./...

      - name: Verify CLI runs
        run: ./wenmar --help

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.27"
      - name: Configure Go module auth for private Wenmar repos
        run: |
          git config --global url."https://x-access-token:${{ secrets.GITHUB_TOKEN }}@github.com/Wenmar-Pro/".insteadOf "https://github.com/Wenmar-Pro/"
      - uses: golangci/golangci-lint-action@v8
        with:
          version: latest
      - name: Vulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...

  surface:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.27"
      - name: Configure Go module auth for private Wenmar repos
        run: |
          git config --global url."https://x-access-token:${{ secrets.GITHUB_TOKEN }}@github.com/Wenmar-Pro/".insteadOf "https://github.com/Wenmar-Pro/"
      - name: Download dependencies
        run: go mod download
      - name: Surface snapshot diff
        run: make surface-diff
      - name: Shellcheck install script
        run: |
          sudo apt-get update && sudo apt-get install -y shellcheck
          shellcheck install-cli
```

Key changes vs today: `secrets.GITHUB_TOKEN` in the auth condition's place (it now runs unconditionally — the secrets context is always populated on GH-hosted runners), gofmt gate, vet, `-race`, golangci-lint + govulncheck job, `make surface-diff`, shellcheck.

- [ ] **Step 2: Create .golangci.yml**

```yaml
version: "2"
linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - gofmt
    - unused
  exclusions:
    rules:
      - path: _test\.go
        linters:
          - errcheck
```

- [ ] **Step 3: gofmt the tree now (so the new gate is green)**

```bash
gofmt -w cmd/ internal/
go build ./... && go test ./...
git diff --stat
```

Expected: ~45 files, whitespace-only changes.

- [ ] **Step 4: Commit (formatting separate from CI)**

```bash
git add .github/workflows/ci.yml .golangci.yml
git commit -m "ci: fix private-module auth; add fmt/vet/race/lint/vulncheck/surface-diff/shellcheck gates"
git add -u
git commit -m "chore: gofmt the tree"
```

### Task 3: Fix .gitignore gencli landmine

**Files:**
- Modify: `.gitignore`

`gencli` (line 5, no slash) matches `cmd/gencli/` — new generator test files are silently ignored (`git check-ignore cmd/gencli/gen_test.go` proves it).

- [ ] **Step 1: Change line 5 from `gencli` to `/gencli`**

```gitignore
/wenmar
go.work
go.work.sum
.worktrees/
/gencli
cmd/gen_*.go
```

- [ ] **Step 2: Verify**

```bash
git check-ignore -v cmd/gencli/gen_test.go && echo "STILL IGNORED — BAD" || echo "not ignored — good"
git check-ignore -v cmd/gen_customers.go
```

Expected: first check finds nothing; second still matches `cmd/gen_*.go`.

- [ ] **Step 3: Commit**

```bash
git add .gitignore
git commit -m "fix: .gitignore no longer ignores new files in cmd/gencli"
```

---

## Phase 1 — Correctness fixes

### Task 4: Fix gencli infinite recursion + dead config

**Files:**
- Modify: `cmd/gencli/gen.go` (sdkMethodNameFor, parseBodyFields, delete dead helpers)
- Modify: `cmd/gencli/main.go` (no change needed — Overrides.FlagOverrides already parsed)
- Modify: `cmd/gen_overrides.yaml` (delete dead entries)
- Create: `cmd/gencli/gen_test.go`

- [ ] **Step 1: Failing test — create `cmd/gencli/gen_test.go`**

```go
package main

import "testing"

func TestSdkMethodNameFor(t *testing.T) {
	tests := []struct {
		name string
		cmd  GenCommand
		want string
	}{
		{"explicit SDKMethod wins", GenCommand{OperationID: "list_vendors", SDKMethod: "ListVendors"}, "ListVendors"},
		{"derives from operationId", GenCommand{OperationID: "list_service_categories"}, "ListServiceCategories"},
		{"multi-segment operationId", GenCommand{OperationID: "show_work_order_estimate"}, "ShowWorkOrderEstimate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sdkMethodNameFor(tt.cmd)
			if got != tt.want {
				t.Errorf("sdkMethodNameFor() = %q, want %q", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Verify it fails**

Run: `go test ./cmd/gencli/ -run TestSdkMethodNameFor -v -timeout 30s`
Expected: FAIL — goroutine stack overflow (infinite recursion) on the no-override cases.

- [ ] **Step 3: Fix the recursion (gen.go:1094-1099)**

```go
func sdkMethodNameFor(cmd GenCommand) string {
	if cmd.SDKMethod != "" {
		return cmd.SDKMethod
	}
	return sdkMethodName(cmd.OperationID)
}
```

- [ ] **Step 4: Delete dead helpers**

```bash
grep -n "pathPrefixForRunner\|pathPrefixString" cmd/gencli/gen.go
```

Delete both functions (gen.go:707-737 and 924-931). Then confirm the package still builds:

```bash
go build ./cmd/gencli/
```

- [ ] **Step 5: Wire flag_overrides into parseBodyFields**

Current state: `Overrides.FlagOverrides` is parsed (main.go:125) but never consumed; the `flag_overrides:` YAML block is dead config. Read `parseBodyFields` in gen.go, then:

1. Change its signature to accept the overrides: `parseBodyFields(op Operation, requestStruct string, flagOverrides map[string]FlagOverride)`.
2. In `buildCommand` (gen.go:139), pass `overrides.FlagOverrides[op.OperationID]`.
3. Inside parseBodyFields, when emitting each body field, look up the override keyed by the field's dotted path (e.g. `customer.first_name`). If present: use `fo.Flag` as the flag name when non-empty, `fo.Help` as the usage text when non-empty, and call the required-marking path when `fo.Required` is true (mirror how vehicle make/model/year are marked in the existing generated create handlers).
4. Verify with the smoke generation in Step 6: `gen_customers.go` must emit `--full-name` for `customer.first_name` (per the YAML at gen_overrides.yaml:305-311).

- [ ] **Step 6: Smoke-generate to a temp dir**

```bash
go run ./cmd/gencli -spec ../wenmar-sdk/spec/openapi.enriched.yaml -overrides cmd/gen_overrides.yaml -out /tmp/opencode/gen-smoke/
grep -n "full-name" /tmp/opencode/gen-smoke/gen_customers.go
```

Expected: generation completes without stack overflow; `--full-name` appears in the generated customers file (proof flag_overrides now works).

- [ ] **Step 7: Delete dead/contradictory YAML entries**

In `cmd/gen_overrides.yaml`, delete these `commands:` entries (each op is also in `exclude:`, which wins — the entries are dead and misleading): `create_customer` (101-106), `delete_customer` (113-117), `update_tags` (287-291), `create_customer_tag` (292-296), `create_vehicle_tag` (297-301). Add above `commands:`:

```yaml
# NOTE: operations listed in `exclude:` must not also appear under
# `commands:` — exclusion takes precedence and the entry would be dead.
```

- [ ] **Step 8: Run generator tests + smoke again, commit**

```bash
go test ./cmd/gencli/ -v
git add cmd/gencli/ cmd/gen_overrides.yaml
git commit -m "fix(gencli): recursion crash, wire flag_overrides, drop dead helpers and YAML entries"
```

### Task 5: Remove API-unsupportable `customers update` flags

**Files:**
- Modify: `cmd/customers.go`
- Modify: `cmd/integration_test.go` (new tests + PATCH body capture)

Spec ground truth (verified above): update_customer supports phones_attributes with `_destroy` only. Emails have no `_destroy`; addresses have no id/`_destroy`; no tag_ids. `--remove-email`, `--remove-address`, `--address`, `--tag-id` on update are impossible → remove those four flags from `customers update` (create keeps `--address` and `--tag-id`, which the create endpoint supports).

- [ ] **Step 1: Add PATCH body capture to the fake API**

In `cmd/integration_test.go`, the `/customers` handler currently only handles GET/POST. Find the `mux.HandleFunc("/customers", ...)` block and extend it. Also add a package-level capture near `startFakeAPI`:

```go
// lastPatchBody records the most recent PATCH body seen by the fake API,
// keyed by path. Tests use this to assert request wiring.
var lastPatchBody sync.Map // path -> []byte
```

Add a `/customers/` subpath handler (the fake currently has none — PATCH targets `/customers/42`):

```go
mux.HandleFunc("/customers/", func(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+token {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
		return
	}
	if r.Method == http.MethodPatch {
		body, _ := io.ReadAll(r.Body)
		lastPatchBody.Store(r.URL.Path, body)
		writeJSON(w, http.StatusOK, map[string]any{"id": 42, "full_name": "Updated Person"})
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "not found")
})
```

Add `"io"` and `"sync"` to imports if missing.

- [ ] **Step 2: Write the failing tests**

```go
func TestCustomersUpdate_PhonesAndEmailsWireThrough(t *testing.T) {
	srv := startFakeAPI(t, "tok-update")
	defer srv.Close()
	t.Setenv("WENMAR_URL", srv.URL)

	if _, err := execute("customers", "update", "42",
		"--email", "work|jane@corp.com",
		"--phone", "cell|555-0100",
		"--remove-phone", "7"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, ok := lastPatchBody.Load("/customers/42")
	if !ok {
		t.Fatal("no PATCH body captured at /customers/42")
	}
	var body struct {
		Customer struct {
			EmailsAttributes []struct {
				Email string `json:"email"`
				Label *string `json:"label"`
			} `json:"emails_attributes"`
			PhonesAttributes []struct {
				UnderscoreDestroy *bool   `json:"_destroy"`
				Id                *int    `json:"id"`
				Label             *string `json:"label"`
				Number            *string `json:"number"`
			} `json:"phones_attributes"`
		} `json:"customer"`
	}
	if err := json.Unmarshal(raw.([]byte), &body); err != nil {
		t.Fatalf("unmarshal captured body: %v\nbody: %s", err, raw.([]byte))
	}
	if n := len(body.Customer.EmailsAttributes); n != 1 || body.Customer.EmailsAttributes[0].Email != "jane@corp.com" {
		t.Errorf("emails_attributes not wired: %+v", body.Customer.EmailsAttributes)
	}
	if n := len(body.Customer.PhonesAttributes); n != 2 {
		t.Fatalf("want 2 phone attrs (1 add + 1 destroy), got %d: %+v", n, body.Customer.PhonesAttributes)
	}
	sawDestroy := false
	for _, p := range body.Customer.PhonesAttributes {
		if p.UnderscoreDestroy != nil && p.Id != nil && *p.Id == 7 {
			sawDestroy = true
		}
	}
	if !sawDestroy {
		t.Errorf("phones_attributes missing {_destroy:true, id:7}: %+v", body.Customer.PhonesAttributes)
	}
}

func TestCustomersUpdate_UnsupportableFlagsRemoved(t *testing.T) {
	cases := map[string]string{
		"remove-email":   "3",
		"remove-address": "5",
		"tag-id":         "11",
		"address":        "1 Main St|Springfield|IL|62704|USA",
	}
	for flag, value := range cases {
		t.Run(flag, func(t *testing.T) {
			srv := startFakeAPI(t, "tok-rm")
			defer srv.Close()
			t.Setenv("WENMAR_URL", srv.URL)

			_, err := execute("customers", "update", "42", "--"+flag, value)
			if err == nil {
				t.Errorf("--%s must error: unsupported by the update API (flag removed)", flag)
			}
		})
	}
}
```

- [ ] **Step 3: Verify the new tests fail**

```bash
go test ./cmd/ -run "TestCustomersUpdate_PhonesAndEmails|TestCustomersUpdate_Unsupportable" -v
```

Expected: `PhonesAndEmails` passes or fails only on wiring details (emails/phones are already wired — this test locks the behavior); `UnsupportableFlagsRemoved` FAILS (all four flags accepted silently today).

- [ ] **Step 4: Remove the four flags**

In `cmd/customers.go`:

1. Delete these registrations (customers.go:164-166,168):
   - `customersUpdateCmd.Flags().StringArrayVar(&customerAddresses, "address", ...)` (line 164)
   - `customersUpdateCmd.Flags().IntSliceVar(&customerTagIDs, "tag-id", ...)` (line 165)
   - `customersUpdateCmd.Flags().IntSliceVar(&customerRemoveEmailIDs, "remove-email", ...)` (line 166)
   - `customersUpdateCmd.Flags().IntSliceVar(&customerRemoveAddressIDs, "remove-address", ...)` (line 168)
2. Delete vars `customerRemoveEmailIDs`, `customerRemoveAddressIDs` (customers.go:121,123). KEEP `customerAddresses` and `customerTagIDs` — the create path (lines 149-150) still uses them.
3. In `applyCustomerUpdateFlags` (customers.go:434-477): no address/tag code exists there (already verified — only emails/phones). Nothing to remove; confirm with:
   ```bash
   grep -n "customerAddresses\|customerTagIDs\|customerRemove" cmd/customers.go
   ```
   Expected after edit: only create-path references remain (lines ~149-150 and in `applyCustomerCreateFlags`).
4. In the update body builder, confirm `wenmar.UpdateCustomerRequest` construction passes only Emails/Phones (it does — customers.go uses `applyCustomerUpdateFlags(&req)`).

- [ ] **Step 5: Run customer tests**

```bash
go test ./cmd/ -run "TestCustomers" -v
```

- [ ] **Step 6: Commit**

```bash
git add cmd/customers.go cmd/integration_test.go
git commit -m "fix(customers): remove update flags the API can't support; lock phones/emails wiring with tests"
```

### Task 6: Remove `work_orders list --page`; wire `customers duplicates --phone`

**Files:**
- Modify: `cmd/work_orders.go` (remove dead `--page`)
- Modify: `cmd/customers.go` (duplicates --phone)
- Modify: `cmd/integration_test.go` (duplicates test)

Verified: no `ListWorkOrdersParams` exists in the SDK — `--page` cannot be wired to `work_orders list`. Remove it (a removed flag errors clearly; a silently-ignored one lies). `CheckCustomerDuplicateParams.Phone` is `*int` — parse the string flag to int.

- [ ] **Step 1: Failing test for duplicates --phone**

```go
func TestCustomersDuplicates_PhoneWiresThrough(t *testing.T) {
	srv := startFakeAPI(t, "tok-dup")
	defer srv.Close()
	t.Setenv("WENMAR_URL", srv.URL)

	if _, err := execute("customers", "duplicates", "--first-name", "Jane", "--phone", "5550100"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

And add to the fake API a handler for the duplicates endpoint (it currently 404s — extend `startFakeAPI`):

```go
mux.HandleFunc("/customers/check_duplicate", func(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+token {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API token")
		return
	}
	lastDupQuery.Store(r.URL.Query())
	writeJSON(w, http.StatusOK, map[string]any{"matches": []any{}})
})
```

with `var lastDupQuery atomic.Value` near the other captures, and strengthen the test's assertion:

```go
	q, ok := lastDupQuery.Load().(url.Values)
	if !ok {
		t.Fatal("no check_duplicate query captured")
	}
	if q.Get("phone") != "5550100" {
		t.Errorf("phone param not sent, got %q", q.Get("phone"))
	}
	if q.Get("first_name") != "Jane" {
		t.Errorf("first_name param not sent, got %q", q.Get("first_name"))
	}
```

(add `"net/url"` import; add `t.Setenv("WENMAR_TOKEN", "tok-dup")` if execute doesn't inherit it — check how other tests pass tokens; most set the env var).

- [ ] **Step 2: Verify it fails**

```bash
go test ./cmd/ -run TestCustomersDuplicates -v
```

Expected: FAIL — phone never sent (current params struct omits Phone).

- [ ] **Step 3: Wire phone in the duplicates handler**

In `cmd/customers.go`, find the duplicates handler (~line 315) building `CheckCustomerDuplicateParams`. Replace with:

```go
params := wenmar.CheckCustomerDuplicateParams{
	FirstName: strPtr(customerDuplicateFirstName),
	LastName:  strPtr(customerDuplicateLastName),
	Email:     strPtr(customerDuplicateEmail),
}
if customerDuplicatePhone != "" {
	phone, err := strconv.Atoi(customerDuplicatePhone)
	if err != nil {
		return fmt.Errorf("--phone must be a numeric phone number (the API expects digits): %w", err)
	}
	params.Phone = &phone
}
```

Confirm `strconv` is imported (it is — used at line 236).

- [ ] **Step 4: Remove `work_orders list --page`**

In `cmd/work_orders.go`, delete line 96:

```go
workOrdersListCmd.Flags().Int("page", 0, "Page number")
```

Nothing reads it (verified). Also update the `work_orders list` Short if it mentions paging by flag — it says "paginated via the Link header" which remains true.

- [ ] **Step 5: Run tests**

```bash
go test ./cmd/ -run "TestCustomersDuplicates|TestWorkOrders" -v && go test ./...
```

- [ ] **Step 6: Commit**

```bash
git add cmd/work_orders.go cmd/customers.go cmd/integration_test.go
git commit -m "fix: wire customers duplicates --phone; remove unimplementable work_orders list --page"
```

### Task 7: Unknown subcommands must fail with suggestions

**Files:**
- Modify: `cmd/customers.go`, `cmd/vehicles.go`, `cmd/work_orders.go`, `cmd/drivers.go`, `cmd/vendors.go`, `cmd/statements.go`, `cmd/tags.go`, `cmd/service_categories.go`, `cmd/account.go`, `cmd/locations.go` (parent commands)
- Modify: `cmd/root.go` (SuggestionsMinimumDistance)
- Modify: `cmd/completion.go` (deprecated ExactValidArgs)
- Modify: `cmd/integration_test.go` (tests)

Today `wenmar customers delete 42` prints help and exits 0 because non-runnable parents fall back to help. Fix: give every parent `Args: cobra.NoArgs` + a help RunE.

- [ ] **Step 1: Failing test**

```go
func TestUnknownSubcommandFails(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"nonexistent customers subcommand", []string{"customers", "delete"}},
		{"typo'd work_orders subcommand", []string{"work_orders", "delet"}},
		{"cross-resource concept", []string{"vehicles", "estimate"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := execute(tc.args...)
			if err == nil {
				t.Errorf("%v: expected error, got exit 0", tc.args)
			}
		})
	}
}
```

- [ ] **Step 2: Verify it fails**

```bash
go test ./cmd/ -run TestUnknownSubcommandFails -v
```

Expected: FAIL — all three exit 0 today.

- [ ] **Step 3: Add Args+RunE to each parent command**

For each resource parent (customers, vehicles, work_orders, drivers, vendors, statements, tags, service_categories), change the command struct from e.g.:

```go
var customersCmd = &cobra.Command{
	Use:   "customers",
	Short: "Manage customers",
}
```

to:

```go
var customersCmd = &cobra.Command{
	Use:   "customers",
	Short: "Manage customers",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
```

Check each file for the parent's current shape — some parents may already have `Run`/`RunE` (locations and account are runnable single-action commands; leave those alone). `locations` has a `show` subcommand and its own parent shape — only add the pattern where the parent has children and no Run/RunE.

- [ ] **Step 4: Enable suggestions on root + fix deprecated cobra API**

In `cmd/root.go` (rootCmd definition, lines 38-45) add:

```go
	SuggestionsMinimumDistance: 2,
```

In `cmd/completion.go:20`, replace `Args: cobra.ExactValidArgs(1)` with:

```go
	Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
```

- [ ] **Step 5: Verify manually and by test**

```bash
go build -o wenmar ./cmd/wenmar
./wenmar customers delete 42; echo "exit=$?"          # non-zero + "unknown command" + suggestion
./wenmar customers                                   # still prints help, exit 0
./wenmar work_orders shwo                            # "Did you mean this? show"
go test ./cmd/ -run TestUnknownSubcommandFails -v && go test ./...
```

- [ ] **Step 6: Commit**

```bash
git add cmd/
git commit -m "fix: unknown subcommands exit non-zero with suggestions instead of exit-0 help"
```

### Task 8: Fix `wenmar help <command>` fallback

**Files:**
- Modify: `cmd/help_topics.go:163-164`
- Modify: `cmd/integration_test.go` (test)

Today the fallback `return cmd.Root().Help()` ignores `args[0]` — `wenmar help customers` prints root help.

- [ ] **Step 1: Failing test**

```go
func TestHelpCommandFallback(t *testing.T) {
	// help <command> prints that command's help, not root help.
	out, err := execute("help", "customers")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Manage customers") || !strings.Contains(out, "Available Commands") {
		t.Errorf("help customers printed:\n%s", out)
	}

	out, err = execute("help", "customers", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "List all customers") {
		t.Errorf("help customers list printed:\n%s", out)
	}

	// Topics still win over command names.
	out, err = execute("help", "output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Output Formats") {
		t.Errorf("help output printed:\n%s", out)
	}

	// Unknown help target errors instead of printing root help.
	_, err = execute("help", "nosuchthing")
	if err == nil {
		t.Error("help nosuchthing should error")
	}
}
```

Note: `help` currently takes `Args: cobra.MaximumNArgs(1)` (help_topics.go:129) — change it to `cobra.MinimumNArgs(0)` (no max) so `help customers list` works:

```go
var helpCmd = &cobra.Command{
	Use:   "help [command|topic]",
	Short: "Help about any command or topic",
	Args:  cobra.ArbitraryArgs,
	RunE:  runHelp,
}
```

- [ ] **Step 2: Verify it fails**

```bash
go test ./cmd/ -run TestHelpCommandFallback -v
```

Expected: FAIL — `help customers` prints root help.

- [ ] **Step 3: Implement the fallback**

In `cmd/help_topics.go`, replace lines 163-164:

```go
	// Fall back to cobra's built-in help for commands.
	return cmd.Root().Help()
```

with:

```go
	// Resolve the target command from the remaining args (topics were
	// checked above).
	target, _, err := cmd.Root().Find(args)
	if err != nil || target == cmd.Root() {
		return fmt.Errorf("unknown help topic or command %q — run `wenmar help` for topics or `wenmar --help` for commands", args[0])
	}
	return target.Help()
```

Note: cobra's `Find` returns the deepest matching command; for `["customers"]` it returns the customers parent; for `["customers","list"]` the list command. If the first arg matches nothing, `Find` returns the root with the args as leftovers — the `target == cmd.Root()` check catches that.

- [ ] **Step 4: Verify manually and by test**

```bash
go build -o wenmar ./cmd/wenmar
./wenmar help customers | head -8
./wenmar help customers list | head -8
./wenmar help output | head -3
./wenmar help nosuchthing; echo "exit=$?"
go test ./cmd/ -run TestHelpCommandFallback -v
```

- [ ] **Step 5: Commit**

```bash
git add cmd/help_topics.go cmd/integration_test.go
git commit -m "fix: wenmar help <command> prints the command's help, not root help"
```

### Task 9: Fix surface-snapshot required-flag bug; hide the command

**Files:**
- Modify: `cmd/surface_snapshot.go:70-75` (range-copy bug), `:13-29` (Hidden)
- Create: `cmd/surface_snapshot_test.go`
- Modify: `surface-snapshot.json` (regenerated)

- [ ] **Step 1: Failing test — create `cmd/surface_snapshot_test.go`**

```go
package cmd

import "testing"

func TestSurfaceSnapshotRecordsRequiredFlags(t *testing.T) {
	surf := buildSurfaceSnapshot(vehiclesCreateCmd, "vehicles create")
	if len(surf.Flags) == 0 {
		t.Fatal("no flags captured for vehicles create")
	}
	for _, f := range surf.Flags {
		if f.Name == "make" && !f.Required {
			t.Errorf("vehicles create --make must be required:true in snapshot (range-copy bug)")
		}
		if f.Name == "customer-id" && !f.Required {
			t.Errorf("vehicles create --customer-id must be required:true in snapshot")
		}
	}
}
```

(Confirm before writing: `vehicles create` marks make/model/year/customer-id required — check `cmd/vehicles.go` init for `MarkFlagRequired` calls and adjust names to what's actually marked.)

- [ ] **Step 2: Verify it fails**

```bash
go test ./cmd/ -run TestSurfaceSnapshotRecordsRequiredFlags -v
```

Expected: FAIL — Required always false.

- [ ] **Step 3: Fix the loop and hide the command**

Replace lines 70-75:

```go
	// Mark required flags.
	for i := range surf.Flags {
		if lf := cmd.Flags().Lookup(surf.Flags[i].Name); lf != nil {
			surf.Flags[i].Required = lf.Annotations["cobra_annotation_bash_completion_one_required_flag"] != nil
		}
	}
```

And in the command struct add `Hidden: true`:

```go
var surfaceSnapshotCmd = &cobra.Command{
	Use:    "surface-snapshot",
	Short:  "Dump the command tree as JSON (for CI diffing)",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		snapshot := buildSurfaceSnapshot(rootCmd, "")
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(snapshot); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}
```

- [ ] **Step 4: Regenerate the committed snapshot**

```bash
go run ./cmd/wenmar surface-snapshot > surface-snapshot.json
grep -c '"required": true' surface-snapshot.json
```

Expected: >0 (today 0). Spot-check `work_orders create` and `vehicles create` entries.

- [ ] **Step 5: Run tests, commit**

```bash
go test ./cmd/ -run TestSurfaceSnapshot -v && go test ./...
git add cmd/surface_snapshot.go cmd/surface_snapshot_test.go surface-snapshot.json
git commit -m "fix: surface-snapshot records required flags (range-copy bug); hide CI-only command"
```

### Task 10: One agent-discovery schema

**Files:**
- Modify: `internal/agent/discovery.go` (required fix + BuildCommandInfo)
- Modify: `cmd/root.go:108-132` (help mode uses the shared builder)
- Modify: `cmd/help_topics.go` (agent-help topic)
- Create: `cmd/agent_surface_test.go`
- Modify: `cmd/integration_test.go` (execute() must reset agentFlag)

- [ ] **Step 1: Failing tests — create `cmd/agent_surface_test.go`**

```go
package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/agent"
)

func TestAgentCatalog_RequiredFlags(t *testing.T) {
	cat := agent.BuildCatalog(rootCmd)
	var woCreate *agent.CommandInfo
	for i := range cat.Commands {
		if cat.Commands[i].Path == "work_orders create" {
			woCreate = &cat.Commands[i]
		}
	}
	if woCreate == nil {
		t.Fatal("work_orders create not in catalog")
	}
	for _, f := range woCreate.Flags {
		if f.Name == "customer-id" && !f.Required {
			t.Errorf("work_orders create --customer-id must be required:true in catalog")
		}
	}
}

func TestAgentSurfacesAgree(t *testing.T) {
	cat := agent.BuildCatalog(rootCmd)
	var catalogInfo *agent.CommandInfo
	for i := range cat.Commands {
		if cat.Commands[i].Path == "customers list" {
			catalogInfo = &cat.Commands[i]
		}
	}
	if catalogInfo == nil {
		t.Fatal("customers list not in catalog")
	}

	agentFlag = true
	defer func() { agentFlag = false }()

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"customers", "list", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var helpInfo agent.CommandInfo
	if err := json.Unmarshal(buf.Bytes(), &helpInfo); err != nil {
		t.Fatalf("unmarshal help JSON: %v\noutput: %s", err, buf.String())
	}
	if helpInfo.Path != "wenmar customers list" {
		t.Errorf("help path = %q, want %q", helpInfo.Path, "wenmar customers list")
	}

	have := map[string]bool{}
	for _, f := range catalogInfo.Flags {
		have[f.Name] = true
	}
	for _, f := range helpInfo.Flags {
		if !have[f.Name] {
			t.Errorf("help surface lists flag %q missing from catalog", f.Name)
		}
	}
}
```

Also extend `execute()` in integration_test.go to reset `agentFlag` (it currently resets only some flags):

```go
func execute(args ...string) (string, error) {
	// Reset global output flags so prior tests don't leak state.
	mdFlag, jsonFlag, agentFlag, jqFlag = false, false, false, ""
	idsOnlyFlag, countFlag, htmlFlag, styledFlag = false, false, false, false
	currentDebugInfo = nil
	rootCmd.SetArgs(args)
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	err := rootCmd.Execute()
	return buf.String(), err
}
```

- [ ] **Step 2: Verify they fail**

```bash
go test ./cmd/ -run "TestAgent" -v
```

Expected: `RequiredFlags` FAILS (`f.Annotations != nil` is the current wrong check); `SurfacesAgree` FAILS (help mode builds different JSON today).

- [ ] **Step 3: Fix required detection in discovery.go:84**

```go
Required: f.Annotations != nil && f.Annotations["cobra_annotation_bash_completion_one_required_flag"] != nil,
```

- [ ] **Step 4: Add BuildCommandInfo and wire help mode to it**

In `internal/agent/discovery.go`, add:

```go
// BuildCommandInfo builds the agent-facing description of one command.
// It is the same builder that produces the catalog, so --help --agent and
// `wenmar commands` can never disagree.
func BuildCommandInfo(root *cobra.Command, cmd *cobra.Command) CommandInfo {
	info := CommandInfo{
		Path:        cmd.CommandPath(),
		Description: cmd.Short,
		Aliases:     cmd.Aliases,
		Args:        extractArgs(cmd),
		Canonical:   true,
		Type:        "command",
	}
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		if pf := root.PersistentFlags().Lookup(f.Name); pf != nil {
			return // global flag — documented once at root, not per leaf
		}
		info.Flags = append(info.Flags, FlagInfo{
			Name:        f.Name,
			Short:       f.Shorthand,
			Type:        f.Value.Type(),
			Required:    f.Annotations != nil && f.Annotations["cobra_annotation_bash_completion_one_required_flag"] != nil,
			Default:     f.DefValue,
			Description: f.Usage,
		})
	})
	return info
}
```

Then in `cmd/root.go`, replace the `SetHelpFunc` body's agent branch (lines 110-129) with:

```go
	defaultHelpFunc := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if agentFlag {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(agent.BuildCommandInfo(rootCmd, cmd)); err != nil {
				fmt.Fprintln(os.Stderr, "wenmar: help:", err)
				os.Exit(1)
			}
			return
		}
		defaultHelpFunc(cmd, args)
	})
```

- [ ] **Step 5: Add the agent-help topic**

Append to `helpTopics` in `cmd/help_topics.go`:

```go
	{
		name:  "agent-help",
		title: "Agent Help Mode",
		content: `--agent --help emits structured JSON describing any command:

  wenmar customers list --help --agent

Fields:
  path        Full command path (e.g. "wenmar customers list")
  description One-line summary
  aliases     Alternative names for this command
  args        Positional arguments [{name, type, required}]
  flags       Flags [{name, short, type, required, default, description}]
              Global output flags are omitted from leaf help; see "wenmar help output".

The same schema powers "wenmar commands" (the full catalog). One builder
produces both, so they always agree.`,
	},
```

- [ ] **Step 6: Run tests and verify the JSON manually**

```bash
go test ./cmd/ -run "TestAgent" -v && go test ./internal/agent/ -v
go build -o wenmar ./cmd/wenmar && ./wenmar customers list --help --agent | head -30
./wenmar work_orders create --help --agent | grep -A4 '"customer-id"'
```

Expected: work_orders create flags show `"required": true`.

- [ ] **Step 7: Commit**

```bash
git add internal/agent/discovery.go cmd/root.go cmd/help_topics.go cmd/agent_surface_test.go cmd/integration_test.go
git commit -m "fix: unify agent-discovery surfaces; correct required flags; add agent-help topic"
```

### Task 11: Test credential isolation (WENMAR_CONFIG_HOME)

**Files:**
- Modify: `internal/config/config.go` (env override in ConfigPath), `internal/config/xdg.go` (xdgConfigPath), `internal/config/repo.go` (trustedReposPath)
- Create: `cmd/credentials.go` (env-aware credential store constructor)
- Modify: `cmd/integration_test.go` (TestMain), `cmd/auth.go`, `cmd/setup.go`, `cmd/doctor.go` (use the constructor)
- Test: `internal/config/config_test.go`, `cmd/credentials_test.go`

The SDK's `NewCredentialStore()` hardcodes `~/.config/wenmar/credentials.json` (credential_store.go:119-124). `FileStore{Path}` is exported, so the CLI can build its own env-aware store. All cmd/ call sites switch to it.

- [ ] **Step 1: Failing test — ConfigPath honors WENMAR_CONFIG_HOME**

Add to `internal/config/config_test.go`:

```go
func TestConfigPathHonorsConfigHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WENMAR_CONFIG_HOME", dir)
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath: %v", err)
	}
	if want := filepath.Join(dir, "wenmar", "config"); path != want {
		t.Errorf("ConfigPath = %q, want %q", path, want)
	}
}
```

- [ ] **Step 2: Verify it fails**

```bash
go test ./internal/config/ -run TestConfigPathHonorsConfigHome -v
```

Expected: FAIL — path is the XDG default.

- [ ] **Step 3: Implement the env override**

In `internal/config/xdg.go`, change `xdgConfigPath`:

```go
// xdgConfigPath returns the config path: $WENMAR_CONFIG_HOME/wenmar/config
// when WENMAR_CONFIG_HOME is set (used by tests to avoid touching real
// credentials), otherwise $XDG_CONFIG_HOME/wenmar/config, or
// ~/.config/wenmar/config as fallback.
func xdgConfigPath() (string, error) {
	if base := os.Getenv("WENMAR_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "wenmar", "config"), nil
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine home directory: %w", err)
		}
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "wenmar", "config"), nil
}
```

In `internal/config/repo.go`, same treatment at the top of `trustedReposPath`:

```go
func trustedReposPath() (string, error) {
	if base := os.Getenv("WENMAR_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "wenmar", "trusted_repos"), nil
	}
	// ... existing XDG fallback unchanged
}
```

- [ ] **Step 4: Create the env-aware credential store constructor**

Create `cmd/credentials.go`:

```go
package cmd

import (
	"os"
	"path/filepath"

	authpkg "github.com/wenmar-pro/wenmar-sdk/go/pkg/auth"
)

// newCredentialStore returns the SDK credential store with the file fallback
// redirected under $WENMAR_CONFIG_HOME when set. Tests set WENMAR_CONFIG_HOME
// so they never touch the developer's real keyring or credentials file.
func newCredentialStore() authpkg.CredentialStore {
	if base := os.Getenv("WENMAR_CONFIG_HOME"); base != "" {
		return authpkg.FileStore{
			Path: filepath.Join(base, "wenmar", "credentials.json"),
		}
	}
	return authpkg.NewCredentialStore()
}
```

Check the SDK's `FileStore` zero value semantics first:

```bash
go doc github.com/wenmar-pro/wenmar-sdk/go/pkg/auth.FileStore
```

If `FileStore` alone can't be used as a `CredentialStore` return value (interface method-set mismatch), wrap it exactly as the SDK's `fallbackStore` does — the doc output will confirm; `FileStore` implements GetToken/SaveToken/DeleteToken (credential_store.go:~85-111), so it satisfies the interface directly.

- [ ] **Step 5: Replace call sites**

```bash
grep -rn "authpkg.NewCredentialStore()" cmd/ internal/
```

Replace every hit in `cmd/` with `newCredentialStore()` (auth.go:104,125,146,197; setup.go — check; doctor.go:62 area; doctor_test.go; auth_test.go). The `internal/auth` hits (auth.go:52,101) need the store injected — those functions already take a store parameter in the `...From`/`...WithStore` variants, so instead change the plain wrappers to accept a store too, or (simpler, fewer ripples) have cmd call sites pass `newCredentialStore()` into the `...WithStore` variants directly. Choose: replace `auth.ResolveTokenWithSource(tokenFlag, configPath)` in cmd with `auth.ResolveTokenWithSourceFrom(tokenFlag, configPath, newCredentialStore())` and similarly for manager resolution. Keep the no-store wrappers for library callers but route them through the env-aware default:

```go
// internal/auth/auth.go
func newDefaultStore() authpkg.CredentialStore {
	if base := os.Getenv("WENMAR_CONFIG_HOME"); base != "" {
		return authpkg.FileStore{Path: filepath.Join(base, "wenmar", "credentials.json")}
	}
	return authpkg.NewCredentialStore()
}
```

Then `ResolveTokenWithSource` / `ResolveAuthManager` call `newDefaultStore()` internally.

- [ ] **Step 6: Isolate the test process**

In `cmd/integration_test.go` TestMain (lines 15-21), redirect config home before any test runs:

```go
func TestMain(m *testing.M) {
	// Ensure tests never inherit a real token or base URL...
	os.Unsetenv("WENMAR_TOKEN")
	os.Unsetenv("WENMAR_URL")
	// ...and never touch the developer's real credentials: point the
	// config home at a temp dir for the whole test binary.
	cfgHome, err := os.MkdirTemp("", "wenmar-test-config-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "mkdir temp:", err)
		os.Exit(1)
	}
	os.Setenv("WENMAR_CONFIG_HOME", cfgHome)
	defer os.RemoveAll(cfgHome)

	code := m.Run()
	os.Exit(code)
}
```

Note: `internal/config` and `internal/auth` tests set the env per-test with `t.Setenv` where needed (they don't share the cmd TestMain). Add the same `t.Setenv("WENMAR_CONFIG_HOME", t.TempDir())` pattern to auth_test.go / doctor_test.go tests that construct stores — after Step 5 they use `newCredentialStore()`, which honors the env var.

- [ ] **Step 7: Update the two credential-polluting tests**

- `cmd/auth_test.go` `TestAuthLogin_StoresToken` (lines 38-60): add `t.Setenv("WENMAR_CONFIG_HOME", t.TempDir())` at the top; change `authpkg.NewCredentialStore()` (line 51) to `newCredentialStore()`; the deferred `clearStoredCredentials(t)` now cleans the temp store only.
- `cmd/doctor_test.go` `TestDoctor_NoToken` (lines 60-63): add the same `t.Setenv` + `newCredentialStore()` — it must stop deleting the developer's real token.
- `clearStoredCredentials` helper (auth_test.go:152-158): switch to `newCredentialStore()`.

- [ ] **Step 8: Add a leak guard test**

In `cmd/credentials_test.go`:

```go
package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWenmarConfigHomeRedirectsCredentials(t *testing.T) {
	t.Setenv("WENMAR_CONFIG_HOME", t.TempDir())
	store := newCredentialStore()
	// The concrete type must be the SDK FileStore pointed under the env dir.
	if _, ok := store.(interface{ GetToken(context.Context) (*authpkg.Token, error) }); !ok {
		t.Fatal("store does not satisfy CredentialStore")
	}
	// Best-effort: ensure the default (no env) path is NOT under $HOME by
	// checking the env-redirected path is used for writes.
	home, _ := os.UserHomeDir()
	realPath := filepath.Join(home, ".config", "wenmar", "credentials.json")
	before, err1 := os.Stat(realPath)
	if err := store.SaveToken(context.Background(), &authpkg.Token{AccessToken: "x"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}
	after, err2 := os.Stat(realPath)
	if err1 == nil && err2 == nil && before.ModTime() != after.ModTime() {
		t.Error("real credentials file was modified — WENMAR_CONFIG_HOME redirect failed")
	}
}
```

(This test needs `"context"` and the authpkg import — add them. If the keyring on the test machine is available, `SaveToken` may write to the keyring rather than the file; in that case the mtime check is a no-op and the test still guards the redirect logic. Simplify if flaky: assert only that `newCredentialStore()` returns non-nil and that resolving a token with the env set never reads `$HOME/.config/wenmar/credentials.json` by pre-creating a canary token there and asserting it is NOT returned.)

If the keyring-vs-file ambiguity makes this test flaky in CI, replace the mtime check with the canary approach:

```go
func TestWenmarConfigHomeRedirectsCredentials(t *testing.T) {
	// Canary: a token at the real path must NOT be visible when
	// WENMAR_CONFIG_HOME points elsewhere.
	home, _ := os.UserHomeDir()
	realDir := filepath.Join(home, ".config", "wenmar")
	if err := os.MkdirAll(realDir, 0o700); err == nil {
		_ = os.WriteFile(filepath.Join(realDir, "credentials.json"),
			[]byte(`{"access_token":"REAL-TOKEN-CANARY"}`), 0o600)
	}

	t.Setenv("WENMAR_CONFIG_HOME", t.TempDir())
	store := newCredentialStore()
	tok, err := store.GetToken(context.Background())
	if err == nil && tok != nil && tok.AccessToken == "REAL-TOKEN-CANARY" {
		t.Error("store read the real credentials file — WENMAR_CONFIG_HOME redirect failed")
	}
}
```

- [ ] **Step 9: Run everything and verify no real-credential access**

```bash
go test ./... -count=1
ls -la ~/.config/wenmar/ 2>/dev/null   # timestamps unchanged from before the run
```

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "fix: WENMAR_CONFIG_HOME isolates test credentials from the real keyring/config"
```

### Task 12: TUI core-loop fixes

**Files:**
- Modify: `internal/tui/app.go` (tick re-arm, message routing, updateOnline dedup)
- Modify: `internal/tui/topbar.go` (real debounce)
- Modify: `internal/tui/list.go` (rune-safe truncate)
- Modify: `internal/tui/list_workorders.go` (Updated column, status padding)
- Modify: `internal/tui/layout.go` (dead contentWidth block)
- Test: `internal/tui/app_test.go`, `internal/tui/list_test.go`

- [ ] **Step 1: Failing tests**

Add to `internal/tui/app_test.go`:

```go
func TestTickReArmsAfterTickMsg(t *testing.T) {
	m := NewApp(nil, "", 10*time.Second)
	// Init arms the first tick.
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned nil cmd")
	}
	// Deliver a tickMsg — the returned command must include a new tick.
	newM, newCmd := m.Update(tickMsg{})
	if newCmd == nil {
		t.Fatal("tickMsg did not schedule the next tick — periodic refresh dies after one poll")
	}
	_ = newM
}

func TestResultMsgReachesTabWhileSearchFocused(t *testing.T) {
	m := NewApp(nil, "", 10*time.Second)
	m.layout.topBar.FocusSearch()

	// A fetch result arriving while search is focused must still be
	// delivered to the active tab (currently swallowed by the topbar
	// early-return).
	before := m.tabs[0]
	m.Update(workOrderListResultMsg{}) // zero-value msg; type is what matters
	if _, same := m.tabs[0].(WorkOrderList); same && m.tabs[0] == before {
		// If the tab is a pointer, identity comparison can't detect the
		// update; the test's real assertion is that Update did NOT return
		// early — see the routing fix. Use a sentinel: an error result
		// must set the tab's error state.
	}
	// Stronger assertion: error result must surface in the tab.
	m2 := NewApp(nil, "", 10*time.Second)
	m2.layout.topBar.FocusSearch()
	m2.Update(workOrderListResultMsg{err: errors.New("boom")})
	// The active tab (WorkOrderList) must show the error, not "Loading...".
	if wo, ok := m2.tabs[0].(WorkOrderList); ok {
		_ = wo
		// Access via the list's error field through the tab interface's
		// View: after the fix, View renders "Error: boom".
		v := m2.tabs[0].View(80)
		if !strings.Contains(v, "boom") && !strings.Contains(v, "Error") {
			t.Errorf("error result swallowed while search focused; view:\n%s", v)
		}
	}
}
```

Add to `internal/tui/list_test.go`:

```go
func TestTruncateMultibyteSafe(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"José", 10, "José"},                        // fits
		{"Zoë", 2, "Z…"},                            // cut on rune boundary
		{"日本語テキスト", 4, "日本…"},                    // CJK cut
		{"abc", 3, "abc"},                           // exact fit unchanged
		{"ab", 0, ""},                               // max<=0 must not panic
	}
	for _, tc := range cases {
		if got := truncate(tc.in, tc.max); got != tc.want {
			t.Errorf("truncate(%q,%d) = %q, want %q", tc.in, tc.max, got, tc.want)
		}
	}
}
```

(If the WorkOrderList result-message struct field name differs — check `list_workorders.go` for `workOrderListResultMsg` — adjust the test to the real field names. Check `errors` import needs.)

- [ ] **Step 2: Verify they fail**

```bash
go test ./internal/tui/ -run "TestTickReArms|TestResultMsgReaches|TestTruncate" -v
```

Expected: all three FAIL (tick not re-armed; result swallowed; truncate slices bytes).

- [ ] **Step 3: Fix app.go**

3a. Tick re-arm (app.go:121-123):

```go
	case tickMsg:
		// Refresh only the active tab to reduce API load, and re-arm
		// the next tick so periodic polling continues.
		return m, tea.Batch(m.tabs[m.active].Init(), tick(m.interval))
```

3b. Message routing (app.go:128-140) — only KeyMsg goes to topbar/sidebar; everything else reaches the active tab:

```go
	// Route key messages through the topbar/sidebar focus rules; all other
	// messages (fetch results, ticks, filter changes) always reach the
	// active tab — the old early-returns swallowed results while search
	// was focused or the sidebar was open, leaving lists stuck on
	// "Loading...".
	if k, ok := msg.(tea.KeyMsg); ok {
		if m.layout.topBar.searchFocused {
			var cmd tea.Cmd
			m.layout.topBar, cmd = m.layout.topBar.Update(k)
			return m, cmd
		}
		if m.layout.sidebar.visible {
			var cmd tea.Cmd
			m.layout.sidebar, cmd = m.layout.sidebar.Update(k)
			return m, cmd
		}
	}

	// Delegate non-key messages to the active tab.
	updated, cmd := m.tabs[m.active].Update(msg)
	m.tabs[m.active] = updated
	return m, cmd
```

(This replaces the old "if search focused route everything" blocks at app.go:128-140. Key handling when NOT focused/search/sidebar remains in `updateKey` — verify no double-handling: `updateKey` handles KeyMsg before this point via the type switch at line 119-120, so the new block only fires for keys when search-focused or sidebar-visible. Check the switch: `case tea.KeyMsg: return m.updateKey(msg)` — updateKey already routes to topbar when searchFocused (app.go:171-175). So the correct minimal fix is: DELETE the early-return blocks at 128-140 entirely and let the fallthrough delegation handle non-key messages; updateKey already routes keys correctly. Implement it that way:)

```go
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	case tickMsg:
		return m, tea.Batch(m.tabs[m.active].Init(), tick(m.interval))
	case workOrderListResultMsg, customerListResultMsg, vehicleListResultMsg:
		m.updateOnline(msg)
	}

	// All non-key messages (fetch results, filter changes, tick payloads)
	// reach the active tab. The previous search/sidebar early-returns
	// swallowed result messages, leaving lists stuck on "Loading...".
	updated, cmd := m.tabs[m.active].Update(msg)
	m.tabs[m.active] = updated
	return m, cmd
```

3c. updateOnline dedup (app.go:149-167) — the switch falls through to the tab delegation after updating online state; results must ALSO reach the tab. After `m.updateOnline(msg)` inside the case, do NOT return; let execution fall to the delegation at the bottom. (The current code already falls through — verify: the case at 124-125 calls updateOnline and then falls to the early-return blocks; with those deleted it now falls to the tab delegation. Correct.)

3d. `updateOnline` body dedup:

```go
// updateOnline updates the shared footer status based on a fetch result.
func (m *AppModel) updateOnline(msg tea.Msg) {
	type resultMsg interface{ err() error }
	switch msg.(type) {
	case workOrderListResultMsg, customerListResultMsg, vehicleListResultMsg:
		// The three types share shape; extract err via a small helper.
	}
	// simpler: keep the type switch but share the body:
	switch msg := msg.(type) {
	case workOrderListResultMsg:
		m.applyOnline(msg.err)
	case customerListResultMsg:
		m.applyOnline(msg.err)
	case vehicleListResultMsg:
		m.applyOnline(msg.err)
	}
}

func (m *AppModel) applyOnline(err error) {
	m.online = err == nil
	if err == nil {
		m.lastRefresh = time.Now()
	}
}
```

(Check the result-msg struct field is exported/lowercase `err` within the package — they are same-package types, so lowercase `err` works. Adjust to the real field name found in list_workorders.go.)

- [ ] **Step 4: Fix topbar.go debounce (emitFilter, lines 86-93)**

Every keystroke currently schedules its own 300ms tick → a refetch per keystroke. Make stale ticks inert by comparing the captured query against the field's value at delivery time:

```go
// emitFilter returns a command that emits a searchFilterMsg after a short
// debounce. The message captures the query at emit time; AppModel ignores
// it if the field has since changed, so only the last keystroke's query
// triggers a refetch.
func (m TopBarModel) emitFilter() tea.Cmd {
	query := m.search.Value()
	return tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
		return searchFilterMsg{query: query}
	})
}
```

And in app.go's searchFilterMsg handling (lines 105-112), skip if stale:

```go
	if f, ok := msg.(searchFilterMsg); ok {
		if cl, ok := m.tabs[m.active].(*CustomerList); ok {
			if cl.SearchQuery() == f.query {
				return m, nil // stale debounce tick — field already at this query
			}
			cl.SetSearchQuery(f.query)
			cl.startLoading()
			return m, cl.Init()
		}
		return m, nil
	}
```

(Check CustomerList has `SearchQuery()`/`SetSearchQuery` methods — read list_customers.go and adjust names. The stale check compares the tab's current filter to the message's query; if equal, either it's a duplicate or already applied — skip. This makes typing "jane" emit one refetch (for the final state after 300ms of quiet) instead of four. Rationale: each keystroke's tick fires; all but the last see a stale query... wait, each tick carries the query AT ITS keystroke, so ticks for "j","ja","jan","jane" all arrive; the tab's current filter is the PREVIOUS state, so each tick's query differs from the stored filter → each refetches. The comparison against the FIELD value, not the tab filter, is required: capture the topbar's value at delivery. Since the command closure captures only the query, compare `f.query` to `m.layout.topBar.SearchValue()` — the field value NOW:)

```go
	if f, ok := msg.(searchFilterMsg); ok {
		// Ignore stale debounce ticks: only refetch if the field still
		// holds the query this tick captured.
		if m.layout.topBar.SearchValue() != f.query {
			return m, nil
		}
		if cl, ok := m.tabs[m.active].(*CustomerList); ok {
			cl.SetSearchQuery(f.query)
			cl.startLoading()
			return m, cl.Init()
		}
		return m, nil
	}
```

- [ ] **Step 5: Fix list.go truncate (lines 45-50)**

```go
// truncate shortens s to at most max runes, appending an ellipsis when cut.
// max <= 0 returns the empty string (never panics on negative slicing).
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
```

- [ ] **Step 6: Fix list_workorders.go Updated column + status padding**

Read the current row renderer (list_workorders.go:107-113). Change the Updated column to render `wo.UpdatedAt` (parse its format — check the WorkOrder type in the SDK: `go doc github.com/wenmar-pro/wenmar-sdk/go/wenmar.WorkOrder` — and format it as `15:04:05` like the old code). And pad the raw status BEFORE styling:

```go
	status := fmt.Sprintf("%-12s", wo.Status)
	return []string{fmt.Sprintf("%-8d %-20s %-20s %-12s %-15s",
		wo.Id, truncate(stringify(wo.Number), 20), ...,
		statusColored(status), wo.UpdatedAt.Format("15:04:05"))}
```

(Adjust to the actual field names/types from the SDK doc — the exact current row format string is at list_workorders.go:107-113; keep its column order, change only: raw-status padding pre-styling and the Updated column source. If `wo.UpdatedAt` is a string, keep parsing minimal: display it as-is, but sourced from the record, not `m.refreshed`.)

- [ ] **Step 7: Delete the dead layout block (layout.go:54-58)**

Remove the `contentWidth` computation that is computed and discarded:

```go
	_ = contentWidth
```

Delete those five lines.

- [ ] **Step 8: Run TUI tests**

```bash
go test ./internal/tui/ -v
```

Expected: all pass, including the three new tests. If existing tests depended on the old swallow-behavior (they might — the review found app_test.go:168 relies on tab mutation), update them in the same commit with a note.

- [ ] **Step 9: Commit**

```bash
git add internal/tui/
git commit -m "fix(tui): re-arm refresh tick, route results past search/sidebar, real debounce, rune-safe truncate"
```

### Task 13: Atomic config writes + safe migration

**Files:**
- Modify: `internal/config/config.go` (SaveTo atomic, 0700 dir)
- Modify: `internal/config/xdg.go` (migration order)
- Test: `internal/config/config_test.go`, `internal/config/xdg_test.go` (create if missing)

- [ ] **Step 1: Failing tests**

Add to `internal/config/config_test.go`:

```go
func TestSaveToIsAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wenmar", "config")
	if err := SaveTo(path, &Config{Token: "t", BaseURL: "https://x"}); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}
	// No temp file left behind.
	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
	// Overwrite succeeds and old content is fully replaced.
	if err := SaveTo(path, &Config{Token: "t2"}); err != nil {
		t.Fatalf("SaveTo overwrite: %v", err)
	}
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Token != "t2" {
		t.Errorf("Token = %q, want t2", cfg.Token)
	}
}

func TestSaveToDirPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wenmar", "config")
	if err := SaveTo(path, &Config{}); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}
	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Errorf("config dir mode = %o, want 700", info.Mode().Perm())
	}
}
```

Migration-failure injection (create `internal/config/xdg_test.go` if missing):

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateOldConfigFailureKeepsOldData(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("WENMAR_CONFIG_HOME", cfgHome)

	// Legacy config with a token.
	home := t.TempDir()
	t.Setenv("HOME", home) // oldConfigPath uses os.UserHomeDir
	legacyDir := filepath.Join(home, ".wenmar")
	os.MkdirAll(legacyDir, 0o700)
	oldPath := filepath.Join(legacyDir, "config")
	oldData := []byte("token: legacy-secret\n")
	os.WriteFile(oldPath, oldData, 0o600)

	// Make the NEW location unwritable to force the write failure.
	// (WENMAR_CONFIG_HOME exists; corrupt it by making it a file.)
	blocker := filepath.Join(cfgHome, "wenmar")
	os.WriteFile(blocker, []byte("not a dir"), 0o600)

	_, err := migrateOldConfig(oldPath)
	if err == nil {
		t.Fatal("expected migration failure")
	}
	// The old file must be intact.
	got, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatalf("old config deleted during failed migration: %v", err)
	}
	if string(got) != string(oldData) {
		t.Errorf("old config corrupted: %q", got)
	}
}
```

(Note: `oldConfigPath` uses `os.UserHomeDir()` which reads `$HOME` on Unix — `t.Setenv("HOME", ...)` works. The blocker trick: making `cfgHome/wenmar` a regular file makes `MkdirAll`/write fail. This test verifies the ordering fix: no empty config at the new path unless the real data landed there.)

- [ ] **Step 2: Verify tests fail**

```bash
go test ./internal/config/ -v
```

Expected: `TestSaveToDirPermissions` FAILS (0755 today); `TestMigrateOldConfigFailureKeepsOldData` FAILS (current code writes empty config first, then real data — with the blocker, the empty write succeeds (dir exists as file → SaveTo's MkdirAll fails → migration aborts BEFORE writing anything — wait, current flow: SaveTo(newPath, &Config{}) → MkdirAll fails because blocker is a file → returns error → migration returns error → old data safe. So current code PASSES this test. The test's real target is the sequence where MkdirAll succeeds but the second write fails. Strengthen: make the FIRST write succeed and the SECOND fail — that requires the same dir, so simulate by checking the intermediate state differently. Simpler, keep the test as the blocker version (it guards the old-file-removal ordering) and add the core ordering fix: the code must never write an empty config first. The direct fix removes the empty-config write entirely, so the intermediate window disappears — the blocker test then exercises "nothing written at all, old intact". Keep both tests; after the fix both pass for the right reasons.)

- [ ] **Step 3: Implement atomic SaveTo**

In `internal/config/config.go`, replace SaveTo (lines 87-103):

```go
func SaveTo(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	// Atomic write: temp file + fsync + rename, so a crash mid-write can
	// never truncate or corrupt the config.
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op after successful rename

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("could not write config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("could not sync config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("could not close config: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("could not set config permissions: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("could not replace config: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Fix migration ordering (xdg.go:50-68)**

Replace the "read old, write new" sequence — delete the empty-config write, write real data atomically, then remove old with error check:

```go
	// Read old, write to new atomically (no empty-config window), then
	// remove the old file.
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return false, fmt.Errorf("could not read old config: %w", err)
	}

	newPath, err := xdgConfigPath()
	if err != nil {
		return false, err
	}

	dir := filepath.Dir(newPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, fmt.Errorf("could not create new config dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".migrate-*.tmp")
	if err != nil {
		return false, fmt.Errorf("could not create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return false, fmt.Errorf("could not write migrated config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return false, fmt.Errorf("could not sync migrated config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return false, fmt.Errorf("could not close migrated config: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return false, fmt.Errorf("could not set permissions: %w", err)
	}
	if err := os.Rename(tmpPath, newPath); err != nil {
		return false, fmt.Errorf("could not move migrated config into place: %w", err)
	}

	// Only remove the old file once the new one is durably in place.
	if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
		return true, fmt.Errorf("migrated config but could not remove old file at %s: %w", oldPath, err)
	}

	return true, nil
```

- [ ] **Step 5: Move the migration notice out of config package**

`ConfigPath` (config.go:47) prints to stderr directly. Change its signature — no: keep the signature, remove the print, and return migration status via a package-level var is worse. Better: change `ConfigPath()` to return `(string, error)` as today but add a separate exported report: change `migrateOldConfig` to be called from a new exported `MigrateAndReport() (bool, error)`? Simplest correct fix preserving all callers: keep ConfigPath as-is except delete the Fprintf line and the `migrated` branch — instead, make `Load()` call migration and log to stderr from the CMD layer is a bigger ripple. Pragmatic: leave the stderr print in place but route it through an injectable package-level `var Notify func(string)` defaulting to stderr writer. Implement:

```go
// Notify receives migration notices. Defaults to stderr; replaced in tests.
var Notify = func(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}
```

and use `Notify("  Migrated config from ~/.wenmar/ to ~/.config/wenmar/\n")` in ConfigPath. Tests can silence/observe it.

- [ ] **Step 6: Run config tests**

```bash
go test ./internal/config/ -v
```

- [ ] **Step 7: Commit**

```bash
git add internal/config/
git commit -m "fix(config): atomic writes with fsync, 0700 dirs, no empty-config migration window"
```

### Task 14: Watch resilience (backoff, clean SIGINT, decode errors)

**Files:**
- Modify: `internal/watch/poller.go` (retry/backoff, decode error, drop mutex, scoped-client reuse)
- Modify: `cmd/watch.go` (SIGINT clean exit, script exit status, Encode check)
- Test: `internal/watch/poller_test.go`, `cmd/watch_test.go`

- [ ] **Step 1: Failing tests**

Add to `internal/watch/poller_test.go` (read the existing helpers `newTestClient`/`TestPoller_EmitsNewItems` first and reuse their fake-server pattern):

```go
func TestPoller_TransientErrorRetries(t *testing.T) {
	var polls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&polls, 1)
		if n == 2 {
			// One transient failure mid-stream.
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, []map[string]any{{"id": 1}})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL, "tok")
	p := &Poller{
		Client:     client,
		Resource:   "customers",
		Interval:   10 * time.Millisecond,
		ExitOnFirst: false,
	}
	var events int32
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	err := p.Run(ctx, func(Event) { atomic.AddInt32(&events, 1) })
	_ = err
	if atomic.LoadInt32(&polls) < 3 {
		t.Errorf("poller gave up after transient 500: only %d polls", polls)
	}
}

func TestPoller_FatalAuthErrorStops(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "bad token")
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL, "tok")
	p := &Poller{Client: client, Resource: "customers", Interval: 10 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := p.Run(ctx, func(Event) {})
	if err == nil {
		t.Error("auth failure must be fatal, not retried")
	}
}
```

(Adjust `writeError`/`writeJSON` helpers to whatever `poller_test.go` already defines — read it first. `newTestClient(t, url, token)` exists per the review.)

- [ ] **Step 2: Verify they fail**

```bash
go test ./internal/watch/ -v
```

Expected: `TransientErrorRetries` FAILS (one 500 ends the loop today); `FatalAuthErrorStops` may pass already (Run returns the error) — keep it as a guard.

- [ ] **Step 3: Implement retry/backoff in poller.go**

Add to Poller:

```go
	// MaxConsecutiveFailures is how many consecutive transient errors are
	// tolerated before giving up (0 = default 3).
	MaxConsecutiveFailures int
```

Rework Run's loop:

```go
func (p *Poller) Run(ctx context.Context, emit func(Event)) error {
	p.previous = nil

	scoped, err := p.scopedClient(ctx)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()

	failures := 0
	maxFailures := p.MaxConsecutiveFailures
	if maxFailures <= 0 {
		maxFailures = 3
	}
	backoff := time.Second

	poll := func() error {
		if err := p.poll(ctx, emit, scoped); err != nil {
			if isAuthError(err) {
				return err // fatal
			}
			failures++
			if failures >= maxFailures {
				return fmt.Errorf("watch: %d consecutive failed polls, giving up: %w", failures, err)
			}
			// Brief backoff, reset on next success.
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			if backoff < 60*time.Second {
				backoff *= 2
			}
			return nil
		}
		failures = 0
		backoff = time.Second
		return nil
	}

	if err := poll(); err != nil {
		return err
	}
	if p.ExitOnFirst {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := poll(); err != nil {
				return err
			}
		}
	}
}

func (p *Poller) scopedClient(ctx context.Context) (*wenmar.Client, error) {
	if p.LocationID == "" {
		return p.Client, nil
	}
	lc, err := p.Client.ForLocation(ctx, p.LocationID)
	if err != nil {
		return nil, err
	}
	return lc.Client, nil
}

func isAuthError(err error) bool {
	var apiErr *wenmar.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 401 || apiErr.StatusCode == 403
	}
	return false
}
```

Change `poll(ctx, emit)` to `poll(ctx, emit, client *wenmar.Client)` and use the passed client instead of calling ForLocation per poll. Delete the `mu sync.Mutex` and its Lock/Unlock calls (single-goroutine loop; the mutex implies concurrency guarantees that don't exist). Add `"errors"` import. The per-poll `ForLocation` (poller.go:120-127) moves to `scopedClient` — done above.

- [ ] **Step 4: decodeList must surface round-trip failures**

Change decodeList's signature to return an error:

```go
func decodeList[T any](list *[]T) ([]map[string]any, error) {
	if list == nil {
		return nil, nil
	}
	items := make([]map[string]any, 0, len(*list))
	for _, item := range *list {
		b, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("decode item: %w", err)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("decode item: %w", err)
		}
		items = append(items, m)
	}
	return items, nil
}
```

Update fetch's call sites: `items, err := decodeList(resp.JSON200); if err != nil { return nil, err }` in all three resource branches. (Silently dropping an item today creates phantom new/removed events on the next poll.)

- [ ] **Step 5: Clean SIGINT in cmd/watch.go**

In `runWatch`, wrap the Run error:

```go
	err := poller.Run(ctx, func(e watch.Event) { ... })
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil // Ctrl-C is a clean stop, not an error
	}
	return err
```

Add `"errors"` import (currently `"errors"` is NOT imported in watch.go — check: it imports context/json/os/exec/os/signal/strings/syscall/time — add errors).

- [ ] **Step 6: Script exit status + Encode check in cmd/watch.go**

Replace runSyncScript (lines 128-136):

```go
func runSyncScript(script string, e watch.Event) (err error) {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	cmd := exec.Command(script)
	cmd.Stdin = strings.NewReader(string(data))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
```

In runWatch's emit callback:

```go
	return poller.Run(ctx, func(e watch.Event) {
		if watchRunSync != "" {
			if err := runSyncScript(watchRunSync, e); err != nil {
				fmt.Fprintf(os.Stderr, "watch: script %q failed: %v\n", watchRunSync, err)
			}
			return
		}
		if watchRunAsync != "" {
			go func(ev watch.Event) {
				if err := runSyncScript(watchRunAsync, ev); err != nil {
					fmt.Fprintf(os.Stderr, "watch: async script %q failed: %v\n", watchRunAsync, err)
				}
			}(e)
			return
		}
		if err := enc.Encode(e); err != nil {
			fmt.Fprintf(os.Stderr, "watch: encode event: %v\n", err)
		}
	})
```

- [ ] **Step 7: Run tests**

```bash
go test ./internal/watch/ -v && go test ./cmd/ -run TestWatch -v && go test ./...
```

(If poller_test.go's `TestPoller_ExitOnFirst` uses the old poll signature, update it. Timing-based tests use tiny intervals — keep the existing pattern.)

- [ ] **Step 8: Commit**

```bash
git add internal/watch/ cmd/watch.go
git commit -m "fix(watch): retry transient poll errors with backoff, clean Ctrl-C exit, surface decode/script errors"
```

### Task 15: Exit-code fallbacks + hints

**Files:**
- Modify: `internal/errors/exit.go` (status fallbacks, offline detection)
- Modify: `internal/errors/debug.go` (conflict/forbidden hints)
- Test: `internal/errors/exit_test.go`

- [ ] **Step 1: Failing tests**

Append to `internal/errors/exit_test.go`:

```go
func TestExitCode_StatusFallbacksWhenCodeUnrecognized(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		want   int
	}{
		{"401 unrecognized code", &wenmar.APIError{Code: "weird_proxy", StatusCode: 401}, 2},
		{"404 unrecognized code", &wenmar.APIError{Code: "", StatusCode: 404}, 3},
		{"422 unrecognized code", &wenmar.APIError{Code: "", StatusCode: 422}, 4},
		{"429 unrecognized code", &wenmar.APIError{Code: "slow_down", StatusCode: 429}, 5},
		{"502 unrecognized code", &wenmar.APIError{Code: "", StatusCode: 502}, 6},
		{"409 unrecognized code", &wenmar.APIError{Code: "dup", StatusCode: 409}, 7},
		{"403 unrecognized code", &wenmar.APIError{Code: "nope", StatusCode: 403}, 8},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCode(tc.err); got != tc.want {
				t.Errorf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

func TestExitCode_ConnectionRefusedIsOffline(t *testing.T) {
	// *url.Error wrapping ECONNREFUSED must map to 10, not 1.
	err := &url.Error{
		Op:  "Get",
		URL: "http://127.0.0.1:1/customers",
		Err: &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: syscall.ECONNREFUSED,
		},
	}
	if got := ExitCode(err); got != ExitOffline {
		t.Errorf("ExitCode(ECONNREFUSED) = %d, want %d", got, ExitOffline)
	}
}
```

Add `"net/url"`, `"syscall"` imports to the test file. Note: `net.OpError` implements `net.Error` (it has Timeout() bool), so `errors.As(err, &netErr)` should already catch it — verify by running; if it passes already, it stays as a regression guard. The genuinely-broken case is errors not implementing net.Error; keep the test regardless.

- [ ] **Step 2: Verify fallbacks test fails**

```bash
go test ./internal/errors/ -run TestExitCode_StatusFallbacks -v
```

Expected: 401/404/422/429 cases FAIL (fall to generic 1 today); 502/403/409 pass already.

- [ ] **Step 3: Implement status fallbacks (exit.go:59-69)**

Replace the default branch:

```go
		default:
			// Status-code fallbacks keep the documented exit-code contract
			// intact when the server sends an unrecognized error Code.
			switch apiErr.StatusCode {
			case 401:
				return ExitAuth
			case 404:
				return ExitNotFound
			case 422:
				return ExitValidation
			case 429:
				return ExitRateLimit
			case 403:
				return ExitForbidden
			case 409:
				return ExitConflict
			}
			if apiErr.StatusCode >= 500 {
				return ExitServer
			}
			return ExitGeneric
```

- [ ] **Step 4: Add hints for conflict/forbidden (debug.go:77-93)**

```go
	case "conflict":
		fmt.Fprintln(w, "  Hint: the resource already exists (e.g. a duplicate VIN). Use a different value or check for an existing record.")
	case "forbidden":
		fmt.Fprintln(w, "  Hint: your token does not grant access to this resource or action.")
```

Insert before the `default:` in printHints. Also add status fallbacks mirroring exit.go's:

```go
	default:
		switch apiErr.StatusCode {
		case 409:
			fmt.Fprintln(w, "  Hint: the resource already exists (e.g. a duplicate VIN).")
		case 403:
			fmt.Fprintln(w, "  Hint: your token does not grant access to this resource or action.")
		}
		if apiErr.StatusCode >= 500 {
			fmt.Fprintln(w, "  Hint: the server reported an error. Check the server logs or retry later.")
		}
	}
```

- [ ] **Step 5: Run errors tests + exit-code table test for all 11 codes**

```bash
go test ./internal/errors/ -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/errors/
git commit -m "fix(errors): status fallbacks close the exit-code contract; conflict/forbidden hints"
```

### Task 16: Error hygiene sweep (enc.Encode, auth status, OAuth, id errors)

**Files:**
- Modify: `cmd/root.go:126-128` (done in Task 10 — verify), `cmd/watch.go` (done in Task 14 — verify)
- Modify: `cmd/auth.go:155-185` (runAuthStatus no os.Exit), `:203` (dead assignment), `:147` (logout surfaces delete error)
- Modify: `cmd/upgrade.go:112,130,207`, `cmd/setup.go:171`, `cmd/doctor.go:108` (ignored errors)
- Modify: `cmd/runners.go:22,56,235,263` (id error includes value)
- Modify: `internal/auth/oauth/pkce.go` (no panic), `internal/auth/oauth/callback.go:47` (no state echo), `internal/auth/oauth/flow.go:65-75` (bounded exchange)
- Test: `cmd/auth_test.go`, `internal/auth/oauth/pkce_test.go`

- [ ] **Step 1: Fix runAuthStatus (auth.go:155-185)**

Remove `os.Exit(2)` mid-function — return a typed error; stop double-printing:

```go
func runAuthStatus(out io.Writer, configPath string) error {
	rt, err := auth.ResolveTokenWithSourceFrom(tokenFlag, configPath, newCredentialStore())
	if err != nil {
		fmt.Fprintln(out, "  Not logged in. Run `wenmar auth login` to configure.")
		return errNotLoggedIn{err}
	}
	// ... rest unchanged, but the final connection-failure path must not
	// both print AND return the raw error (double print). Keep the print,
	// return a wrapped silent error:
	_, err = client.ListAccount(context.Background())
	if err != nil {
		fmt.Fprintln(out, " ✗")
		fmt.Fprintf(out, "  Connection failed: %v\n", err)
		return fmt.Errorf("connection test failed")
	}
	fmt.Fprintln(out, " ✓")
	fmt.Fprintln(out, "  Connected.")
	return nil
}

// errNotLoggedIn marks "no token" so ExitCode can map it to 2 and the
// root printer shows the friendly line instead of the raw resolver error.
type errNotLoggedIn struct{ cause error }

func (e errNotLoggedIn) Error() string { return "not logged in: " + e.cause.Error() }
func (e errNotLoggedIn) Unwrap() error { return e.cause }
```

Map it in `internal/errors/exit.go`:

```go
	var nli auth.ErrNotLoggedIn // or a local sentinel — see below
```

Simplest contract-safe approach without import cycles: define the sentinel in `internal/errors`:

```go
// ErrNotLoggedIn marks a "no token configured" failure (exit 2).
var ErrNotLoggedIn = errors.New("not logged in")
```

auth.go wraps: `return fmt.Errorf("%w: %v", errors.ErrNotLoggedIn, err)`. exit.go adds before the APIError check:

```go
	if errors.Is(err, ErrNotLoggedIn) {
		return ExitAuth
	}
```

And PrintError special-cases it to print only the friendly line (check `errors.Is` in PrintError and skip the debug block). Update `cmd/auth_test.go`'s status tests if they asserted os.Exit behavior (they can now assert the returned error).

- [ ] **Step 2: Sweep the ignored errors**

- `cmd/auth.go:203` — delete `_ = manager` line.
- `cmd/auth.go:147` — logout: replace `_ = store.DeleteToken(...)` with a checked call that reports but does not fail the logout (idempotent):
  ```go
  if err := store.DeleteToken(context.Background()); err != nil {
      fmt.Fprintf(os.Stderr, "  warning: could not delete stored token: %v\n", err)
  }
  ```
- `cmd/upgrade.go:112` — `exe, err := filepath.EvalSymlinks(exe); if err != nil { /* keep original exe */ }` (don't blank-assign to zero).
- `cmd/upgrade.go:130` and `cmd/setup.go:171`, `cmd/doctor.go:108` — check `os.UserHomeDir()` errors and fail with a clear message.
- `cmd/upgrade.go:207` — check `os.Chmod(binPath, 0o755)`; on failure, warn that the binary may not be executable.
- `cmd/runners.go:22,56,235,263` — `parseInt` and the inline Atoi calls: change the error to `fmt.Errorf("id must be an integer, got %q", s)`; route all inline `strconv.Atoi(args[0])` in runners through `parseInt(args[0])` so the message is consistent (runners.go:54, 233, 261).

- [ ] **Step 3: OAuth hardening**

`internal/auth/oauth/pkce.go` — replace panic with error:

```go
func GenerateVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand failed: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
```

(and the same for GenerateState; update callers in flow.go — read flow.go first; keep signatures consistent.)

`internal/auth/oauth/callback.go:47` — stop echoing the expected state:

```go
	resultCh <- result{err: errors.New("state mismatch: callback state did not match the login request (possible CSRF); retry the login")}
```

`internal/auth/oauth/flow.go:65-75` — bound the exchange: pass the callbackCtx (or a fresh bounded context) to ExchangeCode instead of the parent ctx:

```go
	callbackCtx, cancel := context.WithTimeout(ctx, LoginTimeout)
	defer cancel()
	code, err := WaitForCallback(callbackCtx, state, listener)
	...
	exchangeCtx, cancelExchange := context.WithTimeout(ctx, 30*time.Second)
	defer cancelExchange()
	token, err := ExchangeCode(exchangeCtx, tokenEndpoint, code, redirectURI, clientID, verifier)
```

(Read flow.go's actual structure first and preserve its shutdown ordering — listener closed before/after exchange; keep behavior, only bound the exchange timeout.)

Update `internal/auth/oauth/pkce_test.go` — tests calling GenerateVerifier must now handle the error (read pkce_test.go and adjust call sites).

- [ ] **Step 4: Run tests**

```bash
go test ./internal/auth/... -v && go test ./cmd/ -v
```

- [ ] **Step 5: Commit**

```bash
git add cmd/ internal/auth/
git commit -m "fix: error hygiene sweep (no swallowed errors, no panics, no os.Exit in library code, bounded OAuth exchange)"
```

### Task 17: Auth/output consolidation

**Files:**
- Modify: `internal/auth/auth.go` (dedup resolution chains, delete keyringToken)
- Modify: `cmd/setup.go` (delete duplicate maskToken)
- Modify: `internal/output/output.go` (delete CaptureBreadcrumbs/joinArgs, typed envelope)
- Test: existing suites must stay green

- [ ] **Step 1: Dedup the resolution chain**

`ResolveAuthManagerWithStore` (auth.go:106-145) re-implements the precedence of `ResolveTokenWithSourceFrom` (auth.go:57-78). Rewrite the manager path to consult the resolver:

```go
func ResolveAuthManagerWithStore(flagToken, configPath string, store authpkg.CredentialStore) (*authpkg.AuthManager, error) {
	rt, err := ResolveTokenWithSourceFrom(flagToken, configPath, store)
	if err != nil {
		return nil, err
	}
	switch rt.Source {
	case SourceFlag, SourceEnv, SourceConfig:
		return authpkg.NewAuthManager(store, authpkg.NewStaticTokenProvider(rt.Token)), nil
	case SourceKeyring:
		// Keyring / file credential store with auto-refresh.
		manager := authpkg.NewAuthManager(store, nil)
		provider := &authpkg.CredentialStoreProvider{Store: store, Manager: manager}
		manager.Provider = provider
		if tok, err := store.GetToken(context.Background()); err == nil && tok != nil && tok.RefreshToken != "" {
			baseURL := ResolveBaseURLFrom("", configPath)
			if cfg, err := config.LoadFrom(configPath); err == nil && cfg.AuthMethod == "oauth" {
				manager.SetRefreshFn(func(ctx context.Context, refreshToken string) (*authpkg.Token, error) {
					return authpkg.RefreshToken(ctx, baseURL+"/oauth/token", "wenmar-cli", refreshToken)
				})
			}
		}
		return manager, nil
	}
	return nil, fmt.Errorf("API token required. Run `wenmar setup` to configure, or set --token / WENMAR_TOKEN env var")
}
```

Note: this requires `ResolvedToken` to carry enough info — it does (Token + Source). One behavioral nuance: the old manager path checked `tok.RefreshToken` to wire refresh; the resolver path must preserve that — handled by the re-fetch inside SourceKeyring. Keep the keyring-token check consistent with the resolver's `keyringTokenFrom`.

Also delete the dead `keyringToken` (auth.go:82-84) after confirming no callers:

```bash
grep -rn "keyringToken\b" cmd/ internal/
```

- [ ] **Step 2: Delete the duplicate maskToken**

```bash
grep -rn "func maskToken\|maskToken(" cmd/
```

Delete `maskToken` in `cmd/setup.go:146-151`; switch its callers (`cmd/auth.go:163`, `cmd/config.go` — find via grep) to `errors.MaskToken`. Import `internal/errors` where needed.

- [ ] **Step 3: Delete dead CaptureBreadcrumbs + typed envelope**

In `internal/output/output.go`, delete `CaptureBreadcrumbs` and `joinArgs` (lines 92-110) — verify no callers:

```bash
grep -rn "CaptureBreadcrumbs\|joinArgs" cmd/ internal/
```

In `internal/output/json.go`, replace the map-based envelope with a struct for stable field order:

```go
// Envelope is the JSON envelope for --json output.
type Envelope struct {
	OK          bool          `json:"ok"`
	Data        any           `json:"data,omitempty"`
	Summary     string        `json:"summary,omitempty"`
	Meta        *Meta         `json:"meta,omitempty"`
	Breadcrumbs []Breadcrumb  `json:"breadcrumbs,omitempty"`
	Notice      string        `json:"notice,omitempty"`
}
```

Read json.go first; keep the existing field set exactly (drop nothing — check what renderJSON currently emits, including any fields this struct misses). Adjust renderJSON to build the struct.

- [ ] **Step 4: Run the full suite**

```bash
go test ./... -count=1
```

All existing tests must stay green — especially the integration tests asserting envelope JSON shape (they may assert key order as a string; the struct changes ordering to declaration order: ok, data, summary, meta, breadcrumbs — matches the documented order in help_topics.go:38-40).

- [ ] **Step 5: Commit**

```bash
git add internal/ cmd/
git commit -m "refactor: single token-resolution chain, one maskToken, typed JSON envelope, dead code removal"
```

### Task 18: Update SKILL.md + README for Phase 1 reality

**Files:**
- Modify: `skills/wenmar/SKILL.md`
- Modify: `README.md`

Phase 1 changes surface behavior: removed flags, new exit-code guarantees, agent-help topic. The full skill rewrite is Phase 4; this task only removes fabrications and documents Phase 1 deltas.

- [ ] **Step 1: Fix the fabricated claims in SKILL.md**

Read `skills/wenmar/SKILL.md` fully. Apply these specific corrections:

1. Delete/rewrite the "Bulk operations need explicit `--force`" invariant — no bulk operations exist.
2. Remove "unless `--force` is passed" from the delete guidance — no delete command has `--force` (only `--dry-run` on vehicles/drivers/work_orders/service_categories/tags deletes).
3. Replace `work_orders list --status active` / `--overdue` examples with commands that exist (`work_orders list`, then filter via `--jq`).
4. Replace `vehicles list --plate <p> --state <s>` with the real lookup path (`vehicles lookup <query>`, check `cmd/vehicles.go` for the actual flag shape).
5. Fix "Auto-detection is not used" — describe the actual pipe behavior: when stdout is not a TTY and no mode flag is set, wenmar emits raw JSON; `--styled` forces tables.
6. Complete the exit-code table to all 11 entries (0–10), matching `internal/errors/exit.go` and `wenmar help exit-codes`.

- [ ] **Step 2: Update README**

- Remove `--page` from any work_orders example; remove `--remove-email/--remove-address/--address/--tag-id` from any `customers update` example.
- Exit-code table: add the status-fallback guarantee sentence.

- [ ] **Step 3: Verify claims against the binary**

```bash
go build -o wenmar ./cmd/wenmar
./wenmar work_orders list --help          # no --page shown
./wenmar customers update --help         # no removed flags shown
./wenmar help exit-codes                 # 11 entries
```

- [ ] **Step 4: Commit**

```bash
git add skills/wenmar/SKILL.md README.md
git commit -m "docs: SKILL.md and README match Phase 1 CLI behavior; remove fabricated claims"
```

---

## Self-review notes

- **Spec coverage:** Phase 0 (spec §Phase 0) → Tasks 1-3. Phase 1.1 → Task 4. 1.2 → Tasks 5-6 (flags verified against API spec: unsupportable ones removed, per "wire or remove" — removal is the only honest option; `--phone` wired). 1.3 → Task 7. 1.4 → Task 8. 1.5 → Task 9. 1.6 → Task 10. 1.7 → Task 11. 1.8 → Task 12. 1.9 → Task 13. 1.10 → Task 14. 1.11 → Tasks 15-16. 1.12 → Task 17 (+parts of 16). SKILL.md interim fix → Task 18 (full rewrite stays Phase 4). Spec items intentionally deferred to later phases: generic runners, generator cutover, naming, --output, groups/examples, watch TUI polish beyond bug fixes.
- **Known implementation-time reads:** every task that says "read X first" contains the reason (exact field names live in files not fully quoted here). The SDK method names, struct fields, and spec constraints quoted above were verified against the SDK worktree and enriched spec at plan-writing time.
- **Type consistency:** BuildCommandInfo(root, cmd) signature used consistently in Task 10; newCredentialStore() defined in Task 11 and used in Tasks 11/16/17; errNotLoggedIn replaced mid-plan by errors.ErrNotLoggedIn sentinel in Task 16 — the sentinel version is authoritative.