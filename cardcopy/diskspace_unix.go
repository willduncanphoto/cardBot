//go:build darwin || linux

package cardcopy

import "syscall"

// diskFreeBytes returns the number of available bytes at path and true.
// Returns 0, false if the query fails.
func diskFreeBytes(path string) (int64, bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, false
	}
	return int64(stat.Bavail) * int64(stat.Bsize), true
}
