package cardcopy

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/illwill/cardbot/config"
)

// SequenceDigits is the fixed sequence padding width.
// 4 digits (0001–9999) prevents rollover on heavy shoot days (1000+ shots).
const SequenceDigits = 4

// sequenceMax returns the maximum sequence number for the given digit width.
func sequenceMax(digits int) int {
	switch digits {
	case 3:
		return 999
	case 4:
		return 9999
	default:
		return 99999
	}
}

func formatSequence(n, digits int) string {
	// Sequence is 1-based: 0001-9999, loop back to 0001
	if n < 1 {
		n = 1
	}
	if digits < 3 {
		digits = 3
	}
	if digits > 5 {
		digits = 5
	}
	return fmt.Sprintf("%0*d", digits, n)
}

func timestampStem(t time.Time) string {
	if t.IsZero() {
		t = time.Now()
	}
	return t.Format("060102T150405")
}

func renamedRelativePath(relPath string, captureTime time.Time, seq, digits int) string {
	dir := filepath.Dir(relPath)
	ext := strings.ToUpper(filepath.Ext(relPath))
	name := timestampStem(captureTime) + "_" + formatSequence(seq, digits) + ext
	if dir == "." || dir == "" {
		return name
	}
	return filepath.Join(dir, name)
}

// isTimestampMode returns whether the naming mode string means timestamp renaming.
func isTimestampMode(mode string) bool {
	return config.NormalizeNamingMode(mode) == config.NamingTimestamp
}
