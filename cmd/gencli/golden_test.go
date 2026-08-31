package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoldenRegen regenerates every resource file and diffs against the
// committed fixtures in cmd/golden/. Failures mean generator/spec/override
// drift — regenerate fixtures with: make golden-update
func TestGoldenRegen(t *testing.T) {
	specPath := filepath.Join("..", "..", "..", "wenmar-sdk", "spec", "openapi.enriched.yaml")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Skip("wenmar-sdk spec not available (sibling checkout required)")
	}
	out := t.TempDir()
	if err := runGenerate(specPath, filepath.Join("..", "gen_overrides.yaml"), out, "ignore"); err != nil {
		t.Fatalf("generate: %v", err)
	}
	goldenDir := filepath.Join("..", "golden")
	entries, err := os.ReadDir(goldenDir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no golden fixtures — run: make golden-update (err: %v)", err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		want, err := os.ReadFile(filepath.Join(goldenDir, e.Name()))
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		got, err := os.ReadFile(filepath.Join(out, e.Name()))
		if os.IsNotExist(err) {
			t.Errorf("%s: generator no longer produces this file — if intentional, delete the fixture and regenerate surface-snapshot.json", e.Name())
			continue
		}
		if string(want) != string(got) {
			t.Errorf("%s drifted — run: make golden-update", e.Name())
		}
	}
	// Also fail if the generator produced files with no fixture.
	produced, _ := os.ReadDir(out)
	for _, e := range produced {
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if _, err := os.Stat(filepath.Join(goldenDir, e.Name())); os.IsNotExist(err) {
			t.Errorf("%s generated but no fixture — run: make golden-update", e.Name())
		}
	}
}
