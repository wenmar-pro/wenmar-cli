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
	Use:   "skill",
	Short: "Manage the wenmar agent skill",
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

func bundledSkillDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	// The bundled skill lives at <binary-dir>/../skills/wenmar or
	// <binary-dir>/skills/wenmar. Fall back to the repo layout for dev.
	candidates := []string{
		filepath.Join(filepath.Dir(exe), "skills", "wenmar"),
		filepath.Join(filepath.Dir(exe), "..", "skills", "wenmar"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "SKILL.md")); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("bundled skill not found near the wenmar binary")
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
