package oauth

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openBrowser opens the default browser to the given URL.
// It uses platform-specific commands:
//   - macOS: open
//   - Linux: xdg-open
//   - Windows: cmd /c start
//
// If the platform is unrecognized or the command fails, it returns an
// error so the caller can print the URL for the user to open manually.
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	// Don't wait for the browser to close
	_ = cmd.Process.Release()
	return nil
}
