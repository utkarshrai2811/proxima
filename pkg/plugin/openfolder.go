package plugin

import (
	"os/exec"
	"runtime"
)

// OpenPluginDir opens the plugin directory in the native file manager.
// Platform-aware (no hardcoded xdg-open):
//   - macOS:   open <dir>
//   - Linux:   xdg-open <dir>
//   - Windows: explorer <dir>
func OpenPluginDir(dir string) error {
	return openCommand(dir).Start() // non-blocking
}

func openCommand(dir string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", dir)
	case "windows":
		return exec.Command("explorer", dir)
	default: // linux and other unix
		return exec.Command("xdg-open", dir)
	}
}
