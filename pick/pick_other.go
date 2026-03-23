//go:build !darwin

package pick

import "errors"

// Folder is not supported on this platform.
func Folder(defaultPath string) (string, error) {
	return "", errors.New("native folder picker not available on this platform")
}
