package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	upgradeRepo = "Wenmar-Pro/wenmar-cli"
	upgradeBase = "https://github.com/Wenmar-Pro/wenmar-cli/releases/download"
)

var (
	upgradeVersion string
	upgradeCheck   bool
	upgradeForce   bool
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [version]",
	Short: "Upgrade wenmar to the latest (or a pinned) version",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runUpgrade,
}

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeCheck, "check", false, "Print the latest available version without upgrading")
	upgradeCmd.Flags().BoolVar(&upgradeForce, "force", false, "Skip confirmation")
	rootCmd.AddCommand(upgradeCmd)
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		upgradeVersion = args[0]
	}

	latest, err := fetchLatestVersion()
	if err != nil {
		return fmt.Errorf("could not determine latest version: %w", err)
	}

	if upgradeCheck {
		fmt.Fprintf(cmd.OutOrStdout(), "Latest version: %s\n", latest)
		return nil
	}

	target := upgradeVersion
	if target == "" {
		target = latest
	}

	installMethod, binPath, err := detectInstallMethod()
	if err != nil {
		return err
	}

	switch installMethod {
	case "mise":
		return runMiseUpgrade(cmd, target)
	case "homebrew":
		return runBrewUpgrade(cmd)
	case "go_install":
		return fmt.Errorf("wenmar was installed via `go install`. Upgrade with `go install github.com/wenmar-pro/wenmar-cli/cmd/wenmar@latest`")
	case "installer":
		return runInstallerUpgrade(cmd, target, binPath)
	default:
		return fmt.Errorf("could not detect install method")
	}
}

// fetchLatestVersion queries the GitHub releases/latest redirect.
func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", "https://github.com/"+upgradeRepo+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "wenmar-cli-upgrade")
	// Don't follow redirects so we can read the Location header.
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("no Location header in redirect")
	}
	version := loc[strings.LastIndex(loc, "/")+1:]
	version = strings.TrimPrefix(version, "v")
	return version, nil
}

// detectInstallMethod returns the install method and the binary path.
func detectInstallMethod() (string, string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", "", err
	}
	exe, _ = filepath.EvalSymlinks(exe)

	// mise
	if out, err := exec.Command("mise", "which", "wenmar").Output(); err == nil && strings.TrimSpace(string(out)) != "" {
		return "mise", strings.TrimSpace(string(out)), nil
	}

	// Homebrew
	if out, err := exec.Command("brew", "--prefix", "wenmar").Output(); err == nil && strings.TrimSpace(string(out)) != "" {
		return "homebrew", strings.TrimSpace(string(out)), nil
	}

	// go install
	if strings.Contains(exe, "go-build") || strings.Contains(exe, "gopath") || strings.Contains(exe, "go/bin") {
		return "go_install", exe, nil
	}

	// Installer script: ~/.local/bin or ~/bin
	home, _ := os.UserHomeDir()
	for _, dir := range []string{filepath.Join(home, ".local", "bin"), filepath.Join(home, "bin")} {
		if strings.HasPrefix(exe, dir) {
			return "installer", exe, nil
		}
	}

	return "unknown", exe, nil
}

func runMiseUpgrade(cmd *cobra.Command, target string) error {
	args := []string{"upgrade", "wenmar"}
	if target != "" {
		args = append(args, target)
	}
	c := exec.Command("mise", args...)
	c.Stdout = cmd.OutOrStdout()
	c.Stderr = cmd.ErrOrStderr()
	return c.Run()
}

func runBrewUpgrade(cmd *cobra.Command) error {
	c := exec.Command("brew", "upgrade", "wenmar")
	c.Stdout = cmd.OutOrStdout()
	c.Stderr = cmd.ErrOrStderr()
	return c.Run()
}

func runInstallerUpgrade(cmd *cobra.Command, target, binPath string) error {
	if !upgradeForce {
		fmt.Fprintf(cmd.OutOrStdout(), "Upgrade wenmar to v%s? (y/N): ", target)
		var answer string
		fmt.Fscanln(cmd.InOrStdin(), &answer)
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			fmt.Fprintln(cmd.OutOrStdout(), "Upgrade cancelled.")
			return nil
		}
	}

	platform := runtime.GOOS + "_" + runtime.GOARCH
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	archiveName := fmt.Sprintf("wenmar_%s_%s.%s", target, platform, ext)
	url := fmt.Sprintf("%s/v%s/%s", upgradeBase, target, archiveName)

	fmt.Fprintf(cmd.OutOrStdout(), "Downloading wenmar v%s...\n", target)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "wenmar-upgrade-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return err
	}
	tmp.Close()

	// Backup the current binary, then swap.
	backup := binPath + ".bak"
	if err := os.Rename(binPath, backup); err != nil {
		return fmt.Errorf("could not back up current binary: %w", err)
	}
	if err := os.Rename(tmp.Name(), binPath); err != nil {
		// Restore backup.
		_ = os.Rename(backup, binPath)
		return fmt.Errorf("could not install new binary: %w", err)
	}
	os.Chmod(binPath, 0755)
	_ = os.Remove(backup)

	fmt.Fprintf(cmd.OutOrStdout(), "Upgraded wenmar to v%s.\n", target)
	return nil
}
