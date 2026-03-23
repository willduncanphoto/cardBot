//go:build darwin

package pick

import (
	"os/exec"
	"strings"
)

// Folder opens the native macOS folder picker dialog and returns the chosen path.
func Folder(defaultPath string) (string, error) {
	script := folderPickerScript(defaultPath)
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
