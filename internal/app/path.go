package app

import (
	"path/filepath"
	"strings"
)

func normalizeCardPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

func sameCardPath(a, b string) bool {
	a = normalizeCardPath(a)
	b = normalizeCardPath(b)
	if a == "" || b == "" {
		return false
	}
	return a == b
}
