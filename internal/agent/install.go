package agent

import (
	"fmt"
	"os"
	"path/filepath"
)

// managedMarker is written into a skill directory to indicate wenmar-cli
// manages it. wenmar-cli only writes to directories with this marker.
const managedMarker = ".managed-by-wenmar-cli"

// SkillDir returns the target skill directory under ~/.agents/skills/wenmar.
func SkillDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".agents", "skills", "wenmar"), nil
}

// SkillInstalled reports whether the skill is installed and whether it is
// managed by wenmar-cli.
func SkillInstalled() (path string, managed bool, err error) {
	dir, err := SkillDir()
	if err != nil {
		return "", false, err
	}
	skillPath := filepath.Join(dir, "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		return "", false, nil
	}
	_, markerErr := os.Stat(filepath.Join(dir, managedMarker))
	return skillPath, markerErr == nil, nil
}

// InstallSkill copies the bundled SKILL.md to the target directory. It refuses
// to overwrite a hand-authored skill (a directory without the managed marker)
// unless force is true.
func InstallSkill(sourceDir, targetDir string, force bool) error {
	source := filepath.Join(sourceDir, "SKILL.md")
	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read bundled skill: %w", err)
	}

	// Refuse to overwrite a hand-authored skill without --force.
	if !force {
		if _, err := os.Stat(filepath.Join(targetDir, managedMarker)); err != nil {
			if _, statErr := os.Stat(filepath.Join(targetDir, "SKILL.md")); statErr == nil {
				return fmt.Errorf("skill directory exists without the %s marker; use --force to overwrite", managedMarker)
			}
		}
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create skill dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "SKILL.md"), data, 0644); err != nil {
		return fmt.Errorf("write skill: %w", err)
	}
	// Write the managed marker.
	if err := os.WriteFile(filepath.Join(targetDir, managedMarker), []byte("wenmar-cli\n"), 0644); err != nil {
		return fmt.Errorf("write managed marker: %w", err)
	}
	return nil
}

// UninstallSkill removes the skill directory only if it has the managed marker.
func UninstallSkill(targetDir string) error {
	if _, err := os.Stat(filepath.Join(targetDir, managedMarker)); err != nil {
		return fmt.Errorf("refusing to remove %s: not managed by wenmar-cli", targetDir)
	}
	return os.RemoveAll(targetDir)
}
