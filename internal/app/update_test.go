package app

import (
	"errors"
	"testing"
)

func TestStartupUpdateMessages(t *testing.T) {
	t.Parallel()

	t.Run("no signal on error", func(t *testing.T) {
		status, action := StartupUpdateMessages("", errors.New("boom"))
		if status != "NO SIGNAL" {
			t.Fatalf("status = %q, want %q", status, "NO SIGNAL")
		}
		if action != "" {
			t.Fatalf("action = %q, want empty", action)
		}
	})

	t.Run("update available", func(t *testing.T) {
		status, action := StartupUpdateMessages("0.4.1", nil)
		if status != "UPDATE AVAILABLE (0.4.1)" {
			t.Fatalf("status = %q, want %q", status, "UPDATE AVAILABLE (0.4.1)")
		}
		if action != "Run: cardbot self-update" {
			t.Fatalf("action = %q, want %q", action, "Run: cardbot self-update")
		}
	})

	t.Run("up to date", func(t *testing.T) {
		status, action := StartupUpdateMessages("", nil)
		if status != "Up to date" {
			t.Fatalf("status = %q, want %q", status, "Up to date")
		}
		if action != "" {
			t.Fatalf("action = %q, want empty", action)
		}
	})
}
