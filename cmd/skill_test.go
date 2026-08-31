package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBundledSkillDirFindsSharePath(t *testing.T) {
	// Simulate an installed layout: binary in a temp "bin", skill in a
	// sibling "share/wenmar/skills/wenmar".
	root := t.TempDir()
	binDir := filepath.Join(root, "usr", "bin")
	shareSkill := filepath.Join(root, "usr", "share", "wenmar", "skills", "wenmar")
	if err := os.MkdirAll(shareSkill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shareSkill, "SKILL.md"), []byte("# x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := bundledSkillDirFrom(binDir)
	if err != nil {
		t.Fatalf("bundledSkillDirFrom: %v", err)
	}
	if got != shareSkill {
		t.Errorf("bundledSkillDirFrom = %q, want %q", got, shareSkill)
	}
}
