package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wenmar-pro/wenmar-cli/internal/agent"
)

func TestSkillInstallCopiesRepoSkillVerbatim(t *testing.T) {
	// The repo's skills/wenmar/SKILL.md is the source of truth. Tests run
	// with the working directory set to cmd/, so the repo skill is one
	// level up. (bundledSkillDir() can't be used here: os.Executable()
	// points at the test binary in /tmp, not the repo layout.)
	source := filepath.Join("..", "skills", "wenmar")
	want, err := os.ReadFile(filepath.Join(source, "SKILL.md"))
	if err != nil {
		t.Fatalf("read repo skill: %v", err)
	}

	target := filepath.Join(t.TempDir(), "skill")
	if err := agent.InstallSkill(source, target, true); err != nil {
		t.Fatalf("install: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(target, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Error("installed SKILL.md differs from the repo copy — shipping path corrupted")
	}
}
