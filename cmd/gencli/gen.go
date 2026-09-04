package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dave/jennifer/jen"
)

// CommandGroup groups operations by CLI resource.
type CommandGroup struct {
	Resource string
	Commands []GenCommand
}

// GenCommand represents one cobra command to emit.
type GenCommand struct {
	OperationID      string
	Resource         string
	Command          string
	Summary          string
	Method           string
	Path             string
	PathParams       []Parameter
	QueryParams      []Parameter
	RequestBody      *RequestBody
	IsPaginated      bool
	ExtraPathParams  []Parameter
	HasIDParam       bool
	IDType           string
	SDKMethod        string
	RequestStruct    string
	BodyFields       []BodyField
	PositionalArg    string
	QueryParamStruct string
	QueryFields      []BodyField
	Tab              string   // work order tab name
	Aliases          []string // command aliases from overrides
	ActionSummary    string   // past-tense success message for action runners
	WrapperKey       string   // request-body wrapper object key ("driver", "vehicle", "work_order"); "" = flat
	ResponseField    string   // "JSON200" or "JSON201"
	IDParam          string   // name of the path param treated as the positional id (default "id")
	Example          string   // example usage block for cobra help
}

// BodyField represents a scalar field from the request body schema
// that becomes a CLI flag.
type BodyField struct {
	JSONName  string // snake_case (e.g. "full_name")
	GoName    string // PascalCase (e.g. "FullName")
	FlagName  string // kebab-case (e.g. "full-name")
	Type      string // "string", "integer", "boolean", "array"
	Required  bool
	IsPointer bool // true if the SDK struct field is a pointer type
	HelpText  string
	NoFlag    bool   // true if the field appears in the body struct but has no CLI flag (arrays)
	Default   string // optional default value for the flag ("" = type zero value)
	FlagType  string // "intslice" to bind via IntSliceVar (array params)
}

// groupOperations reads the spec and overrides to produce command groups.
func groupOperations(spec *Spec, overrides *Overrides) []CommandGroup {
	groupMap := make(map[string][]GenCommand)
	excluded := make(map[string]bool)
	for _, id := range overrides.Exclude {
		excluded[id] = true
	}

	// Track seen command var names to detect duplicates within a resource.
	seenVarNames := make(map[string][]string) // resource -> list of var names

	// Iterate paths and methods deterministically so the "first wins" dedup
	// below is stable across runs (Go map iteration order is random).
	paths := make([]string, 0, len(spec.Paths))
	for path := range spec.Paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		item := spec.Paths[path]
		methods := make([]string, 0, len(item))
		for method := range item {
			methods = append(methods, method)
		}
		sort.Strings(methods)
		for _, method := range methods {
			op := item[method]
			if op.OperationID == "" || excluded[op.OperationID] {
				continue
			}
			cmd := buildCommand(spec, op, method, path, overrides)
			if cmd == nil {
				continue
			}
			// Check for duplicate command var names within the same resource.
			varName := cmdVarName(*cmd)
			for _, existing := range seenVarNames[cmd.Resource] {
				if existing == varName {
					// Skip duplicate — the first one wins.
					goto nextOp
				}
			}
			seenVarNames[cmd.Resource] = append(seenVarNames[cmd.Resource], varName)
			groupMap[cmd.Resource] = append(groupMap[cmd.Resource], *cmd)
		nextOp:
		}
	}

	var groups []CommandGroup
	for resource, cmds := range groupMap {
		sort.Slice(cmds, func(i, j int) bool {
			return cmds[i].Command < cmds[j].Command
		})
		groups = append(groups, CommandGroup{Resource: resource, Commands: cmds})
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Resource < groups[j].Resource
	})
	return groups
}

func buildCommand(spec *Spec, op Operation, method, path string, overrides *Overrides) *GenCommand {
	cmd := &GenCommand{
		OperationID: op.OperationID,
		Method:      method,
		Path:        path,
		Summary:     op.Summary,
		IDType:      "int",
		IsPaginated: op.XPaginated,
		IDParam:     "id",
	}

	// Response code from the spec's declared responses.
	cmd.ResponseField = "JSON200"
	if _, ok := op.Responses["201"]; ok {
		cmd.ResponseField = "JSON201"
	}

	// Apply override fields (merge-style, no early return) so cmd.IDParam
	// is known before the path-param loop below.
	ov := overrides.Commands[op.OperationID] // zero value if absent
	if ov.Resource != "" {
		cmd.Resource = ov.Resource
	}
	if ov.Command != "" {
		cmd.Command = ov.Command
	}
	if ov.Summary != "" {
		cmd.Summary = ov.Summary
	}
	if ov.Method != "" {
		cmd.SDKMethod = ov.Method
	}
	if ov.RequestStruct != "" {
		cmd.RequestStruct = ov.RequestStruct
		cmd.BodyFields, cmd.WrapperKey = parseBodyFields(spec, op, ov.RequestStruct, overrides.FlagOverrides[op.OperationID])
	} else if op.XWenmarRequestSchema != "" {
		cmd.RequestStruct = op.XWenmarRequestSchema
		cmd.BodyFields, cmd.WrapperKey = parseBodyFields(spec, op, op.XWenmarRequestSchema, overrides.FlagOverrides[op.OperationID])
	}
	if ov.PositionalArg != "" {
		cmd.PositionalArg = ov.PositionalArg
	}
	if ov.QueryParamStruct != "" {
		cmd.QueryParamStruct = ov.QueryParamStruct
		cmd.QueryFields = extractQueryFields(op, ov.QueryParamStruct, overrides.FlagOverrides[op.OperationID])
	}
	if ov.Paginated != nil {
		cmd.IsPaginated = *ov.Paginated
	}
	if ov.Tab != "" {
		cmd.Tab = ov.Tab
	}
	if len(ov.Aliases) > 0 {
		cmd.Aliases = ov.Aliases
	}
	if ov.ActionSummary != "" {
		cmd.ActionSummary = ov.ActionSummary
	}
	if ov.IdParam != "" {
		cmd.IDParam = ov.IdParam
	}
	if ov.Example != "" {
		cmd.Example = ov.Example
	}

	// Path-param loop: the param named cmd.IDParam is the positional id.
	for _, p := range op.Parameters {
		if p.In == "path" {
			cmd.PathParams = append(cmd.PathParams, p)
			if p.Name == cmd.IDParam {
				cmd.HasIDParam = true
				if p.Schema.Type == "string" {
					cmd.IDType = "string"
				}
			} else {
				cmd.ExtraPathParams = append(cmd.ExtraPathParams, p)
			}
		} else if p.In == "query" {
			cmd.QueryParams = append(cmd.QueryParams, p)
		}
	}
	cmd.RequestBody = op.RequestBody

	// Auto-derive resource/command only when no override provided one.
	if cmd.Resource == "" || cmd.Command == "" {
		resource, command := autoDerive(method, path)
		if cmd.Resource == "" {
			cmd.Resource = resource
		}
		if cmd.Command == "" {
			cmd.Command = command
		}
		if cmd.Resource == "" {
			return nil
		}
	}
	return cmd
}

func autoDerive(method, path string) (resource, command string) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 0 {
		return "", ""
	}
	resource = segments[0]
	hasID := strings.Contains(path, "{id}")
	switch method {
	case "get":
		if hasID {
			command = "show"
		} else {
			command = "list"
		}
	case "post":
		command = "create"
	case "patch":
		command = "update"
	case "delete":
		command = "delete"
	default:
		command = method
	}
	return resource, command
}

// emitGroup generates Go source for a command group.
func emitGroup(group CommandGroup, spec *Spec, overrides *Overrides, buildTag string) (string, error) {
	f := jen.NewFile("cmd")
	if buildTag != "" {
		f.HeaderComment("// Code generated by gencli. DO NOT EDIT.\n//go:build " + buildTag)
	} else {
		f.HeaderComment("// Code generated by gencli. DO NOT EDIT.")
	}

	// Collect all unique flag vars across commands to emit once.
	flagVarsSeen := make(map[string]string) // varName -> goType
	for _, cmd := range group.Commands {
		for _, p := range cmd.ExtraPathParams {
			vn := flagVarName(cmd.Resource, p.Name)
			flagVarsSeen[vn] = goType(p.Schema.Type)
		}
		if classifyCommand(cmd) == "delete" {
			dryRunVar := flagVarName(cmd.Resource, "delete_dry_run")
			flagVarsSeen[dryRunVar] = "bool"
		}
		// --all flag for paginated-with-params lists.
		if classifyCommand(cmd) == "listPaginatedWithParams" {
			flagVarsSeen[hasFiltersVarName(cmd)] = "bool"
		}
		// Body field flags.
		for _, bf := range cmd.BodyFields {
			if bf.NoFlag {
				continue
			}
			vn := bodyFieldVarName(cmd.Resource, bf.GoName)
			flagVarsSeen[vn] = goType(bf.Type)
		}
		// Query param flags.
		if classifyCommand(cmd) != "positionalArg" {
			for _, bf := range cmd.QueryFields {
				if bf.NoFlag {
					continue
				}
				vn := bodyFieldVarName(cmd.Resource, bf.GoName)
				if bf.FlagType == "intslice" {
					flagVarsSeen[vn] = "[]int"
				} else {
					flagVarsSeen[vn] = goType(bf.Type)
				}
			}
		}
	}

	// Emit all flag vars at the top, sorted for stability.
	for _, vn := range sortedKeys(flagVarsSeen) {
		f.Var().Id(vn).Id(flagVarsSeen[vn])
	}

	for _, cmd := range group.Commands {
		emitCommand(f, cmd, overrides)
	}

	// Emit has-filters helpers for paginated-with-params lists.
	for _, cmd := range group.Commands {
		if classifyCommand(cmd) != "listPaginatedWithParams" || len(cmd.QueryFields) == 0 {
			continue
		}
		emitHasFiltersFunc(f, cmd)
	}

	// Emit the parent command as a package-level var so companion files
	// (e.g. customers_extras.go) can register extra subcommands against it.
	parentVar := group.Resource + "Cmd"
	parentDict := jen.Dict{
		jen.Id("Use"):     jen.Lit(group.Resource),
		jen.Id("GroupID"): jen.Lit("resources"),
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
			parentDict[jen.Id("Aliases")] = jen.Index().String().Values(aliasLit...)
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
	f.Var().Id(parentVar).Op("=").Op("&").Qual("github.com/spf13/cobra", "Command").Values(parentDict)

	// Emit init() with flag registration + cobra registration.
	f.Func().Id("init").Params().BlockFunc(func(g *jen.Group) {
		if len(group.Commands) == 0 {
			return
		}
		// Register flags for each command.
		for _, cmd := range group.Commands {
			emitFlagRegistration(g, cmd)
		}
		g.Id(parentVar).Dot("AddCommand").Call(jen.ListFunc(func(l *jen.Group) {
			for _, cmd := range group.Commands {
				l.Id(cmdVarName(cmd))
			}
		}))
		g.Id("rootCmd").Dot("AddCommand").Call(jen.Id(parentVar))
	})

	return fmt.Sprintf("%#v", f), nil
}

// emitCommand emits a single cobra command variable + its RunE handler.
func emitCommand(f *jen.File, cmd GenCommand, overrides *Overrides) {
	varName := cmdVarName(cmd)
	cmdType := classifyCommand(cmd)

	dict := jen.Dict{
		jen.Id("Use"):   jen.Lit(cmd.Command + useArgsSuffix(cmdType, cmd)),
		jen.Id("Short"): jen.Lit(cmd.Summary),
		jen.Id("RunE"):  jen.Id(runHandlerName(cmd)),
	}
	if needsExactArgs(cmdType) {
		dict[jen.Id("Args")] = jen.Qual("github.com/spf13/cobra", "ExactArgs").Call(jen.Lit(1))
	}
	if len(cmd.Aliases) > 0 {
		aliasLit := make([]jen.Code, 0, len(cmd.Aliases))
		for _, a := range cmd.Aliases {
			aliasLit = append(aliasLit, jen.Lit(a))
		}
		dict[jen.Id("Aliases")] = jen.Index().String().Values(aliasLit...)
	}
	if cmd.Example != "" {
		dict[jen.Id("Example")] = jen.Lit(cmd.Example)
	}

	f.Var().Id(varName).Op("=").Op("&").Qual("github.com/spf13/cobra", "Command").Values(dict)

	emitHandler(f, cmd, cmdType, overrides)
}

func needsExactArgs(cmdType string) bool {
	switch cmdType {
	case "show", "showStr", "update", "delete", "actionCreate", "actionUpdate", "actionNoBody", "tab", "positionalArg":
		return true
	default:
		return false
	}
}

func classifyCommand(cmd GenCommand) string {
	// Explicit overrides take precedence.
	if cmd.PositionalArg != "" {
		return "positionalArg"
	}
	if cmd.Tab != "" {
		return "tab"
	}
	switch cmd.Method {
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
		if cmd.QueryParamStruct != "" && len(cmd.QueryFields) > 0 {
			return "queryParam"
		}
		return "list"
	case "post":
		if cmd.HasIDParam && cmd.RequestBody != nil && isSubAction(cmd) && len(cmd.BodyFields) > 0 {
			return "actionCreate" // e.g. merge — POST to /resource/{id}/action
		}
		if cmd.HasIDParam && cmd.RequestBody != nil && isSubAction(cmd) {
			return "actionNoBody" // POST sub-action with empty body
		}
		if cmd.RequestBody != nil && len(cmd.BodyFields) == 0 && !cmd.HasIDParam {
			return "seedAction" // e.g. seed_defaults_service_categories
		}
		if cmd.RequestBody != nil {
			return "create"
		}
		return "action"
	case "patch":
		if cmd.HasIDParam && cmd.RequestBody != nil && isSubAction(cmd) {
			if len(cmd.BodyFields) == 0 {
				return "actionNoBody" // e.g. service category deactivate/reactivate/move_up/move_down
			}
			return "actionUpdate" // e.g. transfer — PATCH to /resource/{id}/action
		}
		if cmd.RequestBody != nil {
			return "update"
		}
		return "action"
	case "delete":
		return "delete"
	default:
		return "list"
	}
}

func useArgsSuffix(cmdType string, cmd GenCommand) string {
	switch cmdType {
	case "show", "showStr", "update", "delete", "actionCreate", "actionUpdate", "actionNoBody", "tab":
		return " <id>"
	case "positionalArg":
		return " <" + cmd.PositionalArg + ">"
	default:
		return ""
	}
}

// isSubAction returns true if the path has a sub-action after {id}
// (e.g. /vehicles/{id}/transfer, /customers/{id}/merge).
func isSubAction(cmd GenCommand) bool {
	path := cmd.Path
	idPos := strings.Index(path, "{"+cmd.IDParam+"}")
	if idPos < 0 {
		return false
	}
	afterID := path[idPos+len("{"+cmd.IDParam+"}"):]
	return strings.Contains(afterID, "/")
}

func cmdVarName(cmd GenCommand) string {
	return toCamelCase(cmd.Resource) + titleCase(cmd.Command) + "Cmd"
}

func runHandlerName(cmd GenCommand) string {
	return "run" + titleCase(toCamelCase(cmd.Resource)) + titleCase(cmd.Command)
}

// wrapPtr wraps a value in strPtr() or boolPtr() for optional pointer fields.
func wrapPtr(goType string, val jen.Code) jen.Code {
	switch goType {
	case "integer":
		return jen.Id("intPtr").Call(val)
	case "string":
		return jen.Id("strPtr").Call(val)
	case "boolean":
		return jen.Id("boolPtr").Call(val)
	default:
		return jen.Id("strPtr").Call(val)
	}
}

// bodyFieldVarName generates the Go variable name for a body field flag.
// e.g. resource="drivers", goName="FullName" -> "driversCreateFullName"
func bodyFieldVarName(resource string, goName string) string {
	return toCamelCase(resource) + goName
}

// flagVarName generates the Go variable name for a flag.
// e.g. resource="drivers", param="customer_id" -> "driversCustomerID"
func flagVarName(resource string, paramName string) string {
	return toCamelCase(resource) + titleCase(toCamelCase(paramName))
}

// emitFlagRegistration emits the flag wiring in init().
func emitFlagRegistration(g *jen.Group, cmd GenCommand) {
	cmdVar := cmdVarName(cmd)

	// Extra path params become required flags.
	for _, p := range cmd.ExtraPathParams {
		flagName := kebabCase(p.Name)
		helpText := prettifyParamName(p.Name) + " (required)"
		varName := flagVarName(cmd.Resource, p.Name)
		args := []jen.Code{jen.Op("&").Id(varName), jen.Lit(flagName), DefaultForType(p.Schema.Type), jen.Lit(helpText)}
		g.Id(cmdVar).Dot("Flags").Call().Dot(flagBindMethod(p.Schema.Type)).Call(args...)
		g.Id(cmdVar).Dot("MarkFlagRequired").Call(jen.Lit(flagName))
	}

	// Body field flags.
	for _, bf := range cmd.BodyFields {
		if bf.NoFlag {
			continue
		}
		varName := bodyFieldVarName(cmd.Resource, bf.GoName)
		def := DefaultForType(bf.Type)
		if bf.Default != "" {
			def = jen.Lit(bf.Default)
		}
		args := []jen.Code{jen.Op("&").Id(varName), jen.Lit(bf.FlagName), def, jen.Lit(bf.HelpText)}
		g.Id(cmdVar).Dot("Flags").Call().Dot(flagBindMethod(bf.Type)).Call(args...)
		if bf.Required {
			g.Id(cmdVar).Dot("MarkFlagRequired").Call(jen.Lit(bf.FlagName))
		}
	}

	// Query param flags.
	if classifyCommand(cmd) != "positionalArg" {
		for _, bf := range cmd.QueryFields {
			if bf.NoFlag {
				continue
			}
			varName := bodyFieldVarName(cmd.Resource, bf.GoName)
			if bf.FlagType == "intslice" {
				g.Id(cmdVar).Dot("Flags").Call().Dot("IntSliceVar").Call(
					jen.Op("&").Id(varName), jen.Lit(bf.FlagName), jen.Nil(), jen.Lit(bf.HelpText),
				)
				continue
			}
			args := []jen.Code{jen.Op("&").Id(varName), jen.Lit(bf.FlagName), DefaultForType(bf.Type), jen.Lit(bf.HelpText)}
			g.Id(cmdVar).Dot("Flags").Call().Dot(flagBindMethod(bf.Type)).Call(args...)
			if bf.Required {
				g.Id(cmdVar).Dot("MarkFlagRequired").Call(jen.Lit(bf.FlagName))
			}
		}
	}

	// Delete gets --dry-run.
	if classifyCommand(cmd) == "delete" {
		dryRunVar := flagVarName(cmd.Resource, "delete_dry_run")
		g.Id(cmdVar).Dot("Flags").Call().Dot("BoolVar").Call(
			jen.Op("&").Id(dryRunVar),
			jen.Lit("dry-run"),
			jen.False(),
			jen.Lit("Preview what would be deleted without making an API call"),
		)
	}

	// Paginated-with-params lists get --all auto-pagination.
	if classifyCommand(cmd) == "listPaginatedWithParams" {
		allVar := hasFiltersVarName(cmd)
		g.Id(cmdVar).Dot("Flags").Call().Dot("BoolVar").Call(
			jen.Op("&").Id(allVar),
			jen.Lit("all"),
			jen.False(),
			jen.Lit("Fetch all pages by following pagination links"),
		)
	}
}

// prettifyParamName converts snake_case to "Customer ID" style.
func prettifyParamName(name string) string {
	parts := strings.Split(name, "_")
	for i, p := range parts {
		if p == "id" {
			parts[i] = "ID"
		} else {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

// sortedKeys returns sorted keys of a map.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// emitHandler emits the RunE function.
func emitHandler(f *jen.File, cmd GenCommand, cmdType string, overrides *Overrides) {
	handlerName := runHandlerName(cmd)

	f.Func().Id(handlerName).Params(
		jen.Id("cmd").Op("*").Qual("github.com/spf13/cobra", "Command"),
		jen.Id("args").Index().Id("string"),
	).Id("error").BlockFunc(func(g *jen.Group) {
		switch cmdType {
		case "show":
			emitShowHandler(g, cmd)
		case "showStr":
			emitShowStrHandler(g, cmd)
		case "list":
			emitListHandler(g, cmd)
		case "listPaginated":
			emitListPaginatedHandler(g, cmd)
		case "listPaginatedWithParams":
			emitListPaginatedWithParamsHandler(g, cmd)
		case "create":
			emitCreateHandler(g, cmd)
		case "delete":
			emitDeleteHandler(g, cmd)
		case "update":
			emitUpdateHandler(g, cmd)
		case "actionCreate":
			emitActionCreateHandler(g, cmd)
		case "actionUpdate":
			emitActionUpdateHandler(g, cmd)
		case "actionNoBody":
			emitActionNoBodyHandler(g, cmd)
		case "seedAction":
			emitSeedActionHandler(g, cmd)
		case "positionalArg":
			emitPositionalArgHandler(g, cmd)
		case "queryParam":
			emitQueryParamHandler(g, cmd)
		case "tab":
			emitTabHandler(g, cmd)
		default:
			emitActionHandler(g, cmd)
		}
	})
}

// sdkCallArgs builds the argument list for the SDK method call.
// e.g. for drivers show: ctx, driversCustomerID, id
func sdkCallArgs(cmd GenCommand, includeID bool) []jen.Code {
	args := []jen.Code{jen.Id("ctx")}
	// Extra path params (flags) come first, then the {id} positional.
	for _, p := range cmd.ExtraPathParams {
		args = append(args, jen.Id(flagVarName(cmd.Resource, p.Name)))
	}
	if includeID {
		args = append(args, jen.Id("id"))
	}
	return args
}

// requestPath builds the Sprintf expression for the request path.
func requestPathExpr(cmd GenCommand) jen.Code {
	path := cmd.Path
	// Replace {param} with %d or %s depending on type.
	fmtPath := path
	args := []jen.Code{}
	for _, p := range cmd.PathParams {
		placeholder := "{" + p.Name + "}"
		if p.Name == cmd.IDParam {
			// The id is the positional arg, replaced by args[0]
			fmtPath = strings.ReplaceAll(fmtPath, placeholder, "%s")
			args = append(args, jen.Id("args").Index(jen.Lit(0)))
		} else {
			// Extra path params are flags
			fmtPath = strings.ReplaceAll(fmtPath, placeholder, "%d")
			args = append(args, jen.Id(flagVarName(cmd.Resource, p.Name)))
		}
	}
	if len(args) == 0 {
		return jen.Lit(fmtPath)
	}
	allArgs := append([]jen.Code{jen.Lit(fmtPath)}, args...)
	return jen.Qual("fmt", "Sprintf").Call(allArgs...)
}

// parseBodyFields extracts scalar fields from the request body schema
// and maps them to Go struct field names. Array and object fields are
// skipped (they need hand-written flag logic). It returns the fields and
// the wrapper object key ("" for flat bodies).
func parseBodyFields(spec *Spec, op Operation, requestStruct string, flagOverrides map[string]FlagOverride) ([]BodyField, string) {
	if op.RequestBody == nil {
		return nil, ""
	}
	media, ok := op.RequestBody.Content["application/json"]
	if !ok {
		return nil, ""
	}
	schema := spec.Resolve(media.Schema)

	// Unwrap wrapper object (e.g. { customer: { ... } }).
	props := schemaProps(schema)
	if props == nil {
		return nil, ""
	}

	// Check if there's a single wrapper property that's an object.
	if len(props) == 1 {
		for propName, propSchema := range props {
			if propSchema.Type == "object" {
				// It's a wrapper — use the inner object's properties, keyed
				// by their dotted path (e.g. "customer.first_name").
				return extractScalarFields(spec, propSchema, requestStruct, propName, flagOverrides), propName
			}
			// Not a wrapper — flat body.
			_ = propName
		}
	}

	// Flat body (e.g. merge_customer: { source_customer_id: int }).
	return extractScalarFields(spec, schema, requestStruct, "", flagOverrides), ""
}

// extractScalarFields returns BodyFields for scalar properties of a schema.
// Array and object fields are skipped. flagOverrides are keyed by the field's
// dotted path (e.g. "customer.first_name") and override flag name, help text,
// and required-marking.
func extractScalarFields(spec *Spec, schema Schema, requestStruct, wrapper string, flagOverrides map[string]FlagOverride) []BodyField {
	props := schemaProps(schema)
	if props == nil {
		return nil
	}
	var required []string
	if schema.Required != nil {
		required = *schema.Required
	}

	var fields []BodyField
	for name, prop := range props {
		// Resolve property refs defensively (e.g. a property that is itself
		// a $ref to a component schema).
		if prop.Ref != "" && spec != nil {
			prop = spec.Resolve(prop)
		}
		// Skip objects — they need hand-written flag logic. Arrays are
		// captured so the anonymous struct stays assignable to the SDK's
		// inline struct, but they get no CLI flag.
		if prop.Type == "object" {
			continue
		}
		f := BodyField{
			JSONName:  name,
			GoName:    snakeToPascal(name),
			FlagName:  kebabCase(name),
			Type:      prop.Type,
			Required:  contains(required, name),
			IsPointer: !contains(required, name), // optional fields are pointers
			HelpText:  prettifyParamName(name),
		}
		if prop.Type == "array" {
			// Arrays appear in the anonymous struct for assignability but
			// have no CLI flag (they need hand-written flag logic).
			f.NoFlag = true
			f.IsPointer = true
		}
		if f.Required {
			f.HelpText += " (required)"
		}

		// Apply the flag override for this field's dotted path if present.
		if ov, ok := flagOverrides[dottedKey(wrapper, name)]; ok {
			if ov.Flag != "" {
				f.FlagName = ov.Flag
			}
			if ov.Help != "" {
				f.HelpText = ov.Help
			}
			if ov.Required != nil {
				f.Required = *ov.Required
				// IsPointer stays spec-driven: the SDK's generated struct
				// types pointer fields by schema optionality, not by CLI
				// flag requirements. Only the flag's required-marking follows
				// the override.
				if *ov.Required && !strings.HasSuffix(f.HelpText, ")") {
					f.HelpText += " (required)"
				}
			}
			if ov.Suppress {
				f.NoFlag = true
			}
			if ov.Default != "" {
				f.Default = ov.Default
			}
		}

		fields = append(fields, f)
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].FlagName < fields[j].FlagName
	})
	return fields
}

// dottedKey joins a wrapper prefix and field name into the override key used
// in gen_overrides.yaml (e.g. wrapper "customer" + name "first_name" ->
// "customer.first_name").
func dottedKey(wrapper, name string) string {
	if wrapper == "" {
		return name
	}
	return wrapper + "." + name
}

// schemaProps extracts the "properties" map from a Schema.
// The OpenAPI spec uses inline schemas, so we need to parse the
// raw YAML map. Since our Schema struct doesn't have Properties,
// we use a raw map approach.
func schemaProps(schema Schema) map[string]Schema {
	return schema.Properties
}

// snakeToPascal converts snake_case to PascalCase.
func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		// Special cases for Go naming. The SDK's generated request structs
		// use "Id" (not "ID") for the id suffix (e.g. CustomerId), so match
		// that for anonymous-struct assignability.
		if p == "id" {
			parts[i] = "Id"
		} else {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// extractQueryFields extracts BodyFields from query parameters.
func extractQueryFields(op Operation, queryParamStruct string, flagOverrides map[string]FlagOverride) []BodyField {
	var fields []BodyField
	for _, p := range op.Parameters {
		if p.In != "query" {
			continue
		}
		// Skip object query params — they need custom parsing. Array params
		// are captured so they can be bound via IntSliceVar when overridden.
		if p.Schema.Type == "object" {
			continue
		}
		f := BodyField{
			JSONName:  p.Name,
			GoName:    snakeToPascal(p.Name),
			FlagName:  kebabCase(p.Name),
			Type:      p.Schema.Type,
			Required:  p.Required,
			IsPointer: !p.Required,
			HelpText:  prettifyParamName(p.Name),
		}
		if p.Schema.Type == "array" {
			f.NoFlag = true
			f.IsPointer = true
		}
		if f.Required {
			f.HelpText += " (required)"
		}
		// Apply the flag override for this query field if present.
		if ov, ok := flagOverrides[p.Name]; ok {
			if ov.Flag != "" {
				f.FlagName = ov.Flag
			}
			if ov.Help != "" {
				f.HelpText = ov.Help
			}
			if ov.Required != nil {
				f.Required = *ov.Required
				// IsPointer stays spec-driven: the SDK's generated struct
				// types pointer fields by schema optionality, not by CLI
				// flag requirements. Only the flag's required-marking follows
				// the override.
				if *ov.Required && !strings.HasSuffix(f.HelpText, ")") {
					f.HelpText += " (required)"
				}
			}
			if ov.Suppress {
				f.NoFlag = true
			}
			if ov.FlagType != "" {
				f.FlagType = ov.FlagType
				f.NoFlag = false
			}
		}
		fields = append(fields, f)
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].FlagName < fields[j].FlagName
	})
	return fields
}

// pathFnForRunner wraps the path prefix in a func(args []string) string
// that the runners expect.
func pathFnForRunner(cmd GenCommand) jen.Code {
	if len(cmd.ExtraPathParams) == 0 {
		path := cmd.Path
		idPos := strings.Index(path, "{"+cmd.IDParam+"}")
		if idPos < 0 {
			return jen.Id("idPath").Call(jen.Lit(path))
		}
		// If the id is not the last segment (e.g. /service_categories/{id}/deactivate),
		// build a func that splices args[0] into the middle of the path.
		if idPos+len("{"+cmd.IDParam+"}") < len(path) {
			return jen.Func().Params(jen.Id("a").Index().Id("string")).Id("string").Block(
				jen.Return(pathWithIDExpr(cmd)),
			)
		}
		return jen.Id("idPath").Call(jen.Lit(path[:idPos]))
	}
	// Complex case: prefix includes flag vars and args[0] for the id.
	// Build: func(a []string) string { return fmt.Sprintf("/customers/%d/drivers/%s", driversCustomerID, a[0]) }
	return jen.Func().Params(jen.Id("a").Index().Id("string")).Id("string").Block(
		jen.Return(pathWithIDExpr(cmd)),
	)
}

// pathWithIDExpr builds the Sprintf expression for a path that includes
// both extra path params (flags) and the id positional arg.
func pathWithIDExpr(cmd GenCommand) jen.Code {
	path := cmd.Path
	fmtStr := path
	args := []jen.Code{}
	for _, p := range cmd.PathParams {
		placeholder := "{" + p.Name + "}"
		if p.Name == cmd.IDParam {
			fmtStr = strings.ReplaceAll(fmtStr, placeholder, "%s")
			args = append(args, jen.Id("a").Index(jen.Lit(0)))
		} else {
			fmtStr = strings.ReplaceAll(fmtStr, placeholder, "%d")
			args = append(args, jen.Id(flagVarName(cmd.Resource, p.Name)))
		}
	}
	allArgs := append([]jen.Code{jen.Lit(fmtStr)}, args...)
	return jen.Qual("fmt", "Sprintf").Call(allArgs...)
}

func emitShowHandler(g *jen.Group, cmd GenCommand) {
	resource := cmd.Resource
	g.Return(jen.Id("runShow").Call(
		jen.Id("cmd"), jen.Id("args"),
		jen.Lit(resource), jen.Lit("GET"),
		pathFnForRunner(cmd),
		jen.Func().Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
			jen.Id("id").Id("int"),
		).Params(jen.Any(), jen.Error()).Block(
			jen.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").Dot(sdkMethodNameFor(cmd)).Call(sdkCallArgs(cmd, true)...),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Id("err")),
			),
			jen.Return(jen.Id("resp").Dot("JSON200"), jen.Nil()),
		),
	))
}

func emitShowStrHandler(g *jen.Group, cmd GenCommand) {
	resource := cmd.Resource
	g.Return(jen.Id("runShowStr").Call(
		jen.Id("cmd"), jen.Id("args"),
		jen.Lit(resource), jen.Lit("GET"),
		pathFnForRunner(cmd),
		jen.Func().Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
			jen.Id("id").Id("string"),
		).Params(jen.Any(), jen.Error()).Block(
			jen.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").Dot(sdkMethodNameFor(cmd)).Call(sdkCallArgs(cmd, true)...),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Id("err")),
			),
			jen.Return(jen.Id("resp").Dot("JSON200"), jen.Nil()),
		),
	))
}

func emitListHandler(g *jen.Group, cmd GenCommand) {
	resource := cmd.Resource
	callArgs := sdkCallArgs(cmd, false)
	if len(cmd.QueryParams) > 0 {
		// SDK list methods that take a params struct accept nil for unfiltered.
		callArgs = append(callArgs, jen.Nil())
	}
	g.Return(jen.Id("runList").Call(
		jen.Id("cmd"),
		jen.Lit(resource),
		requestPathExpr(cmd),
		jen.Func().Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
		).Params(jen.Any(), jen.Error()).Block(
			jen.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").Dot(sdkMethodNameFor(cmd)).Call(callArgs...),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Id("err")),
			),
			jen.Return(jen.Id("resp").Dot("JSON200"), jen.Nil()),
		),
	))
}

func emitListPaginatedHandler(g *jen.Group, cmd GenCommand) {
	resource := cmd.Resource
	callArgs := sdkCallArgs(cmd, false)
	if len(cmd.QueryParams) > 0 {
		// SDK list methods that take a params struct accept nil for unfiltered.
		callArgs = append(callArgs, jen.Nil())
	}
	g.Return(jen.Id("runListPaginated").Call(
		jen.Id("cmd"),
		jen.Lit(resource),
		requestPathExpr(cmd),
		jen.Func().Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
		).Params(jen.Any(), jen.Op("*").Qual(wenmarPkg, "Paginator"), jen.Error()).Block(
			jen.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").Dot(sdkMethodNameFor(cmd)).Call(callArgs...),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Nil(), jen.Id("err")),
			),
			jen.Return(jen.Id("resp").Dot("JSON200"), jen.Id("client").Dot("PaginatorFromResponse").Call(jen.Id("resp").Dot("HTTPResponse")), jen.Nil()),
		),
	))
}

func hasFiltersFnName(cmd GenCommand) string {
	return toCamelCase(cmd.Resource) + "ListHasFilters"
}

// emitHasFiltersFunc emits a function that reports whether any filter flag
// was set, so the paginated-with-params handler can decide between the
// filtered and plain SDK calls.
func emitHasFiltersFunc(f *jen.File, cmd GenCommand) {
	f.Func().Id(hasFiltersFnName(cmd)).Params().Id("bool").BlockFunc(func(g *jen.Group) {
		for _, bf := range cmd.QueryFields {
			varName := bodyFieldVarName(cmd.Resource, bf.GoName)
			switch bf.Type {
			case "string":
				g.If(jen.Id(varName).Op("!=").Lit("")).Block(jen.Return(jen.True()))
			case "integer":
				g.If(jen.Id(varName).Op(">").Lit(0)).Block(jen.Return(jen.True()))
			case "boolean":
				g.If(jen.Id(varName)).Block(jen.Return(jen.True()))
			}
		}
		g.Return(jen.False())
	})
}

func hasFiltersVarName(cmd GenCommand) string {
	return toCamelCase(cmd.Resource) + "ListAll"
}

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
			bg.If(jen.Id(hasFiltersFnName(cmd)).Call()).BlockFunc(func(ibg *jen.Group) {
				ibg.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").
					Dot(sdkMethodNameFor(cmd)).Call(jen.Id("ctx"), jen.Op("&").Qual(wenmarPkg, cmd.QueryParamStruct).Values(queryParamDict(cmd)))
				ibg.If(jen.Id("err").Op("!=").Nil()).Block(
					jen.Return(jen.Nil(), jen.Nil(), jen.Id("err")),
				)
				ibg.Return(jen.Id("resp").Dot("JSON200"), jen.Id("client").Dot("PaginatorFromResponse").Call(jen.Id("resp").Dot("HTTPResponse")), jen.Nil())
			}).Else().BlockFunc(func(ebg *jen.Group) {
				ebg.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").
					Dot(sdkMethodNameFor(cmd)).Call(jen.Id("ctx"), jen.Nil())
				ebg.If(jen.Id("err").Op("!=").Nil()).Block(
					jen.Return(jen.Nil(), jen.Nil(), jen.Id("err")),
				)
				ebg.Return(jen.Id("resp").Dot("JSON200"), jen.Id("client").Dot("PaginatorFromResponse").Call(jen.Id("resp").Dot("HTTPResponse")), jen.Nil())
			})
		}),
	))
}

// bodyStructFields renders SDK-compatible anonymous struct fields. The
// SDK's inline structs list fields alphabetically; json tags match the
// spec's property names. The generated request wrappers for this surface
// (driver, service_category, work_order) use plain (non-pointer) field
// types in the SDK, so the anonymous struct uses the plain type regardless
// of spec optionality — IsPointer only affects the CLI flag, not the struct.
func bodyStructFields(cmd GenCommand) []jen.Code {
	fields := make([]jen.Code, 0, len(cmd.BodyFields))
	for _, bf := range cmd.BodyFields {
		fieldType := jen.Id(goType(bf.Type))
		jsonTag := bf.JSONName
		if bf.IsPointer {
			// Optional fields are pointers with omitempty in the generated
			// SDK structs — the anonymous literal must match exactly for
			// assignability (Go requires identical tags).
			fieldType = jen.Op("*").Id(goType(bf.Type))
			jsonTag = bf.JSONName + ",omitempty"
		}
		fields = append(fields,
			jen.Id(bf.GoName).Add(fieldType).Tag(map[string]string{"json": jsonTag}),
		)
	}
	return fields
}

// bodyLiteralDict builds the value dict for the body literal. Fields with
// NoFlag (arrays) are omitted so they stay zero-valued in the struct.
func bodyLiteralDict(cmd GenCommand) jen.Dict {
	dict := jen.Dict{}
	for _, bf := range cmd.BodyFields {
		if bf.NoFlag {
			continue
		}
		varName := bodyFieldVarName(cmd.Resource, bf.GoName)
		if bf.IsPointer {
			dict[jen.Id(bf.GoName)] = wrapPtr(bf.Type, jen.Id(varName))
		} else {
			dict[jen.Id(bf.GoName)] = jen.Id(varName)
		}
	}
	return dict
}

// requestBodyExpr renders the full request-struct literal:
//
//	flat:    wenmar.MergeCustomerRequest{SourceCustomerId: v}
//	wrapped: wenmar.CreateDriverRequest{Driver: struct{...}{FullName: v}}
func requestBodyExpr(cmd GenCommand) jen.Code {
	if cmd.WrapperKey == "" {
		return jen.Qual(wenmarPkg, cmd.RequestStruct).Values(bodyLiteralDict(cmd))
	}
	return jen.Qual(wenmarPkg, cmd.RequestStruct).Values(jen.Dict{
		jen.Id(snakeToPascal(cmd.WrapperKey)): jen.Struct(bodyStructFields(cmd)...).Values(bodyLiteralDict(cmd)),
	})
}

// responseFieldFor returns the response field to read, defaulting to
// JSON200 when unset.
func responseFieldFor(cmd GenCommand) string {
	if cmd.ResponseField == "" {
		return "JSON200"
	}
	return cmd.ResponseField
}

func emitCreateHandler(g *jen.Group, cmd GenCommand) {
	resource := cmd.Resource
	summary := titleCase(singularize(resource)) + " created."

	// Build the body builder closure.
	bodyBuilder := jen.Func().Params().Params(jen.Any(), jen.Error()).BlockFunc(func(bg *jen.Group) {
		if cmd.RequestStruct == "" || len(cmd.BodyFields) == 0 {
			bg.Return(jen.Nil(), jen.Nil())
			return
		}
		bg.Id("req").Op(":=").Add(requestBodyExpr(cmd))
		bg.Return(jen.Id("req"), jen.Nil())
	})

	// Build the sender closure.
	sdkMethod := sdkMethodNameFor(cmd)
	sender := jen.Func().Params(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
		jen.Id("body").Any(),
	).Params(jen.Any(), jen.Error()).BlockFunc(func(bg *jen.Group) {
		if cmd.RequestStruct == "" || len(cmd.BodyFields) == 0 {
			bg.Return(jen.Nil(), jen.Nil())
			return
		}
		callArgs := sdkCallArgs(cmd, false)
		callArgs = append(callArgs, jen.Id("body").Assert(jen.Qual(wenmarPkg, cmd.RequestStruct)))
		bg.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").Dot(sdkMethod).Call(callArgs...)
		bg.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Id("err")),
		)
		bg.Return(jen.Id("resp").Dot(responseFieldFor(cmd)), jen.Nil())
	})

	g.Return(jen.Id("runCreate").Call(
		jen.Id("cmd"),
		jen.Lit(resource),
		requestPathExpr(cmd),
		jen.Lit(summary),
		bodyBuilder,
		sender,
	))
}

func emitUpdateHandler(g *jen.Group, cmd GenCommand) {
	resource := cmd.Resource
	summary := titleCase(singularize(resource)) + " updated."

	// Build the body builder closure (takes id as param).
	bodyBuilder := jen.Func().Params(jen.Id("id").Id("int")).Params(jen.Any(), jen.Error()).BlockFunc(func(bg *jen.Group) {
		if cmd.RequestStruct == "" || len(cmd.BodyFields) == 0 {
			bg.Return(jen.Nil(), jen.Nil())
			return
		}
		bg.Id("req").Op(":=").Add(requestBodyExpr(cmd))
		bg.Return(jen.Id("req"), jen.Nil())
	})

	// Build the sender closure.
	sdkMethod := sdkMethodNameFor(cmd)
	sender := jen.Func().Params(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
		jen.Id("id").Id("int"),
		jen.Id("body").Any(),
	).Params(jen.Any(), jen.Error()).BlockFunc(func(bg *jen.Group) {
		if cmd.RequestStruct == "" || len(cmd.BodyFields) == 0 {
			bg.Return(jen.Nil(), jen.Nil())
			return
		}
		callArgs := sdkCallArgs(cmd, true)
		callArgs = append(callArgs, jen.Id("body").Assert(jen.Qual(wenmarPkg, cmd.RequestStruct)))
		bg.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").Dot(sdkMethod).Call(callArgs...)
		bg.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Id("err")),
		)
		bg.Return(jen.Id("resp").Dot(responseFieldFor(cmd)), jen.Nil())
	})

	g.Return(jen.Id("runUpdate").Call(
		jen.Id("cmd"), jen.Id("args"),
		jen.Lit(resource), pathFnForRunner(cmd), jen.Lit(summary),
		bodyBuilder,
		sender,
	))
}

func emitDeleteHandler(g *jen.Group, cmd GenCommand) {
	resource := cmd.Resource
	label := titleCase(singularize(resource))
	dryRunVar := flagVarName(cmd.Resource, "delete_dry_run")
	g.Return(jen.Id("runDelete").Call(
		jen.Id("cmd"), jen.Id("args"),
		jen.Lit(label), jen.Lit(resource),
		pathFnForRunner(cmd),
		jen.Id(dryRunVar),
		jen.Func().Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
			jen.Id("id").Id("int"),
		).Params(jen.Any(), jen.Error()).BlockFunc(func(bg *jen.Group) {
			callArgs := sdkCallArgs(cmd, true)
			bg.Return(jen.Id("client").Dot(sdkMethodNameFor(cmd)).Call(callArgs...))
		}),
	))
}

func emitActionCreateHandler(g *jen.Group, cmd GenCommand) {
	resource := cmd.Resource
	summary := titleCase(singularize(resource)) + " action completed."

	bodyBuilder := jen.Func().Params(jen.Id("id").Id("int")).Params(jen.Any(), jen.Error()).BlockFunc(func(bg *jen.Group) {
		if cmd.RequestStruct == "" || len(cmd.BodyFields) == 0 {
			bg.Return(jen.Nil(), jen.Nil())
			return
		}
		bg.Id("req").Op(":=").Add(requestBodyExpr(cmd))
		bg.Return(jen.Id("req"), jen.Nil())
	})

	sdkMethod := sdkMethodNameFor(cmd)
	sender := jen.Func().Params(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
		jen.Id("id").Id("int"),
		jen.Id("body").Any(),
	).Params(jen.Any(), jen.Error()).BlockFunc(func(bg *jen.Group) {
		callArgs := sdkCallArgs(cmd, true)
		callArgs = append(callArgs, jen.Id("body").Assert(jen.Qual(wenmarPkg, cmd.RequestStruct)))
		bg.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").Dot(sdkMethod).Call(callArgs...)
		bg.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Id("err")),
		)
		bg.Return(jen.Id("resp").Dot(responseFieldFor(cmd)), jen.Nil())
	})

	g.Return(jen.Id("runAction").Call(
		jen.Id("cmd"), jen.Id("args"),
		jen.Lit(resource), jen.Lit(strings.ToUpper(cmd.Method)),
		pathFnForRunner(cmd), jen.Lit(summary),
		bodyBuilder,
		sender,
	))
}

func emitActionUpdateHandler(g *jen.Group, cmd GenCommand) {
	// Same as actionCreate but with PATCH method.
	emitActionCreateHandler(g, cmd)
}

func emitTabHandler(g *jen.Group, cmd GenCommand) {
	resource := cmd.Resource
	sdkMethod := sdkMethodNameFor(cmd)
	tab := cmd.Tab
	g.Return(jen.Id("runShow").Call(
		jen.Id("cmd"), jen.Id("args"),
		jen.Lit(resource), jen.Lit("GET"),
		jen.Func().Params(jen.Id("a").Index().Id("string")).Id("string").Block(
			jen.Return(jen.Qual("fmt", "Sprintf").Call(jen.Lit("/"+resource+"/%s/"+tab), jen.Id("a").Index(jen.Lit(0)))),
		),
		jen.Func().Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
			jen.Id("id").Id("int"),
		).Params(jen.Any(), jen.Error()).Block(
			jen.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").Dot(sdkMethod).Call(jen.Id("ctx"), jen.Id("id")),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Id("err")),
			),
			jen.Return(jen.Id("resp").Dot("JSON200"), jen.Nil()),
		),
	))
}

// positionalQueryField returns the query field the positional arg maps to.
// The twins map the positional string to the search-term field (Query or
// Vin), never the optional id field, so skip any field named "Id".
func positionalQueryField(cmd GenCommand) BodyField {
	for _, bf := range cmd.QueryFields {
		if bf.GoName != "Id" {
			return bf
		}
	}
	if len(cmd.QueryFields) > 0 {
		return cmd.QueryFields[0]
	}
	return BodyField{}
}

func emitPositionalArgHandler(g *jen.Group, cmd GenCommand) {
	resource := cmd.Resource
	sdkMethod := sdkMethodNameFor(cmd)
	field := positionalQueryField(cmd)
	g.Return(jen.Id("runList").Call(
		jen.Id("cmd"),
		jen.Lit(resource),
		jen.Lit(cmd.Path),
		jen.Func().Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
		).Params(jen.Any(), jen.Error()).Block(
			jen.Id("query").Op(":=").Id("args").Index(jen.Lit(0)),
			jen.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").Dot(sdkMethod).Call(
				jen.Id("ctx"),
				jen.Op("&").Qual(wenmarPkg, cmd.QueryParamStruct).Values(jen.Dict{
					jen.Id(field.GoName): jen.Id("strPtr").Call(jen.Id("query")),
				}),
			),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Id("err")),
			),
			jen.Return(jen.Id("resp").Dot("JSON200"), jen.Nil()),
		),
	))
}

func emitQueryParamHandler(g *jen.Group, cmd GenCommand) {
	resource := cmd.Resource
	sdkMethod := sdkMethodNameFor(cmd)
	g.Return(jen.Id("runList").Call(
		jen.Id("cmd"),
		jen.Lit(resource),
		jen.Lit(cmd.Path),
		jen.Func().Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
		).Params(jen.Any(), jen.Error()).Block(
			jen.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").Dot(sdkMethod).Call(
				jen.Id("ctx"),
				jen.Op("&").Qual(wenmarPkg, cmd.QueryParamStruct).Values(queryParamDict(cmd)),
			),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Id("err")),
			),
			jen.Return(jen.Id("resp").Dot("JSON200"), jen.Nil()),
		),
	))
}

// queryParamDict builds the jen.Dict for a query param struct literal.
func queryParamDict(cmd GenCommand) jen.Dict {
	dict := jen.Dict{}
	for _, bf := range cmd.QueryFields {
		if bf.NoFlag {
			continue
		}
		varName := bodyFieldVarName(cmd.Resource, bf.GoName)
		if bf.FlagType == "intslice" {
			// Convert the []int flag var to the SDK's *[]string param.
			dict[jen.Id(bf.GoName)] = jen.Id("intSliceToStrPtr").Call(jen.Id(varName))
			continue
		}
		if bf.IsPointer {
			dict[jen.Id(bf.GoName)] = wrapPtr(bf.Type, jen.Id(varName))
		} else {
			dict[jen.Id(bf.GoName)] = jen.Id(varName)
		}
	}
	return dict
}

func emitActionHandler(g *jen.Group, cmd GenCommand) {
	// Actions without body (merge, transfer) — treat as list for now.
	g.Comment("TODO: implement action handler for " + cmd.OperationID)
	g.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("action %s not yet generated"), jen.Lit(cmd.OperationID)))
}

// actionSummary returns the past-tense success message for an action command.
// It prefers the ActionSummary override and falls back to a derived message.
func actionSummary(cmd GenCommand) string {
	if cmd.ActionSummary != "" {
		return cmd.ActionSummary
	}
	return titleCase(singularize(cmd.Resource)) + " " + cmd.Command + "."
}

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
		jen.Lit(actionSummary(cmd)),
		jen.Func().Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
			jen.Id("id").Id("int"),
		).Params(jen.Any(), jen.Error()).BlockFunc(func(bg *jen.Group) {
			callArgs := []jen.Code{jen.Id("ctx"), jen.Id("id")}
			if cmd.RequestStruct != "" {
				callArgs = append(callArgs, jen.Qual(wenmarPkg, cmd.RequestStruct).Values())
			}
			bg.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").Dot(sdkMethodNameFor(cmd)).Call(callArgs...)
			bg.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Id("err")),
			)
			bg.Return(jen.Id("resp").Dot(responseFieldFor(cmd)), jen.Nil())
		}),
	))
}

// emitSeedActionHandler emits POST-collection actions with empty bodies.
func emitSeedActionHandler(g *jen.Group, cmd GenCommand) {
	g.Return(jen.Id("runSeedAction").Call(
		jen.Id("cmd"),
		jen.Lit(cmd.Resource),
		jen.Lit(cmd.Path),
		jen.Lit(actionSummary(cmd)),
		jen.Func().Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("client").Op("*").Qual(wenmarPkg, "Client"),
		).Params(jen.Any(), jen.Error()).BlockFunc(func(bg *jen.Group) {
			callArgs := []jen.Code{jen.Id("ctx")}
			if cmd.RequestStruct != "" {
				callArgs = append(callArgs, jen.Qual(wenmarPkg, cmd.RequestStruct).Values())
			}
			bg.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").Dot(sdkMethodNameFor(cmd)).Call(callArgs...)
			bg.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Id("err")),
			)
			bg.Return(jen.Id("resp").Dot(responseFieldFor(cmd)), jen.Nil())
		}),
	))
}

const wenmarPkg = "github.com/wenmar-pro/wenmar-sdk/go/wenmar"

func sdkMethodNameFor(cmd GenCommand) string {
	if cmd.SDKMethod != "" {
		return cmd.SDKMethod
	}
	return sdkMethodName(cmd.OperationID)
}

func sdkMethodName(operationID string) string {
	parts := strings.Split(operationID, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

// goType maps an OpenAPI type to a Go type.
func goType(openapiType string) string {
	switch openapiType {
	case "integer":
		return "int"
	case "string":
		return "string"
	case "boolean":
		return "bool"
	case "number":
		return "float64"
	case "array":
		return "[]interface{}"
	default:
		return "string"
	}
}

// flagBindMethod returns the cobra flag binding method for a type.
func flagBindMethod(openapiType string) string {
	switch openapiType {
	case "integer":
		return "IntVar"
	case "string":
		return "StringVar"
	case "boolean":
		return "BoolVar"
	default:
		return "StringVar"
	}
}

// kebabCase converts snake_case to kebab-case.
func kebabCase(s string) string {
	return strings.ReplaceAll(s, "_", "-")
}

// DefaultForType returns the zero-value default for a flag type.
func DefaultForType(t string) jen.Code {
	switch t {
	case "integer":
		return jen.Lit(0)
	case "string":
		return jen.Lit("")
	case "boolean":
		return jen.False()
	default:
		return jen.Lit("")
	}
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	parts := strings.Split(s, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func toCamelCase(s string) string {
	if s == "" {
		return s
	}
	parts := strings.Split(s, "_")
	if len(parts) == 1 {
		parts = strings.Split(s, "-")
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if parts[i] == "" {
			continue
		}
		result += strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return result
}

func singularize(s string) string {
	// Special cases for compound words
	if strings.HasSuffix(s, "work_orders") || s == "workorders" {
		return "work order"
	}
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}
