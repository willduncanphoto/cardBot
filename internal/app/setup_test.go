package app

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/illwill/cardbot/internal/config"
)

func TestRunSetup_WritesNamingModeToConfig(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfgPath := filepath.Join(t.TempDir(), "config.json")

	err := RunSetup(
		cfg,
		cfgPath,
		func(string) string { return "~/Pictures/Jobs" },
		func(string) string { return config.NamingTimestamp },
		func(bool) bool { return true },
		func(bool) bool { return true },
		func(string) string { return "Ghostty" },
		func(string) string { return "~/Code" },
	)
	if err != nil {
		t.Fatalf("RunSetup error: %v", err)
	}

	loaded, warnings, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if loaded.Destination.Path != "~/Pictures/Jobs" {
		t.Fatalf("Destination.Path = %q, want %q", loaded.Destination.Path, "~/Pictures/Jobs")
	}
	if loaded.Naming.Mode != config.NamingTimestamp {
		t.Fatalf("Naming.Mode = %q, want %q", loaded.Naming.Mode, config.NamingTimestamp)
	}
	if !loaded.Daemon.Enabled {
		t.Fatalf("Daemon.Enabled = %v, want true", loaded.Daemon.Enabled)
	}
	if !loaded.Daemon.StartAtLogin {
		t.Fatalf("Daemon.StartAtLogin = %v, want true", loaded.Daemon.StartAtLogin)
	}
	if loaded.Daemon.TerminalApp != "Ghostty" {
		t.Fatalf("Daemon.TerminalApp = %q, want %q", loaded.Daemon.TerminalApp, "Ghostty")
	}
	if loaded.Daemon.WorkingDirectory != "~/Code" {
		t.Fatalf("Daemon.WorkingDirectory = %q, want %q", loaded.Daemon.WorkingDirectory, "~/Code")
	}
}

func TestRunSetup_NormalizesInvalidNamingMode(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfgPath := filepath.Join(t.TempDir(), "config.json")

	err := RunSetup(
		cfg,
		cfgPath,
		func(string) string { return "~/Pictures/Jobs" },
		func(string) string { return "banana" },
		func(bool) bool { return false },
		func(bool) bool { return true },
		func(string) string {
			t.Fatal("terminal app prompt should not be called when daemon is disabled")
			return "Ghostty"
		},
		func(string) string {
			t.Fatal("working directory prompt should not be called when daemon is disabled")
			return "~"
		},
	)
	if err != nil {
		t.Fatalf("RunSetup error: %v", err)
	}

	loaded, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded.Naming.Mode != config.NamingOriginal {
		t.Fatalf("Naming.Mode = %q, want %q", loaded.Naming.Mode, config.NamingOriginal)
	}
	if loaded.Daemon.Enabled {
		t.Fatalf("Daemon.Enabled = %v, want false", loaded.Daemon.Enabled)
	}
	if loaded.Daemon.StartAtLogin {
		t.Fatalf("Daemon.StartAtLogin = %v, want false", loaded.Daemon.StartAtLogin)
	}
}

func TestRunSetup_DaemonDisabled_SkipsStartAtLoginPrompt(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfgPath := filepath.Join(t.TempDir(), "config.json")

	err := RunSetup(
		cfg,
		cfgPath,
		func(string) string { return "~/Pictures/Jobs" },
		func(string) string { return config.NamingOriginal },
		func(bool) bool { return false },
		func(bool) bool {
			t.Fatal("start-at-login prompt should not be called when daemon is disabled")
			return true
		},
		func(string) string {
			t.Fatal("terminal app prompt should not be called when daemon is disabled")
			return "Terminal"
		},
		func(string) string {
			t.Fatal("working directory prompt should not be called when daemon is disabled")
			return "~"
		},
	)
	if err != nil {
		t.Fatalf("RunSetup error: %v", err)
	}

	loaded, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded.Daemon.StartAtLogin {
		t.Fatal("Daemon.StartAtLogin should remain false")
	}
}

func TestParseNamingChoice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{"1", config.NamingOriginal, true},
		{"original", config.NamingOriginal, true},
		{"o", config.NamingOriginal, true},
		{"2", config.NamingTimestamp, true},
		{"timestamp", config.NamingTimestamp, true},
		{"t", config.NamingTimestamp, true},
		{"nope", "", false},
		{"", "", false},
		{"  2  ", config.NamingTimestamp, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("input_%q", tt.in), func(t *testing.T) {
			got, ok := parseNamingChoice(tt.in)
			if ok != tt.wantOK {
				t.Errorf("parseNamingChoice(%q) ok = %v, want %v", tt.in, ok, tt.wantOK)
				return
			}
			if got != tt.want {
				t.Errorf("parseNamingChoice(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseYesNo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in     string
		want   bool
		wantOK bool
	}{
		{"y", true, true},
		{"Y", true, true},
		{"yes", true, true},
		{"n", false, true},
		{"no", false, true},
		{"maybe", false, false},
		{"", false, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("input_%q", tt.in), func(t *testing.T) {
			got, ok := parseYesNo(tt.in)
			if ok != tt.wantOK {
				t.Errorf("parseYesNo(%q) ok = %v, want %v", tt.in, ok, tt.wantOK)
				return
			}
			if got != tt.want {
				t.Errorf("parseYesNo(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseDaemonTerminalChoice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{"1", "Default", true},
		{"default", "Default", true},
		{"2", "Terminal", true},
		{"terminal", "Terminal", true},
		{"3", "Ghostty", true},
		{"ghostty", "Ghostty", true},
		{"4", "", true},
		{"custom", "", true},
		{"wat", "", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("input_%q", tt.in), func(t *testing.T) {
			got, ok := parseDaemonTerminalChoice(tt.in)
			if ok != tt.wantOK {
				t.Errorf("parseDaemonTerminalChoice(%q) ok = %v, want %v", tt.in, ok, tt.wantOK)
				return
			}
			if got != tt.want {
				t.Errorf("parseDaemonTerminalChoice(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestPromptDaemonTerminalAppIO_Default(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("\n")
	var out bytes.Buffer

	app := promptDaemonTerminalAppIO(in, &out, "Terminal")
	if app != "Terminal" {
		t.Fatalf("app = %q, want %q", app, "Terminal")
	}
	if !strings.Contains(out.String(), "Choice [2]:") {
		t.Fatalf("expected default [2] prompt, got:\n%s", out.String())
	}
}

func TestPromptDaemonTerminalAppIO_ChooseDefault(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("1\n")
	var out bytes.Buffer

	app := promptDaemonTerminalAppIO(in, &out, "Terminal")
	if app != "Default" {
		t.Fatalf("app = %q, want %q", app, "Default")
	}
}

func TestPromptDaemonTerminalAppIO_ChooseGhostty(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("3\n")
	var out bytes.Buffer

	app := promptDaemonTerminalAppIO(in, &out, "Terminal")
	if app != "Ghostty" {
		t.Fatalf("app = %q, want %q", app, "Ghostty")
	}
}

func TestPromptDaemonTerminalAppIO_Custom(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("4\nWezTerm\n")
	var out bytes.Buffer

	app := promptDaemonTerminalAppIO(in, &out, "Terminal")
	if app != "WezTerm" {
		t.Fatalf("app = %q, want %q", app, "WezTerm")
	}
}

func TestPromptDaemonWorkingDirectoryIO_Default(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("\n")
	var out bytes.Buffer

	dir := promptDaemonWorkingDirectoryIO(in, &out, "~")
	if dir != "~" {
		t.Fatalf("dir = %q, want %q", dir, "~")
	}
	if !strings.Contains(out.String(), "Path [~]:") {
		t.Fatalf("expected default path prompt, got:\n%s", out.String())
	}
}

func TestPromptDaemonWorkingDirectoryIO_Custom(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("~/Code\n")
	var out bytes.Buffer

	dir := promptDaemonWorkingDirectoryIO(in, &out, "~")
	if dir != "~/Code" {
		t.Fatalf("dir = %q, want %q", dir, "~/Code")
	}
}

func TestPromptDaemonEnabledIO_DefaultFalse(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("\n")
	var out bytes.Buffer

	enabled := promptDaemonEnabledIO(in, &out, false)
	if enabled {
		t.Fatal("enabled = true, want false")
	}
	if !strings.Contains(out.String(), "Choice [n]:") {
		t.Fatalf("expected default [n] prompt, got:\n%s", out.String())
	}
}

func TestPromptDaemonEnabledIO_DefaultTrue(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("\n")
	var out bytes.Buffer

	enabled := promptDaemonEnabledIO(in, &out, true)
	if !enabled {
		t.Fatal("enabled = false, want true")
	}
	if !strings.Contains(out.String(), "Choice [y]:") {
		t.Fatalf("expected default [y] prompt, got:\n%s", out.String())
	}
}

func TestPromptDaemonEnabledIO_InvalidThenValid(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("wat\ny\n")
	var out bytes.Buffer

	enabled := promptDaemonEnabledIO(in, &out, false)
	if !enabled {
		t.Fatal("enabled = false, want true")
	}
	if !strings.Contains(out.String(), "Please enter y or n.") {
		t.Fatalf("expected invalid-input message, got:\n%s", out.String())
	}
}

func TestPromptDaemonEnabledIO_EOF(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("")
	var out bytes.Buffer

	enabled := promptDaemonEnabledIO(in, &out, true)
	if !enabled {
		t.Fatal("enabled = false, want true on EOF default")
	}
}

func TestPromptDaemonStartAtLoginIO_DefaultFalse(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("\n")
	var out bytes.Buffer

	enabled := promptDaemonStartAtLoginIO(in, &out, false)
	if enabled {
		t.Fatal("enabled = true, want false")
	}
	if !strings.Contains(out.String(), "Choice [n]:") {
		t.Fatalf("expected default [n] prompt, got:\n%s", out.String())
	}
}

func TestPromptDaemonStartAtLoginIO_DefaultTrue(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("\n")
	var out bytes.Buffer

	enabled := promptDaemonStartAtLoginIO(in, &out, true)
	if !enabled {
		t.Fatal("enabled = false, want true")
	}
	if !strings.Contains(out.String(), "Choice [y]:") {
		t.Fatalf("expected default [y] prompt, got:\n%s", out.String())
	}
}

func TestPromptDaemonStartAtLoginIO_InvalidThenValid(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("wat\ny\n")
	var out bytes.Buffer

	enabled := promptDaemonStartAtLoginIO(in, &out, false)
	if !enabled {
		t.Fatal("enabled = false, want true")
	}
	if !strings.Contains(out.String(), "Please enter y or n.") {
		t.Fatalf("expected invalid-input message, got:\n%s", out.String())
	}
}

func TestPromptDaemonStartAtLoginIO_EOF(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("")
	var out bytes.Buffer

	enabled := promptDaemonStartAtLoginIO(in, &out, true)
	if !enabled {
		t.Fatal("enabled = false, want true on EOF default")
	}
}

func TestPromptNamingModeIO_DefaultOriginal(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("\n")
	var out bytes.Buffer

	mode := promptNamingModeIO(in, &out, config.NamingOriginal)
	if mode != config.NamingOriginal {
		t.Fatalf("mode = %q, want %q", mode, config.NamingOriginal)
	}
	if !strings.Contains(out.String(), "Choice [1]:") {
		t.Fatalf("expected default [1] prompt, got:\n%s", out.String())
	}
}

func TestPromptNamingModeIO_DefaultTimestamp(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("\n")
	var out bytes.Buffer

	mode := promptNamingModeIO(in, &out, config.NamingTimestamp)
	if mode != config.NamingTimestamp {
		t.Fatalf("mode = %q, want %q", mode, config.NamingTimestamp)
	}
	if !strings.Contains(out.String(), "Choice [2]:") {
		t.Fatalf("expected default [2] prompt, got:\n%s", out.String())
	}
}

func TestPromptNamingModeIO_InvalidThenValid(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("x\n2\n")
	var out bytes.Buffer

	mode := promptNamingModeIO(in, &out, config.NamingOriginal)
	if mode != config.NamingTimestamp {
		t.Fatalf("mode = %q, want %q", mode, config.NamingTimestamp)
	}
	if !strings.Contains(out.String(), "Please enter 1 or 2.") {
		t.Fatalf("expected invalid-input message, got:\n%s", out.String())
	}
}

func TestPromptNamingModeIO_EOF(t *testing.T) {
	t.Parallel()
	// Simulate EOF (no input at all).
	in := strings.NewReader("")
	var out bytes.Buffer

	mode := promptNamingModeIO(in, &out, config.NamingTimestamp)
	if mode != config.NamingTimestamp {
		t.Fatalf("mode = %q, want %q (should return default on EOF)", mode, config.NamingTimestamp)
	}
}

func TestSetupPrompter_SequentialPromptsShareInputStream(t *testing.T) {
	t.Parallel()

	in := strings.NewReader("2\ny\ny\n3\n~/Code\n")
	var out bytes.Buffer
	p := NewSetupPrompter(in, &out)

	if mode := p.PromptNamingMode(config.NamingOriginal); mode != config.NamingTimestamp {
		t.Fatalf("PromptNamingMode = %q, want %q", mode, config.NamingTimestamp)
	}
	if enabled := p.PromptDaemonEnabled(false); !enabled {
		t.Fatal("PromptDaemonEnabled = false, want true")
	}
	if startAtLogin := p.PromptDaemonStartAtLogin(false); !startAtLogin {
		t.Fatal("PromptDaemonStartAtLogin = false, want true")
	}
	if appName := p.PromptDaemonTerminalApp("Terminal"); appName != "Ghostty" {
		t.Fatalf("PromptDaemonTerminalApp = %q, want %q", appName, "Ghostty")
	}
	if dir := p.PromptDaemonWorkingDirectory("~"); dir != "~/Code" {
		t.Fatalf("PromptDaemonWorkingDirectory = %q, want %q", dir, "~/Code")
	}
}

func TestNamingDisplayLine(t *testing.T) {
	t.Parallel()

	t.Run("original", func(t *testing.T) {
		got := namingDisplayLine(config.NamingOriginal)
		want := "Camera original"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("timestamp", func(t *testing.T) {
		got := namingDisplayLine(config.NamingTimestamp)
		want := "Timestamp + sequence"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}
