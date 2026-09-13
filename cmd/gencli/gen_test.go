package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestEmitCreate_WrapperBody(t *testing.T) {
	cmd := GenCommand{
		OperationID:     "create_driver",
		Resource:        "drivers",
		Command:         "create",
		Method:          "post",
		Path:            "/customers/{customer_id}/drivers",
		RequestStruct:   "CreateDriverRequest",
		SDKMethod:       "CreateDriver",
		WrapperKey:      "driver",
		ResponseField:   "JSON201",
		RequestBody:     &RequestBody{Content: map[string]Media{"application/json": {Schema: Schema{Type: "object"}}}},
		ExtraPathParams: []Parameter{{Name: "customer_id", In: "path", Schema: Schema{Type: "integer"}}},
		BodyFields: []BodyField{
			{JSONName: "full_name", GoName: "FullName", FlagName: "full-name", Type: "string", Required: true, HelpText: "Full name (required)"},
			{JSONName: "phone", GoName: "Phone", FlagName: "phone", Type: "string", Required: true, HelpText: "Phone (required)"},
		},
	}
	group := CommandGroup{Resource: "drivers", Commands: []GenCommand{cmd}}
	code, err := emitGroup(group, nil, &Overrides{}, "")
	if err != nil {
		t.Fatalf("emitGroup: %v", err)
	}
	for _, want := range []string{
		"Driver: struct {", // wrapper key emitted
		"FullName: driversFullName",
		"client.CreateDriver(ctx, driversCustomerId, body.(wenmar.CreateDriverRequest))",
		"resp.JSON201",
	} {
		if !strings.Contains(code, want) {
			t.Errorf("emitted code missing %q:\n%s", want, code)
		}
	}
}

func TestEmitActionNoBody_EmptyStructArg(t *testing.T) {
	cmd := GenCommand{
		OperationID:   "deactivate_service_category",
		Resource:      "servicecategories",
		Command:       "deactivate",
		Method:        "patch",
		Path:          "/service_categories/{id}/deactivate",
		HasIDParam:    true,
		IDParam:       "id",
		IDType:        "int",
		SDKMethod:     "DeactivateServiceCategory",
		RequestStruct: "DeactivateServiceCategoryRequest",
		ActionSummary: "Service category deactivated.",
		RequestBody:   &RequestBody{Content: map[string]Media{"application/json": {Schema: Schema{Type: "object"}}}},
	}
	group := CommandGroup{Resource: "servicecategories", Commands: []GenCommand{cmd}}
	code, err := emitGroup(group, nil, &Overrides{}, "")
	if err != nil {
		t.Fatalf("emitGroup: %v", err)
	}
	if !strings.Contains(code, "DeactivateServiceCategoryRequest{}") {
		t.Errorf("action call missing empty-struct body arg:\n%s", code)
	}
	if !strings.Contains(code, "runActionNoBody") {
		t.Errorf("expected runActionNoBody runner:\n%s", code)
	}
}

func TestEmitNestedList_PositionalId(t *testing.T) {
	cmd := GenCommand{
		OperationID: "list_customers_vehicles",
		Resource:    "customers",
		Command:     "vehicles",
		Method:      "get",
		Path:        "/customers/{customer_id}/vehicles",
		IDParam:     "customer_id",
		HasIDParam:  true,
		IDType:      "int",
		SDKMethod:   "ListCustomersVehicles",
		PathParams:  []Parameter{{Name: "customer_id", In: "path", Schema: Schema{Type: "integer"}}},
	}
	group := CommandGroup{Resource: "customers", Commands: []GenCommand{cmd}}
	code, err := emitGroup(group, nil, &Overrides{}, "")
	if err != nil {
		t.Fatalf("emitGroup: %v", err)
	}
	for _, want := range []string{
		`Use:   "vehicles <id>"`,
		"runShow(cmd, args, \"customers\", \"GET\"",
		"ListCustomersVehicles(ctx, id)",
		`fmt.Sprintf("/customers/%s/vehicles", a[0])`,
	} {
		if !strings.Contains(code, want) {
			t.Errorf("emitted code missing %q:\n%s", want, code)
		}
	}
}

func TestResolveSchemaRef(t *testing.T) {
	spec := &Spec{
		Paths: map[string]PathItem{},
		Components: Components{
			Schemas: map[string]Schema{
				"CreateDriverRequest": {
					Type: "object",
					Properties: map[string]Schema{
						"driver": {Type: "object", Properties: map[string]Schema{
							"full_name": {Type: "string"},
							"phone":     {Type: "string"},
						}},
					},
				},
			},
		},
	}
	body := &RequestBody{Content: map[string]Media{
		"application/json": {Schema: Schema{Ref: "#/components/schemas/CreateDriverRequest"}},
	}}

	resolved := spec.Resolve(body.Content["application/json"].Schema)
	if resolved.Type != "object" || resolved.Properties["driver"].Type != "object" {
		t.Fatalf("ref not resolved: %+v", resolved)
	}
	// Unresolvable refs return the schema untouched (defensive).
	untouched := spec.Resolve(Schema{Ref: "#/components/schemas/NoSuch"})
	if untouched.Ref != "#/components/schemas/NoSuch" {
		t.Fatalf("unresolved ref should pass through, got %+v", untouched)
	}
}

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

func TestGroupOverridesPlumbThrough(t *testing.T) {
	overrides := &Overrides{
		Groups: map[string]GroupOverride{
			"wo": {Aliases: []string{"workorders", "work_orders"}, Short: "Manage work orders"},
		},
		Commands: map[string]CommandOverride{},
	}
	group := CommandGroup{Resource: "wo", Commands: []GenCommand{
		{OperationID: "list_work_orders", Resource: "wo", Command: "list", Method: "get", IsPaginated: true, SDKMethod: "ListWorkOrders"},
	}}
	code, err := emitGroup(group, nil, overrides, "")
	if err != nil {
		t.Fatalf("emitGroup: %v", err)
	}
	if !strings.Contains(code, `Aliases: []string{"workorders", "work_orders"}`) {
		t.Errorf("parent aliases not emitted:\n%s", code)
	}
	if !strings.Contains(code, `Short: "Manage work orders"`) {
		t.Errorf("parent short not emitted:\n%s", code)
	}
	if !strings.Contains(code, "cobra.NoArgs") {
		t.Errorf("parent Args validation not emitted:\n%s", code)
	}
}

func TestEmitGroup_ServiceCategoryActionsCompile(t *testing.T) {
	// Deactivate is PATCH /service_categories/{id}/deactivate with an
	// empty-object body: the case that ships broken commands today.
	cmd := GenCommand{
		OperationID:   "deactivate_service_category",
		Resource:      "servicecategories",
		Command:       "deactivate",
		Method:        "patch",
		Path:          "/service_categories/{id}/deactivate",
		HasIDParam:    true,
		IDParam:       "id",
		Summary:       "Deactivate a service category by ID",
		SDKMethod:     "DeactivateServiceCategory",
		RequestStruct: "DeactivateServiceCategoryRequest",
		RequestBody: &RequestBody{Content: map[string]Media{
			"application/json": {Schema: Schema{Type: "object", Properties: map[string]Schema{}}},
		}},
	}
	group := CommandGroup{Resource: "servicecategories", Commands: []GenCommand{cmd}}
	code, err := emitGroup(group, nil, &Overrides{}, "")
	if err != nil {
		t.Fatalf("emitGroup: %v", err)
	}
	for _, want := range []string{
		"runServicecategoriesDeactivate",
		"client.DeactivateServiceCategory(ctx, id, wenmar.DeactivateServiceCategoryRequest{})",
		"cobra.ExactArgs(1)",
	} {
		if !strings.Contains(code, want) {
			t.Errorf("emitted code missing %q:\n%s", want, code)
		}
	}
	if strings.Contains(code, "not yet generated") {
		t.Error("action stub still emitted")
	}
}

func TestEmitGroup_CustomersListWithFiltersPaginated(t *testing.T) {
	cmd := GenCommand{
		OperationID:      "list_customers",
		Resource:         "customers",
		Command:          "list",
		Method:           "get",
		Path:             "/customers",
		IsPaginated:      true,
		QueryParamStruct: "ListCustomersParams",
		SDKMethod:        "ListCustomers",
		QueryFields: []BodyField{
			{JSONName: "query", GoName: "Query", FlagName: "query", Type: "string", HelpText: "Full-text search"},
			{JSONName: "page", GoName: "Page", FlagName: "page", Type: "integer", HelpText: "Page number"},
		},
	}
	group := CommandGroup{Resource: "customers", Commands: []GenCommand{cmd}}
	code, err := emitGroup(group, nil, &Overrides{}, "")
	if err != nil {
		t.Fatalf("emitGroup: %v", err)
	}
	for _, want := range []string{
		"runListPaginatedWithAll",
		"ListCustomersRaw(ctx, &wenmar.ListCustomersParams",
		"PaginatorFromResponse",
		"customersQuery",
		"customersPage",
		"\"query\"",
		"\"page\"",
	} {
		if !strings.Contains(code, want) {
			t.Errorf("emitted code missing %q:\n%s", want, code)
		}
	}
}

func TestExtractQueryFields_BracketParamNames(t *testing.T) {
	op := Operation{
		Parameters: []Parameter{
			{Name: "filter[status]", In: "query", Required: false, Schema: Schema{Type: "string"}},
			{Name: "filters[q]", In: "query", Required: false, Schema: Schema{Type: "string"}},
			{Name: "filters[has_open_work_order]", In: "query", Required: false, Schema: Schema{Type: "boolean"}},
		},
	}
	fields := extractQueryFields(op, "ListReportsStatementsParams", nil)
	for _, f := range fields {
		if strings.ContainsAny(f.GoName, "[]") {
			t.Errorf("GoName %q contains brackets — invalid Go identifier", f.GoName)
		}
		if strings.ContainsAny(f.FlagName, "[]") {
			t.Errorf("FlagName %q contains brackets", f.FlagName)
		}
	}
	got := map[string]BodyField{}
	for _, f := range fields {
		got[f.JSONName] = f
	}
	// JSONName must keep the raw spec name so the SDK struct binding works.
	if got["filter[status]"].GoName != "FilterStatus" {
		t.Errorf("filter[status] GoName = %q, want %q", got["filter[status]"].GoName, "FilterStatus")
	}
	if got["filter[status]"].FlagName != "status" {
		t.Errorf("filter[status] FlagName = %q, want %q", got["filter[status]"].FlagName, "status")
	}
	if got["filters[q]"].GoName != "FiltersQ" {
		t.Errorf("filters[q] GoName = %q, want %q", got["filters[q]"].GoName, "FiltersQ")
	}
	if got["filters[q]"].FlagName != "q" {
		t.Errorf("filters[q] FlagName = %q, want %q", got["filters[q]"].FlagName, "q")
	}
	if got["filters[has_open_work_order]"].GoName != "FiltersHasOpenWorkOrder" {
		t.Errorf("filters[has_open_work_order] GoName = %q, want %q", got["filters[has_open_work_order]"].GoName, "FiltersHasOpenWorkOrder")
	}
	if got["filters[has_open_work_order]"].FlagName != "has-open-work-order" {
		t.Errorf("filters[has_open_work_order] FlagName = %q, want %q", got["filters[has_open_work_order]"].FlagName, "has-open-work-order")
	}
}

func TestEmitGroup_BracketQueryParamsEmitValidIdentifiers(t *testing.T) {
	cmd := GenCommand{
		OperationID:      "list_reports_statements",
		Resource:         "reports",
		Command:          "list",
		Method:           "get",
		Path:             "/reports/statements",
		IsPaginated:      true,
		QueryParamStruct: "ListReportsStatementsParams",
		SDKMethod:        "ListReportsStatements",
		QueryFields: []BodyField{
			{JSONName: "filter[status]", GoName: "FilterStatus", FlagName: "status", Type: "string", HelpText: "Filter status"},
		},
	}
	group := CommandGroup{Resource: "reports", Commands: []GenCommand{cmd}}
	code, err := emitGroup(group, nil, &Overrides{}, "")
	if err != nil {
		t.Fatalf("emitGroup: %v", err)
	}
	for _, want := range []string{
		"reportsFilterStatus",
		"FilterStatus:",
		`"status"`,
	} {
		if !strings.Contains(code, want) {
			t.Errorf("emitted code missing %q:\n%s", want, code)
		}
	}
	for _, bad := range []string{"Filter[status]", "reportsFilter[status]"} {
		if strings.Contains(code, bad) {
			t.Errorf("emitted code contains invalid identifier %q:\n%s", bad, code)
		}
	}
}

func TestBuildCommand_ResponseField202(t *testing.T) {
	spec := &Spec{}
	op := Operation{
		OperationID: "create_inventory_level_extraction",
		Responses:   map[string]Response{"202": {}, "403": {}, "422": {}},
		RequestBody: &RequestBody{Content: map[string]Media{
			"application/json": {Schema: Schema{Type: "object"}},
		}},
		XWenmarRequestSchema: "CreateInventoryLevelExtractionRequest",
	}
	cmd := buildCommand(spec, op, "post", "/inventory_levels/extractions", &Overrides{Commands: map[string]CommandOverride{}})
	if cmd == nil {
		t.Fatal("buildCommand returned nil")
	}
	if cmd.ResponseField != "JSON202" {
		t.Errorf("ResponseField = %q, want %q", cmd.ResponseField, "JSON202")
	}
}

func TestBuildCommand_ResponseField201WinsOver202(t *testing.T) {
	spec := &Spec{}
	op := Operation{
		OperationID: "create_thing",
		Responses:   map[string]Response{"201": {}, "202": {}},
		RequestBody: &RequestBody{Content: map[string]Media{
			"application/json": {Schema: Schema{Type: "object"}},
		}},
	}
	cmd := buildCommand(spec, op, "post", "/things", &Overrides{Commands: map[string]CommandOverride{}})
	if cmd == nil {
		t.Fatal("buildCommand returned nil")
	}
	if cmd.ResponseField != "JSON201" {
		t.Errorf("ResponseField = %q, want %q (201 should win over 202)", cmd.ResponseField, "JSON201")
	}
}

func TestEmitGroup_UnclassifiableActionFailsAtGeneration(t *testing.T) {
	cmd := GenCommand{
		OperationID: "some_unclassifiable_action",
		Resource:    "widgets",
		Command:     "act",
		Method:      "post",
		Path:        "/widgets/{id}/bogus",
		HasIDParam:  true,
	}
	group := CommandGroup{Resource: "widgets", Commands: []GenCommand{cmd}}
	_, err := emitGroup(group, nil, &Overrides{}, "")
	if err == nil {
		t.Fatal("expected emitGroup to fail on an unclassifiable action, got nil")
	}
	if !strings.Contains(err.Error(), "some_unclassifiable_action") {
		t.Errorf("error should name the operationID, got: %v", err)
	}
	if !strings.Contains(err.Error(), "commands:") {
		t.Errorf("error should mention a commands: override, got: %v", err)
	}
}

func TestGroupOperations_DuplicateVarNameWarns(t *testing.T) {
	spec := &Spec{
		Paths: map[string]PathItem{
			"/things/a": {
				"get": {
					OperationID: "list_a_things",
					Summary:     "list a",
					Responses:   map[string]Response{},
					Parameters:  []Parameter{},
				},
			},
			"/things/b": {
				"get": {
					OperationID: "list_b_things",
					Summary:     "list b",
					Responses:   map[string]Response{},
					Parameters:  []Parameter{},
				},
			},
		},
	}
	var buf bytes.Buffer
	old := warnStderr
	warnStderr = &buf
	defer func() { warnStderr = old }()

	_ = groupOperations(spec, &Overrides{Commands: map[string]CommandOverride{}})
	msg := buf.String()
	if !strings.Contains(msg, "list_a_things") {
		t.Errorf("warning should name the first operationID, got: %q", msg)
	}
	if !strings.Contains(msg, "list_b_things") {
		t.Errorf("warning should name the dropped operationID, got: %q", msg)
	}
	if !strings.Contains(msg, "thingsListCmd") {
		t.Errorf("warning should name the colliding var name, got: %q", msg)
	}
}

func TestEmitCreateHandler_ActionSummaryOverride(t *testing.T) {
	cmd := GenCommand{
		OperationID:   "create_reports_tax_period",
		Resource:      "reports",
		Command:       "create",
		Method:        "post",
		Path:          "/reports/tax_periods",
		RequestStruct: "CreateReportsTaxPeriodRequest",
		SDKMethod:     "CreateReportsTaxPeriod",
		ResponseField: "JSON201",
		ActionSummary: "Tax period created.",
		WrapperKey:    "tax_period",
		RequestBody:   &RequestBody{Content: map[string]Media{"application/json": {Schema: Schema{Type: "object"}}}},
		BodyFields: []BodyField{
			{JSONName: "period_start", GoName: "PeriodStart", FlagName: "period-start", Type: "string", Required: true, HelpText: "Period Start (required)"},
		},
	}
	group := CommandGroup{Resource: "reports", Commands: []GenCommand{cmd}}
	code, err := emitGroup(group, nil, &Overrides{}, "")
	if err != nil {
		t.Fatalf("emitGroup: %v", err)
	}
	if !strings.Contains(code, `"Tax period created."`) {
		t.Errorf("action_summary override not used for create handler:\n%s", code)
	}
	if strings.Contains(code, `"Report created."`) {
		t.Errorf("derived create summary emitted despite action_summary override:\n%s", code)
	}
}
