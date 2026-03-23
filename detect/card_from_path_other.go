//go:build !darwin && !linux

package detect

import "path/filepath"

// CardFromPath provides a best-effort Card on unsupported platforms.
func CardFromPath(path string) *Card {
	path = filepath.Clean(path)
	if path == "" {
		return nil
	}
	return &Card{
		Path:  path,
		Name:  filepath.Base(path),
		Brand: "Unknown",
	}
}
