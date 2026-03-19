package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDaemonLaunchHint_Automation(t *testing.T) {
	t.Parallel()

	tests := []string{
		"not authorized to send Apple events to Terminal",
		"not authorised to send apple events",
		"execution error: Not authorized to send Apple events to Terminal. (-1743)",
		"Error Domain=NSOSStatusErrorDomain Code=-1743",
		"automation permission denied",
		"errAEEventNotPermitted",
	}

	for _, msg := range tests {
		t.Run(fmt.Sprintf("msg_%q", msg), func(t *testing.T) {
			hint := daemonLaunchHint(errors.New(msg))
			if !strings.Contains(strings.ToLower(hint), "automation") {
				t.Fatalf("hint = %q, expected automation guidance for msg=%q", hint, msg)
			}
		})
	}
}

func TestDaemonLaunchHint_FullDiskAccess(t *testing.T) {
	t.Parallel()

	tests := []string{
		"operation not permitted",
		"permission denied",
		"open: EPERM",
	}

	for _, msg := range tests {
		t.Run(fmt.Sprintf("msg_%q", msg), func(t *testing.T) {
			hint := daemonLaunchHint(errors.New(msg))
			if !strings.Contains(strings.ToLower(hint), "full disk access") {
				t.Fatalf("hint = %q, expected full disk access guidance for msg=%q", hint, msg)
			}
		})
	}
}

func TestDaemonLaunchHint_Unknown(t *testing.T) {
	t.Parallel()

	hint := daemonLaunchHint(errors.New("random error"))
	if hint != "" {
		t.Fatalf("hint = %q, want empty", hint)
	}
}
