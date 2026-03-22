package main

import (
	"os"
	"testing"
)

func TestBoolEnabled(t *testing.T) {
	t.Parallel()

	if got := boolEnabled(true); got != "enabled" {
		t.Fatalf("boolEnabled(true) = %q, want %q", got, "enabled")
	}
	if got := boolEnabled(false); got != "disabled" {
		t.Fatalf("boolEnabled(false) = %q, want %q", got, "disabled")
	}
}

func TestBoolYesNo(t *testing.T) {
	t.Parallel()

	if got := boolYesNo(true); got != "yes" {
		t.Fatalf("boolYesNo(true) = %q, want %q", got, "yes")
	}
	if got := boolYesNo(false); got != "no" {
		t.Fatalf("boolYesNo(false) = %q, want %q", got, "no")
	}
}

func TestDaemonTerminalAppLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{"default", "Default (macOS)"},
		{"Default", "Default (macOS)"},
		{"  default  ", "Default (macOS)"},
		{"Terminal", "Terminal"},
		{"Ghostty", "Ghostty"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := daemonTerminalAppLabel(tt.in); got != tt.want {
			t.Errorf("daemonTerminalAppLabel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestResolveDaemonWorkingDirectory(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty defaults to home", "", home},
		{"whitespace defaults to home", "   ", home},
		{"tilde expands to home", "~", home},
		{"absolute path passes through", "/tmp/test", "/tmp/test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveDaemonWorkingDirectory(tt.in)
			if got != tt.want {
				t.Errorf("resolveDaemonWorkingDirectory(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDaemonWorkingDirectoryLabel(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	tests := []struct {
		name        string
		configValue string
		resolved    string
		want        string
	}{
		{"empty config shows tilde", "", home, "~"},
		{"tilde contracts cleanly", "~", home, "~"},
		{"absolute path differs", "~/Photos", home + "/Photos", "~/Photos"},
		{"non-contractable path", "/opt/data", "/opt/data", "/opt/data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := daemonWorkingDirectoryLabel(tt.configValue, tt.resolved)
			if got != tt.want {
				t.Errorf("daemonWorkingDirectoryLabel(%q, %q) = %q, want %q",
					tt.configValue, tt.resolved, got, tt.want)
			}
		})
	}
}
