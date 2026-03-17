package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	cblog "github.com/illwill/cardbot/internal/log"
	"github.com/illwill/cardbot/internal/update"
)

const (
	updateCheckTimeout = 5 * time.Second
	selfUpdateTimeout  = 60 * time.Second
)

var checkLatest = update.CheckLatest

// MaybeCheckForUpdate checks for updates on every app startup.
func MaybeCheckForUpdate(logger *cblog.Logger, version string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()
	res, err := checkLatest(ctx, nil, update.DefaultAPIBase, update.DefaultRepo, version)
	if err != nil {
		if logger != nil {
			logger.Printf("Update check failed: %v", err)
		}
		return "", err
	}

	if res.Update {
		if logger != nil {
			logger.Printf("Update available: %s (current %s)", res.Latest, version)
		}
		return res.Latest, nil
	}
	if logger != nil {
		logger.Printf("Up to date (%s)", version)
	}
	return "", nil
}

// RunSelfUpdate performs a self-update to the latest version.
func RunSelfUpdate(version string) int {
	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not determine executable path: %s\n", FriendlyErr(err))
		return 1
	}

	fmt.Printf("[%s] Downloading update…\n", ts())
	ctx, cancel := context.WithTimeout(context.Background(), selfUpdateTimeout)
	defer cancel()

	installed, err := update.SelfUpdate(ctx, nil, update.DefaultAPIBase, update.DefaultRepo, version, execPath)
	if err == nil {
		fmt.Printf("[%s] Updated to %s\n", ts(), installed)
		fmt.Printf("[%s] Restart CardBot to use the new version.\n", ts())
		return 0
	}

	if errors.Is(err, update.ErrAlreadyUpToDate) {
		fmt.Printf("[%s] Already up to date (%s)\n", ts(), version)
		return 0
	}

	fmt.Fprintf(os.Stderr, "Error: %s\n", FriendlyErr(err))
	if isPermissionErr(err) {
		fmt.Fprintf(os.Stderr, "Try: sudo %q self-update\n", execPath)
	}
	return 1
}

func isPermissionErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrPermission) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "permission denied")
}
