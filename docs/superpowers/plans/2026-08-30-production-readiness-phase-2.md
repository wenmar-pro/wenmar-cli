# Wenmar CLI Production-Readiness — Phase 2 Implementation Plan (Generator Cutover)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-ruby:subagent-driven-development (recommended) or superpowers-ruby:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make gencli the single source of truth for all resource commands — generated files committed as the default build, hand-written resource twins deleted, the `generated` build tag eliminated — with golden tests and a CI regen-drift gate so the CLI surface can never drift from the spec.

**Architecture:** Phase 1 fixed the generator's crash and left hand-written files authoritative. This phase: (1) teach the generator everything the hand-written files know (action handlers, list-with-filters, dry-run, parent-command validation), (2) prove it with golden tests against a fixture spec, (3) flip the build so generated code is canonical, (4) migrate naming to basecamp-cli conventions with permanent aliases, (5) gate the whole thing in CI.

**Tech Stack:** Go 1.27, cobra/pflag, dave/jennifer codegen, wenmar-sdk go/v0.4.0, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-08-30-production-readiness-refactor-design.md` (§Phase 2)
**Prerequisite:** Phase 0 + Phase 1 plans landed (`docs/superpowers/plans/2026-08-30-production-readiness-phase-0-1.md`). Tasks 4 (gencli recursion fix) and 7 (parent Args validation) must be complete — this plan builds on them.

**Conventions:** Run `go build ./... && go test ./...` before every commit. All commands from repo root. Commit style: `feat:`, `fix:`, `test:`, `ci:`, `chore:`, `docs:`.

---

## Verified ground truth (do not re-derive; read this first)

Facts established against the SDK worktree (`../wenmar-sdk/go`, tagged `go/v0.4.0`) and the enriched spec at plan-writing time:

**SDK methods the generator will emit (all verified to exist):**
- `ListCustomersWithPagination(ctx)`, `ListCustomersWithParamsWithPagination(ctx, ListCustomersParams)`, `ListWorkOrdersWithPagination(ctx)` — the ONLY `WithPagination` variants in the SDK.
- `ListVehicles(ctx)`, `ListVendors(ctx)`, `ListDrivers(ctx, customerID)`, `ListStatements(ctx, customerID)`, `ListServiceCategories(ctx)`, `ListTags(ctx)` — plain, no Paginator return. Their spec ops carry NO `x-paginated` marker (only `list_customers`, `list_vehicles`, `list_work_orders` have `x-paginated: true`).
- Service-category actions: `DeactivateServiceCategory(ctx, id)`, `ReactivateServiceCategory(ctx, id)`, `MoveUpServiceCategory(ctx, id)`, `MoveDownServiceCategory(ctx, id)` — each returns `(*XResponse, error)` with `JSON200`.
- `SeedDefaultsServiceCategories(ctx)` returns `(*SeedDefaultsServiceCategoriesResponse, error)` whose `JSON200` has a `Created int` field.
- Nested collections: `ListCustomerVehicles(ctx, customerID)`, `ListCustomerWorkOrders(ctx, customerID)`, `ListCustomerStatements(ctx, customerID)`, `ListVehicleWorkOrders(ctx, vehicleID)` — all return `JSON200` arrays; all live in `wenmar/nested_collections.go`.
- Request types: `CreateServiceCategoryRequest{Name, ServiceType, Icon string}`, `UpdateServiceCategoryRequest{Name string}`, `CreateDriverRequest{FullName, Phone string}`, `UpdateDriverRequest{FullName string}` (phone NOT updatable — check drivers.go:117-119), `CreateCustomerTagRequest{Name}`, `CreateVehicleTagRequest{Name}`, `UpdateTagsRequest` (has `CustomerTags []CustomerTagUpdate` value field + `VehicleTags *[]VehicleTagUpdate` pointer field).

**Spec shapes that drive classification (verified):**
- All four service-category actions are PATCH `/service_categories/{id}/<action>` with an EMPTY requestBody (`type: object, properties: {}`).
- `seed_defaults_service_categories` is POST `/service_categories/seed_defaults` with an empty requestBody and NO `{id}`.
- `update_tags` is PATCH `/settings/tags` with a non-empty body (vehicle_tags/customer_tags arrays) — arrays are skipped by `extractScalarFields`, so BodyFields is empty for it.
- The spec has 76 operationIds total. The current `exclude:` list removes 7.

**Generator internals (verified by reading gen.go end-to-end):**
- `classifyCommand` (gen.go:280-325): PATCH with `{id}` + requestBody + sub-path → `actionUpdate` (service-category actions land here); POST with `{id}` + body + sub-path → `actionCreate` (merge); POST without `{id}` + body → `create`; GET with `{id}` → `show`/`showStr`; `x-paginated` GET without `{id}` → `listPaginated`.
- `emitActionHandler` (gen.go:1086-1090) is the stub returning "action %s not yet generated" — it fires for POST/PATCH with NO body and NO request_struct override. With an empty-object body present, classification lands on `actionUpdate` instead, whose handler (emitActionCreateHandler, gen.go:953-998) requires `cmd.RequestStruct != ""` and returns `nil, nil` body otherwise → the SDK call compiles but sends garbage. This is why the generated build ships broken service-categories today.
- `emitGroup` (gen.go:191-250) emits `//go:build generated` header, flag vars, command vars, handlers, and ONE init() per resource file registering flags + `rootCmd.AddCommand`.
- The `--all` flag exists ONLY in hand-written customers.go (customersListAll bool); the generator has no `--all` support at all.
- `pathFnForRunner` handles both plain `idPath` and extra-path-param Sprintf cases.
- Hand-written parents set `Aliases` (work_orders has `wo`; all lists have `ls`); the generator emits NO aliases today.
- Hand-written delete commands bind `--dry-run` via `runDelete(cmd, args, label, slug, pathFn, dryRunVar, deleter)`. The generator emits `--dry-run` flags (emitFlagRegistration gen.go:416-425) and `runDelete` calls correctly already.
- Phase 1's Task 7 added `Args: cobra.NoArgs, RunE: cmd.Help()` to hand-written parents. The generator's emitted parents (gen.go:233-236) have neither.

**What the hand-written files do that the generator CANNOT express today:**
1. `customers list` — `ListCustomersWithParamsWithPagination` with 9 filter flags + `--all` auto-pagination (customers.go:184-247).
2. `customers create` — `splitName(full-name)` into first/last; nested emails/phones/addresses/tags parsing (`parseLabelValue`, pipe-syntax).
3. `customers work-orders`/`vehicles work-orders` naming (kebab subcommand under snake parent) — generator's `useArgsSuffix` always emits ` <id>` for show-type; nested-list ops classify as `show` because they have `{id}` (gen.go:293-297).
4. `work_orders` tab commands (`estimate/wip/inspection/parts/payments`) — generator supports via `tab:` override (emitTabHandler exists and works).
5. `tags create/delete/rename` — a TYPE-branching create (customer vs vehicle tag) and non-REST mutation via `UpdateTags`. Fundamentally NOT derivable from a per-operation generator: two spec ops (`create_customer_tag`, `create_vehicle_tag`) must merge into one `tags create` with a `--type` flag.
6. `vehicles decode-vin <vin>` / `customers lookup <query>` — positional string arg (generator supports via `positional_arg:`).
7. `locations show <string-id>` — `showStr` (generator supports via IDType string detection).
8. Parent `Short` strings ("Manage customers") — generator emits `titleCase(resource) + " commands"`.
9. Truncation-check on `work_orders show` (checkTruncatedResponse) — hand-written only.

**Naming decision (D2, locked):** squashed compounds. `workorders` (aliases `work_orders`, `wo`), `servicecategories` (aliases `service-categories`, `sc`). Nested: `customers workorders`, `vehicles workorders` (alias `work-orders`). Flags stay kebab-case. Watch `servicecategories` readability on the real help screen (spec §2.5 caveat) — validate before release, revert cheaply via aliases if unreadable.

---

## Task 1: Generator — resource aliases + parent Args/Short

**Files:**
- Modify: `cmd/gencli/gen.go` (Overrides parsing, emitGroup parent emission)
- Modify: `cmd/gencli/main.go` (CommandOverride gains `aliases`, `short` fields; GroupOverride struct)
- Modify: `cmd/gen_overrides.yaml` (aliases + shorts per resource)
- Test: `cmd/gencli/gen_test.go`

The generator currently emits bare parents with no aliases, no `Args` validation, and synthesized Shorts. Extend the overrides schema so humans shape these per resource.

- [ ] **Step 1: Extend the overrides types (main.go)**

```go
// Overrides is the gen_overrides.yaml structure.
type Overrides struct {
	Commands      map[string]CommandOverride     `yaml:"commands"`
	FlagOverrides map[string]map[string]FlagOverride `yaml:"flag_overrides"`
	Exclude       []string                      `yaml:"exclude"`
	Groups        map[string]GroupOverride       `yaml:"groups"`
}

// GroupOverride shapes a resource parent command.
type GroupOverride struct {
	Aliases []string `yaml:"aliases"`
	Short   string   `yaml:"short"`
}
```

Extend `CommandOverride` (main.go:129-139):

```go
type CommandOverride struct {
	Resource         string   `yaml:"resource"`
	Command          string   `yaml:"command"`
	Summary          string   `yaml:"summary"`
	ActionSummary    string   `yaml:"action_summary"`
	Method           string   `yaml:"method"`
	RequestStruct    string   `yaml:"request_struct"`
	PositionalArg    string   `yaml:"positional_arg"`
	QueryParamStruct string   `yaml:"query_param_struct"`
	Paginated        *bool    `yaml:"paginated"`
	Tab              string   `yaml:"tab"`
	Aliases          []string `yaml:"aliases"`
	Short            string   `yaml:"short"`
}
```

`action_summary` is the past-tense message the runner renders on success (e.g. "Service category deactivated."); `summary` remains the command's Short shown in help. Add `ActionSummary string` to `GenCommand`, wire it in `buildCommand`, and Task 2's emitters use it with a `titleCase(singularize(resource)) + " " + cmd.Command + "."` fallback when unset.

- [ ] **Step 2: Failing tests — group/command alias + short plumbing**

Add to `cmd/gencli/gen_test.go`:

```go
func TestGroupOverridesPlumbThrough(t *testing.T) {
	overrides := &Overrides{
		Groups: map[string]GroupOverride{
			"workorders": {Aliases: []string{"work_orders", "wo"}, Short: "Manage work orders"},
		},
		Commands: map[string]CommandOverride{},
	}
	group := CommandGroup{Resource: "workorders", Commands: []GenCommand{
		{OperationID: "list_work_orders", Resource: "workorders", Command: "list", Method: "get", IsPaginated: true, SDKMethod: "ListWorkOrders"},
	}}
	code, err := emitGroup(group, nil, overrides)
	if err != nil {
		t.Fatalf("emitGroup: %v", err)
	}
	if !strings.Contains(code, `Aliases: []string{"work_orders", "wo"}`) {
		t.Errorf("parent aliases not emitted:\n%s", code)
	}
	if !strings.Contains(code, `Short: "Manage work orders"`) {
		t.Errorf("parent short not emitted:\n%s", code)
	}
	if !strings.Contains(code, "Args:  cobra.NoArgs") {
		t.Errorf("parent Args validation not emitted:\n%s", code)
	}
}
```

Note: exact formatting depends on jennifer output — if `Args:  cobra.NoArgs` fails on whitespace, assert `strings.Contains(code, "cobra.NoArgs")` instead. Run once, adjust the assertion to the actual emitted form, keep it strict.

- [ ] **Step 3: Verify it fails**

```bash
go test ./cmd/gencli/ -run TestGroupOverridesPlumbThrough -v
```

Expected: FAIL — no aliases, synthesized Short, no Args.

- [ ] **Step 4: Implement in emitGroup (gen.go:228-247)**

Replace the parent emission:

```go
	parentVar := group.Resource + "Cmd"
	parentDict := jen.Dict{
		jen.Id("Use"): jen.Lit(group.Resource),
	}
	short := group.Resource + " commands"
	if ov, ok := overrides.Groups[group.Resource]; ok {
		if ov.Short != "" {
			short = ov.Short
		}
		if len(ov.Aliases) > 0 {
			aliasLit := make([]jen.Code, 0, len(ov.Aliases))
			for _, a := range ov.Aliases {
				aliasLit = append(aliasLit, jen.Lit(a))
			}
			parentDict[jen.Id("Aliases")] = jen.Values(aliasLit...)
		}
	}
	parentDict[jen.Id("Short")] = jen.Lit(short)
	// Phase 1 Task 7 parity: typo'd subcommands must fail, bare parents show help.
	parentDict[jen.Id("Args")] = jen.Qual("github.com/spf13/cobra", "NoArgs")
	parentDict[jen.Id("RunE")] = jen.Func().Params(
		jen.Id("cmd").Op("*").Qual("github.com/spf13/cobra", "Command"),
		jen.Id("args").Index().Id("string"),
	).Id("error").Block(
		jen.Return(jen.Id("cmd").Dot("Help").Call()),
	)
	g.Id(parentVar).Op(":=").Op("&").Qual("github.com/spf13/cobra", "Command").Values(parentDict)
```

- [ ] **Step 5: Per-command aliases in emitCommand (gen.go:253-269)**

Thread `cmd.Aliases` (new GenCommand field, populated from `CommandOverride.Aliases` in buildCommand) into the command dict:

```go
	if len(cmd.Aliases) > 0 {
		aliasLit := make([]jen.Code, 0, len(cmd.Aliases))
		for _, a := range cmd.Aliases {
			aliasLit = append(aliasLit, jen.Lit(a))
		}
		dict[jen.Id("Aliases")] = jen.Values(aliasLit...)
	}
```

Add `Aliases []string` to `GenCommand` (gen.go:18-39) and wire it in `buildCommand`'s override application:

```go
		if len(ov.Aliases) > 0 {
			cmd.Aliases = ov.Aliases
		}
```

- [ ] **Step 6: Run tests, commit**

```bash
go test ./cmd/gencli/ -v
git add cmd/gencli/ cmd/gen_overrides.yaml
git commit -m "feat(gencli): resource aliases, parent shorts, NoArgs validation via group overrides"
```

### Task 2: Generator — action handlers for empty-body sub-path ops

**Files:**
- Modify: `cmd/gencli/gen.go` (emitActionHandler → real emission for PATCH/POST no-scalar-body ops)
- Test: `cmd/gencli/gen_test.go`

The broken case: PATCH/POST to `/resource/{id}/<action>` whose requestBody has no scalar fields (empty `properties: {}`). Today: either "not yet generated" stub or nil-body actionUpdate. The fix mirrors the hand-written `runServiceCategoryAction`: parse id, call SDK method with just `(ctx, id)`, render.

- [ ] **Step 1: Failing test**

```go
func TestEmitGroup_ServiceCategoryActionsCompile(t *testing.T) {
	// Deactivate is PATCH /service_categories/{id}/deactivate with an
	// empty-object body: the case that ships broken commands today.
	cmd := GenCommand{
		OperationID: "deactivate_service_category",
		Resource:    "servicecategories",
		Command:     "deactivate",
		Method:      "patch",
		Path:        "/service_categories/{id}/deactivate",
		HasIDParam:  true,
		Summary:     "Deactivate a service category by ID",
		SDKMethod:   "DeactivateServiceCategory",
		RequestBody: &RequestBody{Content: map[string]Media{
			"application/json": {Schema: Schema{Type: "object", Properties: map[string]Schema{}}},
		}},
	}
	group := CommandGroup{Resource: "servicecategories", Commands: []GenCommand{cmd}}
	code, err := emitGroup(group, nil, &Overrides{})
	if err != nil {
		t.Fatalf("emitGroup: %v", err)
	}
	for _, want := range []string{
		"runServicecategoriesDeactivate",            // handler exists
		"client.DeactivateServiceCategory(ctx, id)",  // direct (ctx, id) call — NOT runAction with nil body
		"cobra.ExactArgs(1)",                          // id positional enforced
	} {
		if !strings.Contains(code, want) {
			t.Errorf("emitted code missing %q:\n%s", want, code)
		}
	}
	// The stub must be gone.
	if strings.Contains(code, "not yet generated") {
		t.Error("action stub still emitted")
	}
}
```

- [ ] **Step 2: Verify it fails**

```bash
go test ./cmd/gencli/ -run TestEmitGroup_ServiceCategoryActionsCompile -v
```

Expected: FAIL — current emitActionUpdateHandler emits `runAction` with a bodyBuilder returning `nil, nil` and a sender asserting `body.(wenmar."")` (RequestStruct empty → sender returns nil,nil without calling the SDK — the command compiles but silently does nothing).

- [ ] **Step 3: Implement**

In `classifyCommand`, add a new class for empty-body sub-path actions before the existing actionUpdate/actionCreate cases:

```go
	case "patch":
		if cmd.HasIDParam && cmd.RequestBody != nil && isSubAction(cmd) {
			if len(cmd.BodyFields) == 0 {
				return "actionNoBody" // e.g. service category deactivate/reactivate/move_up/move_down
			}
			return "actionUpdate"
		}
		if cmd.RequestBody != nil {
			return "update"
		}
		return "action"
```

Same insertion for `post` (with `actionCreate` kept for body-carrying sub-actions like merge). Then replace `emitActionHandler` (gen.go:1086-1090) with a real emitter, and wire it for `actionNoBody` in `emitHandler`'s switch:

```go
// emitActionNoBodyHandler emits the handler for id-scoped actions whose
// body carries no scalar fields (e.g. service category deactivate). Mirrors
// the hand-written runServiceCategoryAction: parse id, call (ctx, id), render.
func emitActionNoBodyHandler(g *jen.Group, cmd GenCommand) {
	resource := cmd.Resource
	g.Return(jen.Id("runActionNoBody").Call(
		jen.Id("cmd"), jen.Id("args"),
		jen.Lit(resource),
		jen.Lit(strings.ToUpper(cmd.Method)),
		pathFnForRunner(cmd),
		jen.Lit(actionSummary(cmd)), // from ActionSummary override, with fallback
		jen.Func().Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
			jen.Id("id").Id("int"),
		).Params(jen.Any(), jen.Error()).BlockFunc(func(bg *jen.Group) {
			callArgs := sdkCallArgs(cmd, true)
			bg.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").Dot(sdkMethodNameFor(cmd)).Call(callArgs...)
			bg.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Id("err")),
			)
			bg.Return(jen.Id("resp").Dot("JSON200"), jen.Nil())
		),
	))
}
```

And delete the old stub entirely. `needsExactArgs` (gen.go:271-278) gains `actionNoBody`. `useArgsSuffix` gains `actionNoBody → " <id>"`.

- [ ] **Step 4: Add the shared runner `runActionNoBody` to cmd/runners.go**

```go
// runActionNoBody is the shared skeleton for id-scoped action commands whose
// request body has no scalar fields (service category deactivate/reactivate/
// move-up/move-down). It parses the id, calls the SDK, renders the response.
func runActionNoBody(cmd *cobra.Command, args []string, resource, method string, pathFn func(args []string) string, summary string,
	action func(ctx context.Context, client *wenmar.Client, id int) (any, error)) error {
	id, err := parseInt(args[0])
	if err != nil {
		return err
	}

	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest(method, pathFn(args))

	data, err := extract(action(context.Background(), client, id))
	if err != nil {
		return err
	}

	mode := resolveMode()
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: showBreadcrumbs(resource, args[0])}
	return output.Render(cmd.OutOrStdout(), data, summary, nil, opts)
}

// extract normalizes a typed response's JSON200 field for rendering.
// (Renamed from extractData in Phase 1 — use whatever the current helper is
// named; check cmd/helpers.go. If extractData still exists, call it.)
```

CORRECTION — do not rename anything. Use the existing `extractData` helper exactly as the hand-written code does:

```go
	data, err := extractData(action(context.Background(), client, id))
```

Delete the `extract` variant above; the runner body is exactly `runServiceCategoryAction`'s (service_categories.go:226-247) generalized. `parseInt` exists from Phase 1's runners (runners.go:19-25).

- [ ] **Step 5: Add seed-defaults support (POST, no {id}, empty body)**

`seed_defaults_service_categories` classifies today as `create` (POST + body) with empty BodyFields → `runCreate` with nil body → sender returns nil,nil silently. Add to classifyCommand's `post` branch:

```go
	case "post":
		if cmd.HasIDParam && cmd.RequestBody != nil && isSubAction(cmd) && len(cmd.BodyFields) > 0 {
			return "actionCreate"
		}
		if cmd.HasIDParam && cmd.RequestBody != nil && isSubAction(cmd) {
			return "actionNoBody"
		}
		if cmd.RequestBody != nil && len(cmd.BodyFields) == 0 && !cmd.HasIDParam {
			return "seedAction" // e.g. seed_defaults_service_categories
		}
		if cmd.RequestBody != nil {
			return "create"
		}
		return "action"
```

And emit via the same `runActionNoBody` runner (works fine for a no-id call — pathFn ignores args; the summary string differs). For seed-defaults the summary should use the response's `Created` count like the hand-written version (service_categories.go:214-217). Simplest honest emission: summary constant `"Default service categories seeded."` — the count nuance is a display nicety; keep parity by emitting a `summaryFrom` variant instead:

```go
// emitSeedActionHandler emits POST-collection actions with empty bodies.
func emitSeedActionHandler(g *jen.Group, cmd GenCommand) {
	g.Return(jen.Id("runSeedAction").Call(
		jen.Id("cmd"),
		jen.Lit(cmd.Resource),
		jen.Lit(cmd.Path),
		jen.Lit(titleCase(singularize(cmd.Resource)) + " defaults seeded."),
		jen.Func().Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
		).Params(jen.Any(), jen.Error()).BlockFunc(func(bg *jen.Group) {
			bg.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").Dot(sdkMethodNameFor(cmd)).Call(jen.Id("ctx"))
			bg.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Id("err")),
			)
			bg.Return(jen.Id("resp").Dot("JSON200"), jen.Nil())
		),
	))
}
```

With runner (add to cmd/runners.go):

```go
// runSeedAction is the skeleton for POST-collection actions with empty
// bodies (e.g. service categories seed-defaults): no id, call, render.
func runSeedAction(cmd *cobra.Command, resource, path string, summary string,
	action func(ctx context.Context, client *wenmar.Client) (any, error)) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("POST", path)

	data, err := extractData(action(context.Background(), client))
	if err != nil {
		return err
	}

	mode := resolveMode()
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs(resource)}
	return output.Render(cmd.OutOrStdout(), data, summary, nil, opts)
}
```

- [ ] **Step 6: Golden-style compile verification**

```bash
go run ./cmd/gencli -spec ../wenmar-sdk/spec/openapi.enriched.yaml -overrides cmd/gen_overrides.yaml -out /tmp/opencode/gen-p2-smoke/
grep -c "runActionNoBody\|runSeedAction" /tmp/opencode/gen-p2-smoke/gen_servicecategories.go
```

Expected: servicecategories file has deactivate/reactivate/move-up/move-down via runActionNoBody and seed-defaults via runSeedAction; zero "not yet generated" anywhere:

```bash
grep -rn "not yet generated" /tmp/opencode/gen-p2-smoke/ ; echo "exit=$?"   # want exit 1 (no matches)
```

- [ ] **Step 7: Run tests, commit**

```bash
go test ./cmd/gencli/ -v
git add cmd/gencli/ cmd/runners.go
git commit -m "feat(gencli): real action handlers for empty-body sub-path ops; runActionNoBody/runSeedAction runners"
```

### Task 3: Generator — list with query filters + --all auto-pagination

**Files:**
- Modify: `cmd/gencli/gen.go` (queryParam list handler variant)
- Modify: `cmd/runners.go` (no change needed — runListPaginatedWithAll exists)
- Modify: `cmd/gen_overrides.yaml` (customers list wiring)
- Test: `cmd/gencli/gen_test.go`

`customers list` needs: `ListCustomersWithParamsWithPagination(ctx, ListCustomersParams)` with 9 filter flags, plus `--all`. The `query_param:` override exists but emits `runList` (no paginator). New classification: a queryParam GET that is ALSO `x-paginated`.

- [ ] **Step 1: Failing test**

```go
func TestEmitGroup_CustomersListWithFiltersPaginated(t *testing.T) {
	cmd := GenCommand{
		OperationID: "list_customers",
		Resource:    "customers",
		Command:     "list",
		Method:      "get",
		Path:        "/customers",
		IsPaginated: true,
		QueryParamStruct: "ListCustomersParams",
		SDKMethod:   "ListCustomers",
		QueryFields: []BodyField{
			{JSONName: "query", GoName: "Query", FlagName: "query", Type: "string", HelpText: "Full-text search"},
			{JSONName: "page", GoName: "Page", FlagName: "page", Type: "integer", HelpText: "Page number"},
		},
	}
	group := CommandGroup{Resource: "customers", Commands: []GenCommand{cmd}}
	code, err := emitGroup(group, nil, &Overrides{})
	if err != nil {
		t.Fatalf("emitGroup: %v", err)
	}
	for _, want := range []string{
		"runListPaginatedWithAll",                                  // paginated skeleton
		"ListCustomersWithParamsWithPagination",                    // SDK filter variant
		`FlagName: "query"`,                                        // hmm — jen emits Go, not YAML
	} {
		_ = want
	}
}
```

CORRECTION — the assertions above must check emitted Go source. Replace the whole test:

```go
func TestEmitGroup_CustomersListWithFiltersPaginated(t *testing.T) {
	cmd := GenCommand{
		OperationID: "list_customers",
		Resource:    "customers",
		Command:     "list",
		Method:      "get",
		Path:        "/customers",
		IsPaginated: true,
		QueryParamStruct: "ListCustomersParams",
		SDKMethod:   "ListCustomers",
		QueryFields: []BodyField{
			{JSONName: "query", GoName: "Query", FlagName: "query", Type: "string", HelpText: "Full-text search"},
			{JSONName: "page", GoName: "Page", FlagName: "page", Type: "integer", HelpText: "Page number"},
		},
	}
	group := CommandGroup{Resource: "customers", Commands: []GenCommand{cmd}}
	code, err := emitGroup(group, nil, &Overrides{})
	if err != nil {
		t.Fatalf("emitGroup: %v", err)
	}
	for _, want := range []string{
		"runListPaginatedWithAll",                       // paginated skeleton with --all support
		"ListCustomersWithParamsWithPagination",        // SDK filter variant
		"customersQuery",                               // query flag var (camelCase resource prefix)
		"customersPage",                                // page flag var
		"\"query\"",                                     // flag name literal
		"\"page\"",                                      // flag name literal
	} {
		if !strings.Contains(code, want) {
			t.Errorf("emitted code missing %q:\n%s", want, code)
		}
	}
}
```

- [ ] **Step 2: Verify it fails**

```bash
go test ./cmd/gencli/ -run TestEmitGroup_CustomersListWithFiltersPaginated -v
```

Expected: FAIL — queryParam lists emit `runList` + plain SDK call today.

- [ ] **Step 3: Implement**

In `classifyCommand`, `get` branch (gen.go:291-303):

```go
	case "get":
		if cmd.HasIDParam {
			if cmd.IDType == "string" {
				return "showStr"
			}
			return "show"
		}
		if cmd.IsPaginated {
			if cmd.QueryParamStruct != "" {
				return "listPaginatedWithParams"
			}
			return "listPaginated"
		}
		if cmd.QueryParamStruct != "" {
			return "queryParam"
		}
		return "list"
```

New emitter (place near emitListPaginatedHandler, gen.go:798-815):

```go
// emitListPaginatedWithParamsHandler emits a paginated list whose SDK call
// takes a query-params struct (customers list with filters). Falls back to
// the plain paginated call when no filter flags were provided.
func emitListPaginatedWithParamsHandler(g *jen.Group, cmd GenCommand) {
	resource := cmd.Resource
	g.Return(jen.Id("runListPaginatedWithAll").Call(
		jen.Id("cmd"),
		jen.Lit(resource),
		requestPathExpr(cmd),
		jen.Id(hasFiltersVarName(cmd)),
		jen.Func().Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
		).Params(jen.Any(), jen.Op("*").Qual(wenmarPkg, "Paginator"), jen.Error()).BlockFunc(func(bg *jen.Group) {
			// if <resource>ListHasFilters() { params := ...; resp, paginator, err := client.XWithParamsWithPagination(ctx, params) }
			bg.If(jen.Id(hasFiltersFnName(cmd)).Call()).BlockFunc(func(ibg *jen.Group) {
				ibg.List(jen.Id("resp"), jen.Id("paginator"), jen.Id("err")).Op(":=").Id("client").
					Dot(sdkMethodNameFor(cmd) + "WithParamsWithPagination").Call(
					jen.Id("ctx"),
					jen.Qual(wenmarPkg, cmd.QueryParamStruct).Values(queryParamDict(cmd)),
				)
				ibg.If(jen.Id("err").Op("!=").Nil()).Block(
					jen.Return(jen.Nil(), jen.Nil(), jen.Id("err")),
				)
				ibg.Return(jen.Id("resp").Dot("JSON200"), jen.Id("paginator"), jen.Nil())
			})
			// Plain variant: resp, paginator, err := client.XWithPagination(ctx)
			bg.List(jen.Id("resp"), jen.Id("paginator"), jen.Id("err")).Op(":=").Id("client").
				Dot(sdkMethodNameFor(cmd) + "WithPagination").Call(jen.Id("ctx"))
			bg.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Nil(), jen.Id("err")),
			)
			bg.Return(jen.Id("resp").Dot("JSON200"), jen.Id("paginator"), jen.Nil())
		}),
	))
}

func hasFiltersFnName(cmd GenCommand) string {
	return toCamelCase(cmd.Resource) + "ListHasFilters"
}

func hasFiltersVarName(cmd GenCommand) string {
	return toCamelCase(cmd.Resource) + "ListAll"
}
```

Wait — the fallback emits BOTH branches unconditionally in sequence; the second would be unreachable after the if returns. Structure it as if/else:

```go
			bg.If(jen.Id(hasFiltersFnName(cmd)).Call()).BlockFunc(func(ibg *jen.Group) {
				// ... filtered call, returns
			}).Else().BlockFunc(func(ebg *jen.Group) {
				ebg.List(jen.Id("resp"), jen.Id("paginator"), jen.Id("err")).Op(":=").Id("client").
					Dot(sdkMethodNameFor(cmd) + "WithPagination").Call(jen.Id("ctx"))
				ebg.If(jen.Id("err").Op("!=").Nil()).Block(
					jen.Return(jen.Nil(), jen.Nil(), jen.Id("err")),
				)
				ebg.Return(jen.Id("resp").Dot("JSON200"), jen.Id("paginator"), jen.Nil())
			})
```

Also emit the has-filters helper + `--all` flag registration. In `emitFlagRegistration` (for `listPaginatedWithParams` commands), append:

```go
	if classifyCommand(cmd) == "listPaginatedWithParams" {
		allVar := toCamelCase(cmd.Resource) + "ListAll"
		g.Id(cmdVar).Dot("Flags").Call().Dot("BoolVar").Call(
			jen.Op("&").Id(allVar),
			jen.Lit("all"),
			jen.False(),
			jen.Lit("Fetch all pages by following pagination links"),
		)
	}
```

And emit the has-filters function in the file (emitCommand or emitGroup level — emitGroup, once per group, after handlers):

```go
// hasFilters function: returns true when any filter flag was set.
f.Func().Id(hasFiltersFnName(firstParamsCmd)).Params().Id("bool").BlockFunc(func(g *jen.Group) {
	conditions := []jen.Code{}
	for _, bf := range firstParamsCmd.QueryFields {
		varName := bodyFieldVarName(cmd.Resource, bf.GoName)
		switch bf.Type {
		case "string":
			conditions = append(conditions, jen.Id(varName).Op("!=").Lit(""))
		case "integer":
			conditions = append(conditions, jen.Id(varName).Op(">").Lit(0))
		case "boolean":
			conditions = append(conditions, jen.Id(varName))
		}
	}
	if len(conditions) == 0 {
		g.Return(jen.False())
		return
	}
	g.Return(jen.Op("").Add(conditions[0])) // jen join with || below
})
```

jennifer boolean-join detail — build it as:

```go
	cond := conditions[0]
	for _, c := range conditions[1:] {
		cond = jen.Op("||").Add ... // jen does not chain like this; use:
	}
```

jennifer's `jen.Op("||")` must wrap two codes: `conditions[0].Op("||").Add(conditions[1])` is wrong; the correct pattern is to emit `return a || b` via `jen.Return(jen.Op("").Add(conditions[0]))` does not work either. At implementation time, build the disjunction with jen.List or emit an if-chain:

```go
	// Simplest robust emission: if a { return true }; if b { return true }; return false
	for _, c := range conditions {
		g.If(c).Block(jen.Return(jen.True()))
	}
	g.Return(jen.False())
```

Use the if-chain form — it's unambiguous in jennifer and reads fine in generated code.

Flag var naming collision note: `bodyFieldVarName(resource, "Query")` → `customersQuery`. The hand-written file used `customersListQuery` (with "List" infix). The generated names differ but are internal — no compatibility concern; surface (flag NAMES) stays identical. Update the golden test to match the generated names.

- [ ] **Step 4: Wire customers list in gen_overrides.yaml**

The customers list entry today has `method: ListCustomers` and no query_param_struct. Update it:

```yaml
  list_customers:
    resource: customers
    command: list
    summary: List all customers, paginated via the Link header
    method: ListCustomers
    query_param_struct: ListCustomersParams
```

`extractQueryFields` derives the 9 filter flags from the spec's query parameters (query, type, has_vehicle, has_balance, last_visit_months, tag_ids, per_page, page) — verify against the spec's list_customers parameters block (spec lines ~330-360, the `ListCustomersParams` fields match the generated SDK struct exactly: HasBalance, HasVehicle, LastVisitMonths, Page, PerPage, Query, TagIds, Type). The `tag_ids` array param will emit an `IntVar` flag bound via `flagBindMethod("array")` → default `StringVar` — wrong. Handle arrays in extractQueryFields: skip `array`-typed query params (they need custom parsing, same policy as body arrays). Then add tag_ids via a `flag_overrides`-style hand-tuned path OR keep `--tag-ids` as a hand-maintained addition in the post-cutover world via a `custom_flags:` override — see Task 5's `flags.go` companion-file mechanism.

DECISION for this task: `extractQueryFields` skips arrays (one-line change mirroring extractScalarFields: `if p.Schema.Type == "array" || p.Schema.Type == "object" { continue }`). The `--tag-ids` filter is dropped from generated output; it is a rarely-used filter and the surface-snapshot diff will show the removal. Document in the cutover commit that `--tag-ids` returns via Task 5's companion mechanism if wanted. (Alternative — implement array query params now — rejected as scope creep: it needs a parser convention (comma-split IntSlice), which belongs to a follow-up.)

- [ ] **Step 5: Verify against smoke generation**

```bash
go run ./cmd/gencli -spec ../wenmar-sdk/spec/openapi.enriched.yaml -overrides cmd/gen_overrides.yaml -out /tmp/opencode/gen-p2b/
grep -n "ListCustomersWithParamsWithPagination" /tmp/opencode/gen-p2b/gen_customers.go | head -2
grep -n "customersListHasFilters\|customersListAll" /tmp/opencode/gen-p2b/gen_customers.go | head -4
```

- [ ] **Step 6: Run tests, commit**

```bash
go test ./cmd/gencli/ -v
git add cmd/gencli/ cmd/gen_overrides.yaml
git commit -m "feat(gencli): paginated list with query filters + --all (customers list parity)"
```

### Task 4: Companion files — non-derivable commands stay hand-written, generated files replace twins

**Files:**
- Modify: `cmd/gen_overrides.yaml` (exclude list grows)
- Create: `cmd/tags.go` (rewrite — see below), `cmd/customers_extras.go`, `cmd/vehicles_extras.go` (hand-written nested-collection/naming companions)
- Delete: `cmd/customers.go`, `cmd/vehicles.go`, `cmd/work_orders.go`, `cmd/drivers.go`, `cmd/vendors.go`, `cmd/statements.go`, `cmd/service_categories.go`, `cmd/account.go`, `cmd/locations.go`, `cmd/tags.go` (the twins)
- Modify: `.gitignore` (drop `cmd/gen_*.go`), `Makefile` (generate target output stays; test-generated dies)

Some commands can NEVER be derived from a per-operation generator (verified in ground truth: tags type-branching, customers splitName, nested-collection naming, work_orders tab truncation check). These become small hand-written "companion" files that live ALONGSIDE the generated files in the default build — no build tags — while the generator's exclusions prevent duplicate emission.

- [ ] **Step 1: Update the exclusion strategy in gen_overrides.yaml**

The `exclude:` list currently removes ops "with no SDK wrapper yet or needing hand-written logic." After cutover it removes ops handled by companion files OR still missing SDK methods. New exclude list (replace existing):

```yaml
exclude:
  # Hand-written companion commands (see cmd/*_extras.go / cmd/tags.go):
  - create_customer            # splitName + nested attribute parsing (customers_extras.go)
  - update_customer            # nested emails/phones parsing (customers_extras.go)
  - list_customers_vehicle_history   # no SDK wrapper (unchanged)
  - delete_customer           # no SDK wrapper (unchanged)
  - list_team                 # no SDK wrapper (unchanged)
  - get_work_orders_summary   # no SDK wrapper (unchanged)
  - create_work_order_payment # no SDK wrapper (unchanged)
  - create_customer_tag       # tags create type-branches (tags.go)
  - create_vehicle_tag        # tags create type-branches (tags.go)
  - update_tags               # tags delete/rename via UpdateTags (tags.go)
  - list_customer_tags        # tags list (tags.go)
  - list_vehicle_tags         # tags list (tags.go)
  - delete_customer_tag       # tags delete (tags.go)
  - delete_vehicle_tag        # tags delete (tags.go)
  - update_customer_tag       # tags rename (tags.go)
  - update_vehicle_tag        # tags rename (tags.go)
  - list_customers_drivers    # drivers resource nests under customers (drivers companion)
  - show_driver
  - create_driver
  - update_driver
  - delete_driver
  - list_customers_statements # statements companion
  - show_statement
  - list_customers_vehicles    # nested-collection naming (customers_extras.go)
  - list_customers_work_orders # nested-collection naming (customers_extras.go)
  - list_vehicles_work_orders  # nested-collection naming (vehicles_extras.go)
  - show_work_order           # truncation check on show (work_orders companion)
  - show_work_order_estimate   # tabs via companion fetchWorkOrderTab
  - show_work_order_wip
  - show_work_order_inspection
  - show_work_order_parts
  - show_work_order_payments
  - create_work_order
  - update_work_order
  - delete_work_order
  - create_service_category    # typed request with defaults (service_categories companion)
  - update_service_category
  - delete_service_category
  - deactivate_service_category
  - reactivate_service_category
  - move_up_service_category
  - move_down_service_category
  - seed_defaults_service_categories
  - list_service_categories
```

STOP — this exclusion list is wrong-headed. Excluding everything makes the generator pointless. The division of labor must be: the generator emits everything it CAN emit correctly (which after Tasks 1-3 is nearly everything); companion files exist ONLY for what truly cannot be derived. Re-derive the minimal exclusion list:

**Generator-emitted (remove from exclude):** `list_service_categories`, `create_service_category`, `update_service_category`, `delete_service_category`, `deactivate/reactivate/move_up/move_down_service_category`, `seed_defaults_service_categories` (all now handled by Tasks 1-2 emission paths).

**Companion-only (keep in exclude):**
- `create_customer`, `update_customer` (splitName + nested attributes)
- `create_customer_tag`, `create_vehicle_tag`, `update_tags`, `list_tags`, `delete_customer_tag`, `delete_vehicle_tag`, `update_customer_tag`, `update_vehicle_tag`, `list_customer_tags`, `list_vehicle_tags` (tags.go — type-branching)
- `show_work_order` (truncation check), `show_work_order_estimate/wip/inspection/parts/payments` (tab fetch switch), `create_work_order`, `update_work_order`, `delete_work_order` — wait, these three are plain create/update/delete with request structs; the generator handles them. Verify: `create_work_order` body has customer_id/vehicle_id ints (required) → create path works. `update_work_order` body has intake_method string → update path works. `delete_work_order` → delete path works. REMOVE from exclude; generator emits them.
- `show_work_order` + tabs: generator's tab handler exists and works (emitTabHandler); but `show_work_order`'s truncation check is hand-written logic inside the handler. Options: (a) add a `truncated: true` override + generator support — new emission path for a niche behavior; (b) keep `show_work_order` + tabs in a small `work_orders_extras.go` companion. Choose (b): 5 tab commands + 1 show = 6 commands in one companion file, no generator growth. Keep `show_work_order*` in exclude.
- `list_customers_drivers`, `show_driver`, `create_driver`, `update_driver`, `delete_driver` — drivers nest under `/customers/{customer_id}/drivers` with a required `customer-id` flag. The generator's extra-path-param machinery handles this (pathFnForRunner + sdkCallArgs both support ExtraPathParams). Verify by checking what the current generated output for drivers looked like — the exclude list never included drivers, so the generator ALREADY emits them today; Phase 1's smoke generation would have produced gen_drivers.go. Do NOT exclude drivers; verify emission in Step 2.
- `list_customers_statements`, `show_statement` — same nested pattern as drivers (customer-id flag on list). Generator handles. Verify in Step 2.
- `list_customers_vehicles`, `list_customers_work_orders`, `list_vehicles_work_orders` — nested-collection LIST commands named `customers vehicles <id>` etc. These classify as `show` (GET + {id}) with the SDK method taking (ctx, customerID) — the generator's show handler passes `(ctx, id)` where id comes from args[0], but the SDK call needs the CUSTOMER id from args[0] — which IS the positional. Verify: `runCustomersVehicles` hand-written calls `client.ListCustomerVehicles(ctx, id)` with id = parsed args[0]. The generated show handler does exactly this via the getter closure. So the generator CAN emit these as show-type commands with kebab command names via overrides. REMOVE from exclude; wire via commands: overrides with `command: vehicles`, `command: work-orders`.

**Final minimal exclude list** (replace the current one):

```yaml
exclude:
  # No SDK wrapper yet (unchanged from before):
  - get_customers_vehicle_history
  - list_team
  - get_work_orders_summary
  - create_work_order_payment

  # Hand-written companions (cmd/*_extras.go) — not derivable per-operation:
  - create_customer            # splitName + nested emails/phones/addresses/tags
  - update_customer            # nested emails/phones parsing
  - create_customer_tag        # tags create --type branches (tags.go)
  - create_vehicle_tag
  - update_tags                # tags delete/rename (tags.go)
  - list_tags                  # tags list (tags.go)
  - delete_customer_tag        # tags delete (tags.go)
  - delete_vehicle_tag
  - update_customer_tag        # tags rename (tags.go)
  - update_vehicle_tag
  - list_customer_tags         # tags list (tags.go)
  - list_vehicle_tags
  - show_work_order            # truncation check (work_orders_extras.go)
  - show_work_order_estimate   # tab fetch switch (work_orders_extras.go)
  - show_work_order_wip
  - show_work_order_inspection
  - show_work_order_parts
  - show_work_order_payments
```

And the `commands:` map gains the nested-collection + naming entries:

```yaml
  list_customers_vehicles:
    resource: customers
    command: vehicles
    summary: List a customer's vehicles
    method: ListCustomerVehicles
  list_customers_work_orders:
    resource: customers
    command: workorders
    summary: List a customer's work orders
    method: ListCustomerWorkOrders
    aliases: [work-orders]
  list_vehicles_work_orders:
    resource: vehicles
    command: workorders
    summary: List a vehicle's work orders
    method: ListVehicleWorkOrders
    aliases: [work-orders]
  show_location:
    resource: locations
    command: show
    summary: Show a location by ID
    method: ShowLocation
  list_account:
    resource: account
    command: show
    summary: Show account details
    method: ListAccount
  lookup_customer:
    resource: customers
    command: lookup
    summary: Search customers by name/email/phone
    method: LookupCustomer
    positional_arg: string
  lookup_vehicle:
    resource: vehicles
    command: lookup
    summary: Search vehicles by make/model/plate/vin
    method: LookupVehicle
    positional_arg: string
  decode_vin:
    resource: vehicles
    command: decode-vin
    summary: Decode a VIN into make/model
    method: DecodeVin
    positional_arg: string
```

Note: `list_customers_vehicles` classifies as show (GET + {id}) — the emitted handler is correct (positional id → customer id → SDK call). The `show` classification applies even though it returns an array; output rendering is mode-driven, not shape-driven. Confirm in smoke output that `customers vehicles <id>` emits `runShow` with `ListCustomerVehicles(ctx, id)`.

- [ ] **Step 2: Full smoke generation and compile check**

```bash
go run ./cmd/gencli -spec ../wenmar-sdk/spec/openapi.enriched.yaml -overrides cmd/gen_overrides.yaml -out /tmp/opencode/gen-p2c/
ls /tmp/opencode/gen-p2c/
# Compile the generated files together with the non-resource cmd files:
mkdir -p /tmp/opencode/gen-p2c-build && cp /tmp/opencode/gen-p2c/*.go /tmp/opencode/gen-p2c-build/
go vet ./cmd/ 2>&1 | head -5   # still vetting old build — fine
# The real compile test happens in Task 5's cutover; here assert file inventory:
grep -L "not yet generated" /tmp/opencode/gen-p2c/*.go   # all files clean
```

Expected file inventory (76 ops − 18 excluded − duplicate-path skips): `gen_account.go`, `gen_customers.go`, `gen_drivers.go`, `gen_locations.go`, `gen_servicecategories.go`, `gen_statements.go`, `gen_tags.go` (only if any tag op remains unexcluded — after the exclusion above, NO tag ops remain, so NO gen_tags.go), `gen_vehicles.go`, `gen_vendors.go`, `gen_workorders.go`.

- [ ] **Step 3: Write the companion files**

`cmd/tags.go` — REWRITE the existing file (delete build tag, keep body):

```go
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// This file is a hand-written companion to the generated resource commands.
// tags create/delete/rename cannot be derived per-operation: one CLI command
// branches across two API resources (customer_tags vs vehicle_tags) selected
// by --type, and delete/rename both flow through the single UpdateTags op.

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "Manage customer and vehicle tags",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() },
}

var tagsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all customer and vehicle tags",
	RunE:    runTagsList,
}

var tagsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a customer or vehicle tag",
	RunE:  runTagsCreate,
}

var tagsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a customer or vehicle tag",
	RunE:  runTagsDelete,
}

var tagsRenameCmd = &cobra.Command{
	Use:   "rename",
	Short: "Rename a customer or vehicle tag",
	RunE:  runTagsRename,
}

var (
	tagsType string
	tagsName string
	tagsID   int
)

func init() {
	tagsCreateCmd.Flags().StringVar(&tagsType, "type", "customer", "Tag type (customer or vehicle)")
	tagsCreateCmd.Flags().StringVar(&tagsName, "name", "", "Tag name (required)")
	tagsCreateCmd.MarkFlagRequired("name")
	tagsDeleteCmd.Flags().StringVar(&tagsType, "type", "customer", "Tag type (customer or vehicle)")
	tagsDeleteCmd.Flags().IntVar(&tagsID, "id", 0, "Tag ID (required)")
	tagsDeleteCmd.MarkFlagRequired("id")
	tagsRenameCmd.Flags().StringVar(&tagsType, "type", "customer", "Tag type (customer or vehicle)")
	tagsRenameCmd.Flags().IntVar(&tagsID, "id", 0, "Tag ID (required)")
	tagsRenameCmd.Flags().StringVar(&tagsName, "name", "", "New tag name (required)")
	tagsRenameCmd.MarkFlagRequired("id")
	tagsRenameCmd.MarkFlagRequired("name")

	tagsCmd.AddCommand(tagsListCmd, tagsCreateCmd, tagsDeleteCmd, tagsRenameCmd)
	rootCmd.AddCommand(tagsCmd)
}

func runTagsList(cmd *cobra.Command, args []string) error {
	return runList(cmd, "tags", "/settings/tags", func(ctx context.Context, client *wenmar.Client) (any, error) {
		resp, err := client.ListTags(ctx)
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

func runTagsCreate(cmd *cobra.Command, args []string) error {
	if tagsType == "vehicle" {
		return runCreate(cmd, "tags", "/vehicle_tags", "Vehicle tag created.", func() (any, error) {
			return wenmar.CreateVehicleTagRequest{Name: tagsName}, nil
		}, func(ctx context.Context, client *wenmar.Client, body any) (any, error) {
			resp, err := client.CreateVehicleTag(ctx, body.(wenmar.CreateVehicleTagRequest))
			if err != nil {
				return nil, err
			}
			return resp.JSON201, nil
		})
	}

	return runCreate(cmd, "tags", "/customer_tags", "Customer tag created.", func() (any, error) {
		return wenmar.CreateCustomerTagRequest{Name: tagsName}, nil
	}, func(ctx context.Context, client *wenmar.Client, body any) (any, error) {
		resp, err := client.CreateCustomerTag(ctx, body.(wenmar.CreateCustomerTagRequest))
		if err != nil {
			return nil, err
		}
		return resp.JSON201, nil
	})
}

func runTagsDelete(cmd *cobra.Command, args []string) error {
	return runTagsMutation(cmd, "Tag deleted.", func() wenmar.UpdateTagsRequest {
		req := wenmar.UpdateTagsRequest{}
		destroy := "1"
		if tagsType == "vehicle" {
			vt := []wenmar.VehicleTagUpdate{{UnderscoreDestroy: destroy, Id: tagsID}}
			req.VehicleTags = &vt
		} else {
			req.CustomerTags = []wenmar.CustomerTagUpdate{{UnderscoreDestroy: &destroy, Id: tagsID}}
		}
		return req
	})
}

func runTagsRename(cmd *cobra.Command, args []string) error {
	return runTagsMutation(cmd, fmt.Sprintf("Tag %d renamed to %s.", tagsID, tagsName), func() wenmar.UpdateTagsRequest {
		req := wenmar.UpdateTagsRequest{}
		name := tagsName
		if tagsType == "vehicle" {
			vt := []wenmar.VehicleTagUpdate{{Id: tagsID}}
			req.VehicleTags = &vt
		} else {
			req.CustomerTags = []wenmar.CustomerTagUpdate{{Id: tagsID, Name: &name}}
		}
		return req
	})
}

// runTagsMutation is the shared skeleton for tags delete/rename, which both
// PATCH /settings/tags and render the response.
func runTagsMutation(cmd *cobra.Command, summary string, bodyBuilder func() wenmar.UpdateTagsRequest) error {
	client, err := newScopedClient(context.Background())
	if err != nil {
		return err
	}
	setRequest("PATCH", "/settings/tags")

	body := bodyBuilder()
	resp, err := client.UpdateTags(context.Background(), body)
	if err != nil {
		return err
	}

	data := extractData(resp.JSON200)
	mode := resolveMode()
	opts := output.Options{Mode: mode, JQFilter: jqFlag, Breadcrumbs: listBreadcrumbs("tags")}
	return output.Render(cmd.OutOrStdout(), data, summary, nil, opts)
}
```

`cmd/customers_extras.go`:

```go
package cmd

import (
	"context"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// Hand-written companion to the generated customers commands. create/update
// need flag logic the per-operation generator cannot derive: full-name
// splitting and pipe-syntax parsing for nested emails/phones/addresses/tags.

var (
	customerCreateFullName string
	customerUpdateFullName string
	customerCompanyName    string
	// ... (all customerCreate*/customerUpdate* flag vars from the deleted
	// customers.go — copy the var block verbatim from customers.go:104-124,
	// including customerEmails, customerPhones, customerAddresses,
	// customerTagIDs, customerRemovePhoneIDs. NOTE: Phase 1 Task 5 already
	// removed customerRemoveEmailIDs/customerRemoveAddressIDs.)
)

var customersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new customer",
	RunE:  runCustomersCreate,
}

var customersUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a customer by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runCustomersUpdate,
}

func init() {
	// Copy the create/update flag registrations verbatim from the deleted
	// customers.go init() (lines 136-168 post-Phase-1), excluding the
	// customersList* registrations (generated file owns list now).
	customersCmd.AddCommand(customersCreateCmd, customersUpdateCmd)
}

func runCustomersCreate(cmd *cobra.Command, args []string) error {
	return runCreate(cmd, "customers", "/customers", "Customer created.", func() (any, error) {
		firstName, lastName := splitName(customerCreateFullName)
		req := wenmar.CreateCustomerRequest{
			FirstName: firstName,
			LastName:  lastName,
		}
		applyCustomerFlags(&req)
		return req, nil
	}, func(ctx context.Context, client *wenmar.Client, body any) (any, error) {
		resp, err := client.CreateCustomer(ctx, body.(wenmar.CreateCustomerRequest))
		if err != nil {
			return nil, err
		}
		return resp.JSON201, nil
	})
}

func runCustomersUpdate(cmd *cobra.Command, args []string) error {
	return runUpdate(cmd, args, "customers", idPath("/customers/"), "Customer updated.", func(id int) (any, error) {
		req := wenmar.UpdateCustomerRequest{}
		applyCustomerUpdateFlags(&req)
		return req, nil
	}, func(ctx context.Context, client *wenmar.Client, id int, body any) (any, error) {
		resp, err := client.UpdateCustomer(ctx, id, body.(wenmar.UpdateCustomerRequest))
		if err != nil {
			return nil, err
		}
		return resp.JSON200, nil
	})
}

// splitName, parseLabelValue, applyCustomerFlags, applyCustomerUpdateFlags:
// copy verbatim from the deleted customers.go (lines 358-477).
```

`cmd/vehicles_extras.go`: not needed — re-verify. The vehicles create/update use shared flag vars across BOTH commands (vehicleVin etc. bound on both create and update). The generated file binds body fields per-command with per-resource var names, and create_vehicle/update_vehicle body fields differ (create requires make/model/year/customer_id; update only has make + optionals). Shared vars across two generated commands would collide (bodyFieldVarName is resource-scoped, not command-scoped: `vehicleVin` would be declared once per file — actually fine, the var is emitted once and bound on both commands only if both list Vin as a BodyField). Verify in smoke output: gen_vehicles.go binds `vehiclesVin` on create AND update if both ops carry vin. If the collision is handled (one var, two bindings), vehicles needs NO companion. Check `flag_overrides` for create_vehicle/update_vehicle in gen_overrides.yaml — none exist for update_vehicle, so update_vehicle's vin/submodel/... flags derive from its spec body. Both ops carry `vin` in their bodies → one `vehiclesVin` var bound to both commands → correct behavior, matches hand-written. BUT the hand-written vehicles create marks make/model/year/customer-id required with "(required)" help suffix — generator does that from spec `required` arrays. CONCLUSION: no vehicles companion; delete vehicles.go after verifying smoke output parity.

`cmd/work_orders_extras.go`:

```go
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/output"
	wenmar "github.com/wenmar-pro/wenmar-sdk/go/wenmar"
)

// Hand-written companion to generated workorders commands. show + tab
// commands need the truncation check and per-tab SDK dispatch that the
// per-operation generator does not model.

var workOrdersShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single work order by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrdersShow,
}

var workOrdersEstimateCmd = &cobra.Command{
	Use:   "estimate <id>",
	Short: "Show the estimate tab (services) for a work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrdersTab("estimate"),
}

var workOrdersWipCmd = &cobra.Command{
	Use:   "wip <id>",
	Short: "Show the work-in-progress tab (services) for a work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrdersTab("wip"),
}

var workOrdersInspectionCmd = &cobra.Command{
	Use:   "inspection <id>",
	Short: "Show the inspection tab (inspection reports) for a work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrdersTab("inspection"),
}

var workOrdersPartsCmd = &cobra.Command{
	Use:   "parts <id>",
	Short: "Show the parts tab (services) for a work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrdersTab("parts"),
}

var workOrdersPaymentsCmd = &cobra.Command{
	Use:   "payments <id>",
	Short: "Show the payments tab (payments) for a work order",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkOrdersTab("payments"),
}

func init() {
	workordersCmd.AddCommand(workOrdersShowCmd, workOrdersEstimateCmd, workOrdersWipCmd,
		workOrdersInspectionCmd, workOrdersPartsCmd, workOrdersPaymentsCmd)
}

// runWorkOrdersShow, fetchWorkOrderTab, runWorkOrdersTab: copy verbatim from
// the deleted work_orders.go (lines 121-233), renaming parent references
// (workOrdersCmd → workordersCmd) and resource slug "work_orders" →
// "workorders" in breadcrumbs/showBreadcrumbs calls.
```

CAUTION — var name collisions: the generated `gen_workorders.go` will ALSO emit `workordersShowCmd` etc. for ops it owns. After the exclusion list above, `show_work_order*` ops are excluded, so the generated file emits only list/create/update/delete — no collision with the companion's show/tab vars. But the companion names its vars `workOrdersShowCmd` while the generated parent is `workordersCmd`. `runWorkOrdersTab` references the resource slug `"work_orders"` in breadcrumbs — change to `"workorders"`. Also Phase 1 Task 7's `Args: cobra.NoArgs` on parents: the generated parent now carries it (Task 1), companions must NOT redeclare the parent.

`cmd/drivers.go` and `cmd/statements.go` and `cmd/vendors.go`, `cmd/locations.go`, `cmd/account.go`, `cmd/service_categories.go`: DELETE — the generator owns them fully (Task 2's action handlers cover service categories; Task 1's aliases cover locations/account which are generated as resources with one show command each).

- [ ] **Step 4: The cutover commit structure**

This task's changes land as ONE commit per file family to keep review possible, but they only compile together. Land as a single cutover commit:

1. Write all companion files (tags.go rewrite, customers_extras.go, work_orders_extras.go).
2. Update gen_overrides.yaml (exclusions + new commands: entries).
3. Run `make generate` → generated files land in cmd/ as `gen_*.go`.
4. Delete the ten twin files: `customers.go vehicles.go work_orders.go drivers.go vendors.go statements.go service_categories.go account.go locations.go tags.go` (tags.go is replaced by its rewrite, which is a Modify not Delete).
5. Drop `cmd/gen_*.go` from `.gitignore` so generated files are committed.
6. Delete the `//go:build !generated` lines — NO LONGER NEEDED: twins are gone; generated files carry no build tag. The generated header becomes just `// Code generated by gencli. DO NOT EDIT.` — update the HeaderComment in emitGroup (gen.go:193) to drop `//go:build generated`.
7. Update Makefile: delete `test-generated` target (obsolete); `generate` target unchanged.
8. `go build ./... && go test ./...` — ALL tests must pass, including integration tests that reference deleted symbols (update test references: e.g. `vehiclesCreateCmd` var name in surface tests may have changed to `vehiclesCreateCmd` vs generated `vehiclesCreateCmd` — the generated cmdVarName produces `vehiclesCreateCmd` for resource "vehicles" command "create": toCamelCase("vehicles")="vehicles" + titleCase("create")="Create" + "Cmd" = `vehiclesCreateCmd`. Same name. drivers: `driversCreateCmd` same. workorders: toCamelCase("workorders")="workorders" + "Create" + "Cmd" = `workordersCreateCmd` — the old var was `workOrdersCreateCmd` (capital O). Tests referencing `workOrdersCreateCmd` must be renamed to `workordersCreateCmd`. Grep and fix.)

The full command sequence for the cutover:

```bash
make generate
git add cmd/gen_*.go cmd/*_extras.go cmd/tags.go cmd/gen_overrides.yaml
git rm cmd/customers.go cmd/vehicles.go cmd/work_orders.go cmd/drivers.go cmd/vendors.go cmd/statements.go cmd/service_categories.go cmd/account.go cmd/locations.go
# fix test symbol references, then:
go build ./... && go test ./...
```

- [ ] **Step 5: Surface snapshot regeneration**

The surface changes (renames, aliases, removed `--tag-ids` filter, removed `--remote` etc.). Regenerate:

```bash
go run ./cmd/wenmar surface-snapshot > surface-snapshot.json
git diff surface-snapshot.json | head -50   # eyeball: renames with aliases present, no unexpected losses
```

Verify alias entries exist:

```bash
grep -A2 '"workorders"' surface-snapshot.json | head -5
grep '"work_orders"' surface-snapshot.json | head -3
```

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: generator is the source of truth — committed generated commands replace hand-written twins"
```

### Task 5: Naming migration to squashed compounds with permanent aliases

**Files:**
- Modify: `cmd/gen_overrides.yaml` (resource names + aliases)
- Regenerate: all `cmd/gen_*.go`
- Modify: `surface-snapshot.json`
- Modify: `README.md`, `skills/wenmar/SKILL.md` (command names)
- Test: `cmd/integration_test.go` (alias resolution tests)

Lands TOGETHER WITH Task 4's cutover or immediately after — aliases make it non-breaking. The overrides drive everything:

- [ ] **Step 1: Update gen_overrides.yaml resources**

Every `resource:` field with a multi-word name changes:

| Old resource | New canonical | `groups:` entry |
|---|---|---|
| `work_orders` | `workorders` | `aliases: [work_orders, wo]`, `short: Manage work orders` |
| `service_categories` | `servicecategories` | `aliases: [service-categories, sc]`, `short: Manage service categories` |

Plus `groups:` for the single-word resources (Shorts only):

```yaml
groups:
  customers:
    short: Manage customers
  vehicles:
    short: Manage vehicles
  drivers:
    short: Manage drivers
  vendors:
    short: Manage vendors
  statements:
    short: Manage statements
  locations:
    short: Manage locations
  account:
    short: Show the current account
  tags:
    short: Manage customer and vehicle tags
```

Nested subcommand names: `customers workorders` (alias `work-orders`), `vehicles workorders` (alias `work-orders`) — set in the commands: entries added in Task 4.

- [ ] **Step 2: Regenerate + verify**

```bash
make generate
go build ./... && go test ./...
go run ./cmd/wenmar surface-snapshot > surface-snapshot.json
```

Manual verification of the alias resolution (Phase 0's build is fresh):

```bash
go build -o wenmar ./cmd/wenmar
./wenmar workorders list --help
./wenmar work_orders list --help    # alias works
./wenmar wo list --help            # alias works
./wenmar servicecategories list --help
./wenmar service-categories list --help   # alias
./wenmar sc list --help            # alias
./wenmar customers workorders 42 --help  # nested canonical
./wenmar customers work-orders 42 --help # nested alias
```

All 9 must print help, exit 0.

- [ ] **Step 3: Readability check (spec §2.5 caveat)**

```bash
./wenmar --help
```

Eyeball `workorders` and `servicecategories` in the command list. If `servicecategories` is unreadable cold (the spec's stated concern), the revert is cheap: change resource back to `service-categories` with `aliases: [servicecategories, sc]` in groups, regenerate. Record the decision in the commit message either way.

- [ ] **Step 4: Alias tests**

Add to `cmd/integration_test.go`:

```go
func TestResourceAliasesResolve(t *testing.T) {
	cases := []struct {
		args []string
		want string // substring of the help output proving the right command ran
	}{
		{[]string{"workorders", "list", "--help"}, "List all work orders"},
		{[]string{"work_orders", "list", "--help"}, "List all work orders"},
		{[]string{"wo", "list", "--help"}, "List all work orders"},
		{[]string{"servicecategories", "list", "--help"}, "List all service categories"},
		{[]string{"service-categories", "list", "--help"}, "List all service categories"},
		{[]string{"sc", "list", "--help"}, "List all service categories"},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			out, err := execute(tc.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(out, tc.want) {
				t.Errorf("%v did not resolve to the expected command; got:\n%s", tc.args, out)
			}
		})
	}
}
```

(If the generated Short for workorders list differs from "List all work orders" — check the summary override in gen_overrides.yaml, which sets it — adjust `want` to the actual Short.)

- [ ] **Step 5: Update README + SKILL.md command names**

Replace all `work_orders` → `workorders`, `service-categories` → `servicecategories` in examples; note aliases in both files ("`work_orders` and `wo` still work as aliases").

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: squashed-compound resource names with permanent aliases (workorders, servicecategories)"
```

### Task 6: Golden tests for the generator

**Files:**
- Create: `cmd/gencli/testdata/fixture-spec.yaml` (minimal spec)
- Create: `cmd/gencli/testdata/golden/` (golden .go files)
- Create: `cmd/gencli/golden_test.go`
- Modify: `cmd/gencli/gen.go` (spec path indirection if needed)

Golden tests lock the generator's emission format. They use a SMALL fixture spec — not the live spec — so SDK spec changes never break them; the regen-drift CI job (Task 7) covers live-spec sync.

- [ ] **Step 1: Create the fixture spec**

```yaml
paths:
  "/widgets":
    get:
      summary: index
      operationId: list_widgets
      x-paginated: true
      responses:
        "200":
          description: ok
    post:
      summary: create
      operationId: create_widget
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                widget:
                  type: object
                  properties:
                    name:
                      type: string
                    size:
                      type: integer
                  required: [name]
      responses:
        "201":
          description: ok
  "/widgets/{id}":
    get:
      summary: show
      operationId: show_widget
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        "200":
          description: ok
    delete:
      summary: destroy
      operationId: delete_widget
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        "200":
          description: ok
  "/widgets/{id}/archive":
    patch:
      summary: archive
      operationId: archive_widget
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties: {}
      responses:
        "200":
          description: ok
```

- [ ] **Step 2: Failing test**

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoldenEmission locks the generator's output format for a fixture spec.
// To regenerate goldens: go test ./cmd/gencli/ -run TestGoldenEmission -update-golden
func TestGoldenEmission(t *testing.T) {
	spec, err := loadSpec(filepath.Join("testdata", "fixture-spec.yaml"))
	if err != nil {
		t.Fatalf("loadSpec: %v", err)
	}
	overrides := &Overrides{
		Groups: map[string]GroupOverride{
			"widgets": {Short: "Manage widgets", Aliases: []string{"w"}},
		},
	}

	groups := groupOperations(spec, overrides)
	if len(groups) != 1 || groups[0].Resource != "widgets" {
		t.Fatalf("expected one widgets group, got %+v", groups)
	}

	update := flag.Bool("update-golden", false, "rewrite golden files")
	flag.Parse() // note: uses testing flags — see correction below

	for _, group := range groups {
		code, err := emitGroup(group, spec, overrides)
		if err != nil {
			t.Fatalf("emitGroup: %v", err)
		}
		goldenPath := filepath.Join("testdata", "golden", "gen_"+group.Resource+".go")
		if *update {
			if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(goldenPath, []byte(code), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("golden missing (run with -update-golden): %v", err)
		}
		if strings.TrimSpace(code) != strings.TrimSpace(want) {
			t.Errorf("emitted code drifted from golden %s:\n--- got ---\n%s\n--- want ---\n%s",
				goldenPath, code, want)
		}
	}
}
```

CORRECTION — `flag.Bool` + `flag.Parse()` inside a test conflicts with the testing framework's flag handling. Use the standard env-var or os.Args approach:

```go
func TestGoldenEmission(t *testing.T) {
	spec, err := loadSpec(filepath.Join("testdata", "fixture-spec.yaml"))
	if err != nil {
		t.Fatalf("loadSpec: %v", err)
	}
	overrides := &Overrides{
		Groups: map[string]GroupOverride{
			"widgets": {Short: "Manage widgets", Aliases: []string{"w"}},
		},
	}

	groups := groupOperations(spec, overrides)
	if len(groups) != 1 || groups[0].Resource != "widgets" {
		t.Fatalf("expected one widgets group, got %+v", groups)
	}

	update := os.Getenv("UPDATE_GOLDEN") == "1"

	for _, group := range groups {
		code, err := emitGroup(group, spec, overrides)
		if err != nil {
			t.Fatalf("emitGroup: %v", err)
		}
		goldenPath := filepath.Join("testdata", "golden", "gen_"+group.Resource+".go")
		if update {
			if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(goldenPath, []byte(code), 0o644) {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("golden missing (run with UPDATE_GOLDEN=1): %v", err)
		}
		if strings.TrimSpace(code) != strings.TrimSpace(want) {
			t.Errorf("emission drifted from golden %s", goldenPath)
		}
	}
}
```

- [ ] **Step 3: Generate goldens, verify, commit**

```bash
UPDATE_GOLDEN=1 go test ./cmd/gencli/ -run TestGoldenEmission -v
go test ./cmd/gencli/ -run TestGoldenEmission -v   # green now
# Compile-check the golden file (catches emission of non-compiling code):
mkdir -p /tmp/opencode/golden-build && cp cmd/gencli/testdata/golden/gen_widgets.go /tmp/opencode/golden-build/
```

The golden file references `runShow`, `runListPaginated`, `runCreate`, `runDelete`, `runActionNoBody`, `idPath`, helper vars — it can't compile standalone without the cmd package context; the compile check happens implicitly via Task 7's drift job (which regenerates into the real cmd/ and runs `go build`). Golden tests assert FORMAT, the drift job asserts COMPILABILITY. Note this in the test's doc comment.

Add unit tests for the pure helpers while here (they're cheap and lock behavior):

```go
func TestKebabCase(t *testing.T) {
	cases := map[string]string{
		"full_name":  "full-name",
		"lastVisit":  "last-visit",  // NOTE: kebabCase only replaces underscores; camel input passes through
	}
	for in, want := range cases {
		if got := kebabCase(in); got != want {
			t.Errorf("kebabCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSingularize(t *testing.T) {
	cases := map[string]string{
		"workorders":        "workorder",
		"servicecategories": "servicecategory",
		"customers":         "customer",
		"companies":         "company",
	}
	for in, want := range cases {
		if got := singularize(in); got != want {
			t.Errorf("singularize(%q) = %q, want %q", in, got, want)
		}
	}
}
```

VERIFY BEFORE COMMITTING: `singularize("servicecategories")` — the current implementation (gen.go:1195-1207) strips a trailing "s" → "servicecategorie". The hand-written summary strings never hit this path (summaries come from overrides), but Task 2's `emitActionNoBodyHandler` uses `titleCase(singularize(resource))` in summary strings → "Servicecategorie deactivated." — WRONG. Fix singularize in this task:

```go
func singularize(s string) string {
	// Special cases for compound words
	if strings.HasSuffix(s, "workorders") {
		return "work order"
	}
	if strings.HasSuffix(s, "categories") {
		return strings.TrimSuffix(s, "categories") + "category"
	}
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}
```

`"categories"` must be checked BEFORE the generic "ies" rule (which would produce "categor-y" → wrong: "servicecategories" → "servicecategor-y"... actually the ies rule gives "servicecategor" + "y" = "servicecategory" — check: trim "ies" (3 chars) from "servicecategories" → "servicecategor" + "y" = "servicecategory". CORRECT already! And "workorders" → strips "s" → "workorder" — the hand-written special case `work_orders` is stale after renaming. Simplify: DELETE the work_orders special case, add nothing — the ies rule handles categories, the s rule handles the rest. Verify with the test above; drop the special cases entirely:

```go
func singularize(s string) string {
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}
```

- [ ] **Step 4: Commit**

```bash
git add cmd/gencli/
git commit -m "test(gencli): golden emission tests + helper unit tests; simplify singularize"
```

### Task 7: CI regen-drift gate

**Files:**
- Modify: `.github/workflows/ci.yml` (new job)
- Modify: `Makefile` (regen-driff target; test-generated removed)

The committed generated files must never drift from the spec. CI regenerates from the pinned SDK's spec and fails on any diff.

- [ ] **Step 1: Makefile target**

```makefile
# Regenerate commands from the spec and fail if committed output drifted.
# Requires the wenmar-sdk worktree (CI clones it via the drift job below).
regen-drift: generate
	git diff --exit-code -- cmd/gen_*.go \
		|| (echo "Generated commands drifted from the spec. Run: make generate" && exit 1)
```

Delete the `test-generated` target (obsolete post-cutover) and its `.PHONY` entry.

- [ ] [ ] **Step 2: CI job**

```yaml
  regen-drift:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          path: wenmar-cli
      - uses: actions/checkout@v4
        with:
          repository: Wenmar-Pro/wenmar-sdk
          path: wenmar-sdk
          token: ${{ secrets.GITHUB_TOKEN }}
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.27"
      - name: Configure Go module auth
        run: |
          git config --global url."https://x-access-token:${{ secrets.GITHUB_TOKEN }}@github.com/Wenmar-Pro/".insteadOf "https://github.com/Wenmar-Pro/"
      - name: Regen and diff
        run: |
          cd wenmar-cli
          go mod download
          make regen-drift
        env:
          SPEC_PATH: ../wenmar-sdk/spec/openapi.enriched.yaml
```

Wait — the Makefile's generate target hardcodes `SPEC_PATH ?= ../wenmar-sdk/spec/openapi.enriched.yaml` relative to the CLI repo root. In the CI layout above (`wenmar-cli/` and `wenmar-sdk/` siblings), `../wenmar-sdk` resolves correctly. But the generate command runs `go run ./cmd/gencli` — it needs the module deps, fine. However `go mod download` for the private SDK requires the git config — done in the previous step. One wrinkle: the SDK checkout needs its own go.mod resolution for nothing — gencli only reads the YAML spec; no Go build of the SDK happens. The CLI build needs the published SDK module (go.mod pin), which the git-config rewrite handles. Correct as written.

- [ ] **Step 3: Verify locally**

```bash
make regen-drift
```

Expected: pass (no drift). Then intentionally touch a generated file and confirm failure:

```bash
echo "// drifted" >> cmd/gen_vendors.go
make regen-drift; echo "exit=$?"    # want exit 1
git checkout cmd/gen_vendors.go
```

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci.yml Makefile
git commit -m "ci: regen-drift job fails when generated commands drift from the spec"
```

### Task 8: CLI conventions skill

**Files:**
- Create: `.opencode/skill/cli-conventions/SKILL.md`

Per spec §2.6, the naming rules live in a project skill so future agents follow them.

- [ ] **Step 1: Write the skill**

```markdown
# CLI Conventions

Rules for naming and shaping wenmar-cli commands, derived from basecamp-cli /
hey-cli precedent. Apply these whenever adding or regenerating commands.

## Naming

- Resource nouns are squashed compounds: `todolist`, `workorders`,
  `servicecategories` — never `todo-list`, `todo_list`, or `work-orders` as
  the canonical name.
- Multi-word **flags** are kebab-case: `--full-name`, `--message-board`,
  `--decode-vin`.
- Subcommands are space-separated verb/noun tokens: `customers workorders
  list`, `basecamp profile create`. Hyphens inside a single token appear
  only in flags (`--plate-state`) or subcommand names that must be one token
  (`decode-vin`, `move-up`).
- snake_case appears ONLY in API/JSON field names — never as a CLI-facing
  token. Env vars are SCREAMING_SNAKE (`WENMAR_LOCATION_ID`).
- Renames ship with permanent backward-compatible aliases via
  `gen_overrides.yaml` (`groups.<resource>.aliases`). Aliases never expire.

## Command surface rules

- Parent commands get `Args: cobra.NoArgs` + help RunE; typo'd subcommands
  must exit non-zero with suggestions.
- Every list command gets the `ls` alias. Every resource gets its `groups:`
  short. Canonical `list` Shorts say what pagination applies ("paginated via
  the Link header") ONLY when the SDK exposes a WithPagination variant.
- Delete commands always get `--dry-run`.
- Add `Example:` fields when adding commands (generator reads them from
  overrides `examples:` — Phase 3 wires this).
- Flag removals are releases of record: document in README migration table;
  never remove an alias.

## Codegen rules

- Resource commands come from `make generate` (gen_overrides.yaml is the
  human-tuned surface). Hand-written companions exist only for
  non-derivable logic: `cmd/tags.go`, `cmd/customers_extras.go`,
  `cmd/work_orders_extras.go`. Do not add new hand-written resource files —
  extend the generator instead.
- Generated files (`cmd/gen_*.go`) are committed. CI's regen-drift job
  fails if they drift from the spec.
- Golden tests (`cmd/gencli/golden_test.go`) lock emission format; update
  with UPDATE_GOLDEN=1 when the generator intentionally changes output.
```

- [ ] **Step 2: Verify the skill loads**

```bash
ls .opencode/skill/cli-conventions/SKILL.md
```

(If the project uses a different skill location convention — check for existing `.opencode/` structure — place accordingly.)

- [ ] **Step 3: Commit**

```bash
git add .opencode/
git commit -m "docs: CLI conventions skill (naming + codegen rules)"
```

### Task 9: Makefile + docs cleanup

**Files:**
- Modify: `Makefile` (dead targets), `README.md`, `docs/superpowers/specs/...` (progress note)

- [ ] **Step 1: Makefile**

- Delete `test-generated` (Task 7 did this if not already).
- `generate` stays; add a comment that output is committed.

- [ ] **Step 2: README**

- Update the "Agent discovery" and "Usage" examples to canonical names.
- Add a "Generated commands" note: `make generate` regenerates; drift is CI-enforced; renames carry permanent aliases.

- [ ] **Step 3: Commit**

```bash
git add Makefile README.md
git commit -m "docs: post-cutover README/Makefile updates"
```

---

## Self-review notes

- **Spec coverage (§Phase 2):** 2.1 action handlers → Task 2 (+runActionNoBody/runSeedAction runners). Generator `request_struct` auto-derivation — intentionally dropped (YAGNI: every current op needing it already carries an explicit request_struct or derives; no op relies on auto-derivation today; the golden tests + smoke generation prove emission). 2.1 pagination honesty → Task 3 (queryParam+paginated classification) and verified classification of `x-paginated` ops (vendors/drivers/statements generate plain `list` — matching their actual SDK surfaces — which is the honest emission; their Shorts via overrides stay accurate). 2.1 examples/breadcrumbs from generator — deferred to Phase 3 (spec §3.3) where the whole help overhaul lives. 2.2 generic runners — DROPPED from this plan by scope decision: the cutover deletes the twins and the boilerplate dies with them; `runShow`/`runList`/etc. remain the shared skeleton layer the generated one-line handlers call; converting them to generics adds no behavior and risks the whole cutover on a mechanical rewrite. Recorded as follow-up. 2.3 cutover → Task 4. 2.4 golden tests + drift gate → Tasks 6-7. 2.5 naming → Task 5. 2.6 conventions skill → Task 8. check-published/surface-diff in CI → already landed in Phase 0's CI work.
- **Deliberate surface changes** (documented in cutover commit): `--tag-ids` filter on customers list is dropped (array query params skipped; companion mechanism documented). `--remote` on tui was already dead (Phase 1). Nested `customers work-orders` → `customers workorders` with alias.
- **Type consistency:** `runActionNoBody`/`runSeedAction` defined in Task 2 and used by Task 2's emitters. `GroupOverride`/`CommandOverride.Aliases` defined Task 1, consumed Tasks 4-5. Companion files reference generated parent vars (`workordersCmd`) — naming verified via cmdVarName derivation.
- **Known risks:** (1) Generated var names differ from hand-written (`workOrdersCreateCmd` → `workordersCreateCmd`) — tests referencing old names must be updated in the cutover commit; grep `workOrders.*Cmd` in cmd/*_test.go. (2) The `has-filters` if-chain emission is the one jennifer pattern with no precedent in the current codebase — the golden test locks it. (3) Deleting `--tag-ids` changes the surface snapshot — the diff is expected and part of the cutover commit, not a failure.