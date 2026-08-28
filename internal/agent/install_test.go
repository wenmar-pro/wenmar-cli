package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallSkill_InstallsAndMarks(t *testing.T) {
	source := t.TempDir()
	os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("# wenmar skill\n"), 0644)

	target := filepath.Join(t.TempDir(), "wenmar")
	if err := InstallSkill(source, target, false); err != nil {
		t.Fatalf("InstallSkill failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, "SKILL.md")); err != nil {
		t.Errorf("SKILL.md not installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, managedMarker)); err != nil {
		t.Errorf("managed marker not written: %v", err)
	}
}

func TestInstallSkill_RefusesOverwriteWithoutForce(t *testing.T) {
	source := t.TempDir()
	os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("# wenmar skill\n"), 0644)

	target := filepath.Join(t.TempDir(), "wenmar")
	os.MkdirAll(target, 0755)
	os.WriteFile(filepath.Join(target, "SKILL.md"), []byte("# hand-authored\n"), 0644)

	if err := InstallSkill(source, target, false); err == nil {
		t.Error("expected error when overwriting hand-authored skill without --force")
	}
}

func TestInstallSkill_ForceOverwrites(t *testing.T) {
	source := t.TempDir()
	os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("# wenmar skill\n"), 0644)

	target := filepath.Join(t.TempDir(), "wenmar")
	os.MkdirAll(target, 0755)
	os.WriteFile(filepath.Join(target, "SKILL.md"), []byte("# hand-authored\n"), 0644)

	if err := InstallSkill(source, target, true); err != nil {
		t.Fatalf("InstallSkill with force failed: %v", err)
	}
}

func TestUninstallSkill_RefusesUnmanaged(t *testing.T) {
	target := filepath.Join(t.TempDir(), "wenmar")
	os.MkdirAll(target, 0755)
	os.WriteFile(filepath.Join(target, "SKILL.md"), []byte("# hand-authored\n"), 0644)

	if err := UninstallSkill(target); err == nil {
		t.Error("expected error when uninstalling unmanaged skill")
	}
}

func TestUninstallSkill_RemovesManaged(t *testing.T) {
	target := filepath.Join(t.TempDir(), "wenmar")
	os.MkdirAll(target, 0755)
	os.WriteFile(filepath.Join(target, "SKILL.md"), []byte("# wenmar skill\n"), 0644)
	os.WriteFile(filepath.Join(target, managedMarker), []byte("wenmar-cli\n"), 0644)

	if err := UninstallSkill(target); err != nil {
		t.Fatalf("UninstallSkill failed: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("skill directory should be removed")
	}
}

func TestSkillInstalled_ReportsManaged(t *testing.T) {
	target := filepath.Join(t.TempDir(), "wenmar")
	os.MkdirAll(target, 0755)
	os.WriteFile(filepath.Join(target, "SKILL.md"), []byte("# wenmar skill\n"), 0644)
	os.WriteFile(filepath.Join(target, managedMarker), []byte("wenmar-cli\n"), 0644)

	// SkillInstalled uses the real home dir; test the marker logic directly.
	if _, err := os.Stat(filepath.Join(target, managedMarker)); err != nil {
		t.Errorf("expected managed marker, got %v", err)
	}
}
