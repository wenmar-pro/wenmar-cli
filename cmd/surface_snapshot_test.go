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
