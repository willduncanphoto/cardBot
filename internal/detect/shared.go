//go:build darwin || linux

package detect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// buildCard constructs a Card from a mount path and volume name.
// Returns nil if filesystem stats cannot be read.
func buildCard(path, name string) *Card {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil
	}

	total := int64(stat.Blocks) * int64(stat.Bsize)
	free := int64(stat.Bavail) * int64(stat.Bsize)
	used := total - free

	// Get hardware info (may fail silently on some systems/readers)
	hardware, _ := GetHardwareInfo(path)

	return &Card{
		Path:       path,
		Name:       name,
		TotalBytes: total,
		UsedBytes:  used,
		Brand:      detectBrand(path),
		Hardware:   hardware,
	}
}

// detectBrand identifies camera brand from DCIM subfolder naming patterns.
// Supports Nikon, Canon, Sony, Fujifilm, Panasonic, and Olympus.
// See docs/FOOTNOTES.md for confidence levels and verification status.
func detectBrand(path string) string {
	dcim := filepath.Join(path, "DCIM")
	entries, err := os.ReadDir(dcim)
	if err != nil {
		return "Unknown"
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.ToUpper(entry.Name())

		// Nikon: 100NIKON, 100NCZ_9 (Z9), 100NCZ_8 (Z8), 100NZ_6 (Z6), 100ND850 (D850)
		// "ND" requires a digit suffix to avoid false positives (e.g. STANDARD, ANDROID).
		// Confidence: High for NIKON, Medium for NCZ_/NZ_/ND prefix patterns.
		if strings.Contains(name, "NIKON") ||
			strings.Contains(name, "NCZ") ||
			strings.Contains(name, "NZ_") ||
			containsNDModel(name) {
			return "Nikon"
		}

		// Canon: 100CANON, 100EOS5D, 100EOSR5
		// Confidence: High for CANON, Medium for EOS patterns.
		if strings.Contains(name, "CANON") ||
			strings.Contains(name, "EOS") {
			return "Canon"
		}

		// Sony: 100MSDCF (consistent across Alpha series), 101SONY
		// Confidence: High.
		if strings.Contains(name, "MSDCF") ||
			strings.Contains(name, "SONY") {
			return "Sony"
		}

		// Fujifilm: 100_FUJI, 101_FUJI
		// Confidence: High.
		if strings.Contains(name, "FUJI") {
			return "Fujifilm"
		}

		// Panasonic: 100_PANA (inferred, needs verification)
		// Confidence: Low.
		if strings.Contains(name, "PANA") ||
			strings.Contains(name, "LUMIX") {
			return "Panasonic"
		}

		// Olympus: 100OLYMP
		// Confidence: Medium — older Olympus confirmed, OM System uncertain.
		if strings.Contains(name, "OLYMP") {
			return "Olympus"
		}
	}

	return "Unknown"
}

// containsNDModel reports whether s contains an "ND" followed immediately by a digit,
// matching Nikon D-series folder names like 100ND850 or 100ND750 while rejecting
// common false positives like STANDARD or ANDROID.
func containsNDModel(s string) bool {
	for i := 0; i+2 < len(s); i++ {
		if s[i] == 'N' && s[i+1] == 'D' && s[i+2] >= '0' && s[i+2] <= '9' {
			return true
		}
	}
	return false
}

// FormatBytes converts bytes to human readable string.
// Used across the detect package for consistent formatting.
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
