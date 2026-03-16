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

func TestNamingDisplayLine(t *testing.T) {
	t.Parallel()

	t.Run("original", func(t *testing.T) {
		got := namingDisplayLine(config.NamingOriginal, 3048)
		want := "Camera original (DSC_xxxx.NEF)"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("timestamp", func(t *testing.T) {
		got := namingDisplayLine(config.NamingTimestamp, 3048)
		want := "Timestamp + sequence (xxxx = 0001-9999)"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}
