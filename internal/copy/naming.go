package copy

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/illwill/cardbot/internal/config"
)

// SequenceDigits returns the sequence padding (fixed at 3 for 0.3.x).
// Future: per-date detection for cards with >999 files in a single day.
func SequenceDigits(totalFiles int) int {
	_ = totalFiles // reserved for future per-date detection
	return 3
}

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
	// Sequence is 1-based: 001-999, loop back to 001
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
