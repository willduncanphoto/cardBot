//go:build darwin || linux

package detect

import "path/filepath"

// CardFromPath builds a Card from a mount path using filesystem stats and
// quick hardware enrichment. Returns nil when the path cannot be inspected.
func CardFromPath(path string) *Card {
	path = filepath.Clean(path)
	if path == "" {
		return nil
	}
	return buildCard(path, filepath.Base(path))
}
