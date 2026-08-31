package main

import (
	"strings"
	"testing"
)

func TestEmitCreate_WrapperBody(t *testing.T) {
	cmd := GenCommand{
		OperationID:   "create_driver",
		Resource:      "drivers",
		Command:       "create",
		Method:        "post",
		Path:           "/customers/{customer_id}/drivers",
		RequestStruct: "CreateDriverRequest",
		SDKMethod:     "CreateDriver",
		WrapperKey:    "driver",
		ResponseField: "JSON201",
		RequestBody:   &RequestBody{Content: map[string]Media{"application/json": {Schema: Schema{Type: "object"}}}},
		ExtraPathParams: []Parameter{{Name: "customer_id", In: "path", Schema: Schema{Type: "integer"}}},
		BodyFields: []BodyField{
			{JSONName: "full_name", GoName: "FullName", FlagName: "full-name", Type: "string", Required: true, HelpText: "Full name (required)"},
			{JSONName: "phone", GoName: "Phone", FlagName: "phone", Type: "string", Required: true, HelpText: "Phone (required)"},
		},
	}
	group := CommandGroup{Resource: "drivers", Commands: []GenCommand{cmd}}
	code, err := emitGroup(group, nil, &Overrides{})
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
		OperationID:    "deactivate_service_category",
		Resource:      "servicecategories",
		Command:       "deactivate",
		Method:         "patch",
		Path:           "/service_categories/{id}/deactivate",
		HasIDParam:     true,
		IDParam:        "id",
		IDType:         "int",
		SDKMethod:      "DeactivateServiceCategory",
		RequestStruct:  "DeactivateServiceCategoryRequest",
		ActionSummary:  "Service category deactivated.",
		RequestBody:    &RequestBody{Content: map[string]Media{"application/json": {Schema: Schema{Type: "object"}}}},
	}
	group := CommandGroup{Resource: "servicecategories", Commands: []GenCommand{cmd}}
	code, err := emitGroup(group, nil, &Overrides{})
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
		Path:         "/customers/{customer_id}/vehicles",
		IDParam:      "customer_id",
		HasIDParam:   true,
		IDType:       "int",
		SDKMethod:    "ListCustomersVehicles",
		PathParams:   []Parameter{{Name: "customer_id", In: "path", Schema: Schema{Type: "integer"}}},
	}
	group := CommandGroup{Resource: "customers", Commands: []GenCommand{cmd}}
	code, err := emitGroup(group, nil, &Overrides{})
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
	if !strings.Contains(code, "cobra.NoArgs") {
		t.Errorf("parent Args validation not emitted:\n%s", code)
	}
}

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
		IDParam:     "id",
		Summary:     "Deactivate a service category by ID",
		SDKMethod:   "DeactivateServiceCategory",
		RequestStruct: "DeactivateServiceCategoryRequest",
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
	code, err := emitGroup(group, nil, &Overrides{})
	if err != nil {
		t.Fatalf("emitGroup: %v", err)
	}
	for _, want := range []string{
		"runListPaginatedWithAll",
		"ListCustomers(ctx, &wenmar.ListCustomersParams",
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
