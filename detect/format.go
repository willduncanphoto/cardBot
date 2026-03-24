package detect

import (
	"fmt"
	"strings"
)

// IsHidden reports whether a filename should be skipped (dot-prefixed).
// Shared by analyze and cardcopy during DCIM tree walks.
func IsHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

// FormatBytes converts bytes to human readable string.
// This is in a platform-agnostic file (no build constraints) because it's a
// pure function used across the codebase — detect, copy, and main all call it.
func FormatBytes(b int64) string {
	const unit = 1024
	if b <= 0 {
		return "0 B"
	}
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
