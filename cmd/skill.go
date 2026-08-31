package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/agent"
)

var (
	skillForce bool
)

var skillCmd = &cobra.Command{
	Use:     "skill",
	Short:   "Manage the wenmar agent skill",
	GroupID: "agents",
}

var skillInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the wenmar skill for AI agents",
	RunE:  runSkillInstall,
}

var skillUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the wenmar skill (only if managed by wenmar-cli)",
	RunE:  runSkillUninstall,
}

var skillUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Re-copy the wenmar skill (only if managed by wenmar-cli)",
	RunE:  runSkillUpdate,
}

func init() {
	skillInstallCmd.Flags().BoolVar(&skillForce, "force", false, "Overwrite a hand-authored skill")
	skillCmd.AddCommand(skillInstallCmd, skillUninstallCmd, skillUpdateCmd)
	rootCmd.AddCommand(skillCmd)
}

// bundledSkillDirFrom returns the skill directory for the given binary
// directory. Candidate locations cover the repo layout, the goreleaser
// archive layout (binary next to skills/), and package installs
// (/usr/bin + /usr/share/wenmar/skills).
func bundledSkillDirFrom(binDir string) (string, error) {
	candidates := []string{
		filepath.Join(binDir, "skills", "wenmar"),
		filepath.Join(binDir, "..", "skills", "wenmar"),
		filepath.Join(binDir, "..", "share", "wenmar", "skills", "wenmar"),
		filepath.Join(binDir, "..", "..", "share", "wenmar", "skills", "wenmar"),
		"/usr/share/wenmar/skills/wenmar",
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "SKILL.md")); err == nil {
			return filepath.Clean(c), nil
		}
	}
	return "", fmt.Errorf("bundled skill not found near the wenmar binary (looked in %v)", candidates)
}

func bundledSkillDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return bundledSkillDirFrom(filepath.Dir(exe))
}

func runSkillInstall(cmd *cobra.Command, args []string) error {
	source, err := bundledSkillDir()
	if err != nil {
		return err
	}
	target, err := agent.SkillDir()
	if err != nil {
		return err
	}
	if err := agent.InstallSkill(source, target, skillForce); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Installed wenmar skill to %s\n", target)
	return nil
}

func runSkillUninstall(cmd *cobra.Command, args []string) error {
	target, err := agent.SkillDir()
	if err != nil {
		return err
	}
	if err := agent.UninstallSkill(target); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Removed wenmar skill from %s\n", target)
	return nil
}

func runSkillUpdate(cmd *cobra.Command, args []string) error {
	source, err := bundledSkillDir()
	if err != nil {
		return err
	}
	target, err := agent.SkillDir()
	if err != nil {
		return err
	}
	if err := agent.InstallSkill(source, target, false); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Updated wenmar skill at %s\n", target)
	return nil
}
