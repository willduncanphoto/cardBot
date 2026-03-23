package pick

import (
	"fmt"
	"strings"
)

func escapeAppleScriptPath(path string) string {
	safe := strings.ReplaceAll(path, `\`, `\\`)
	safe = strings.ReplaceAll(safe, `"`, `\"`)
	return safe
}

func folderPickerScript(defaultPath string) string {
	safe := escapeAppleScriptPath(defaultPath)
	return fmt.Sprintf(
		`POSIX path of (choose folder with prompt "Where should CardBot copy your work?" default location POSIX file "%s")`,
		safe,
	)
}
