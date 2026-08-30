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
