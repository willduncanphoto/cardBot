package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	t.Parallel()
	cfg := Defaults()
	if cfg.Schema != schemaVersion {
		t.Errorf("Schema = %q, want %q", cfg.Schema, schemaVersion)
	}
	if cfg.Destination.Path != "~/Pictures/CardBot" {
		t.Errorf("Destination.Path = %q, want ~/Pictures/CardBot", cfg.Destination.Path)
	}
	if cfg.Naming.Mode != NamingOriginal {
		t.Errorf("Naming.Mode = %q, want %q", cfg.Naming.Mode, NamingOriginal)
	}
	if cfg.Advanced.BufferSizeKB != 256 {
		t.Errorf("BufferSizeKB = %d, want 256", cfg.Advanced.BufferSizeKB)
	}
	if cfg.Advanced.ExifWorkers != 4 {
		t.Errorf("ExifWorkers = %d, want 4", cfg.Advanced.ExifWorkers)
	}
	if !cfg.Output.Color {
		t.Error("Color should default to true")
	}
	if cfg.Update.LastCheck != "" {
		t.Errorf("Update.LastCheck = %q, want empty", cfg.Update.LastCheck)
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := Defaults()
	cfg.Destination.Path = "~/Photos/Test"
	cfg.Naming.Mode = NamingTimestamp

	if err := Save(cfg, path); err != nil {
		t.Fatal(err)
	}

	loaded, warnings, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if loaded.Destination.Path != "~/Photos/Test" {
		t.Errorf("Destination.Path = %q, want ~/Photos/Test", loaded.Destination.Path)
	}
	if loaded.Naming.Mode != NamingTimestamp {
		t.Errorf("Naming.Mode = %q, want %q", loaded.Naming.Mode, NamingTimestamp)
	}
	if loaded.Schema != schemaVersion {
		t.Errorf("Schema = %q, want %q", loaded.Schema, schemaVersion)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	cfg, warnings, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	// Should return defaults.
	if cfg.Destination.Path != "~/Pictures/CardBot" {
		t.Errorf("expected defaults, got path %q", cfg.Destination.Path)
	}
}

func TestLoad_MalformedJSON(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, warnings, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if cfg.Destination.Path != "~/Pictures/CardBot" {
		t.Error("should return defaults for malformed JSON")
	}
}

func TestLoad_WrongSchema(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"$schema": "cardbot-config-v99"}`), 0600); err != nil {
		t.Fatal(err)
	}

	_, warnings, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}

func TestLoad_ClampBufferSizeKB(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		json     string
		want     int
		wantWarn bool
	}{
		{"too small", `{"$schema":"cardbot-config-v1","advanced":{"buffer_size_kb":10}}`, 64, true},
		{"too large", `{"$schema":"cardbot-config-v1","advanced":{"buffer_size_kb":9999}}`, 4096, true},
		{"valid", `{"$schema":"cardbot-config-v1","advanced":{"buffer_size_kb":512}}`, 512, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(tt.json), 0600); err != nil {
				t.Fatal(err)
			}

			cfg, warnings, err := Load(path)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Advanced.BufferSizeKB != tt.want {
				t.Errorf("BufferSizeKB = %d, want %d", cfg.Advanced.BufferSizeKB, tt.want)
			}
			if tt.wantWarn && len(warnings) == 0 {
				t.Error("expected a warning")
			}
			if !tt.wantWarn && len(warnings) != 0 {
				t.Errorf("unexpected warnings: %v", warnings)
			}
		})
	}
}

func TestLoad_ClampExifWorkers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		json     string
		want     int
		wantWarn bool
	}{
		{"too small", `{"$schema":"cardbot-config-v1","advanced":{"exif_workers":0}}`, 1, true},
		{"too large", `{"$schema":"cardbot-config-v1","advanced":{"exif_workers":99}}`, 16, true},
		{"valid", `{"$schema":"cardbot-config-v1","advanced":{"exif_workers":8}}`, 8, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(tt.json), 0600); err != nil {
				t.Fatal(err)
			}

			cfg, warnings, err := Load(path)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Advanced.ExifWorkers != tt.want {
				t.Errorf("ExifWorkers = %d, want %d", cfg.Advanced.ExifWorkers, tt.want)
			}
			if tt.wantWarn && len(warnings) == 0 {
				t.Error("expected a warning")
			}
		})
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	t.Parallel()
	// Only override destination; everything else should use defaults.
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"$schema":"cardbot-config-v1","destination":{"path":"~/Custom"}}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, warnings, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if cfg.Destination.Path != "~/Custom" {
		t.Errorf("Destination.Path = %q, want ~/Custom", cfg.Destination.Path)
	}
	// All unset fields should preserve defaults.
	defaults := Defaults()
	if cfg.Advanced.BufferSizeKB != defaults.Advanced.BufferSizeKB {
		t.Errorf("BufferSizeKB = %d, want %d (default)", cfg.Advanced.BufferSizeKB, defaults.Advanced.BufferSizeKB)
	}
	if cfg.Advanced.ExifWorkers != defaults.Advanced.ExifWorkers {
		t.Errorf("ExifWorkers = %d, want %d (default)", cfg.Advanced.ExifWorkers, defaults.Advanced.ExifWorkers)
	}
	if cfg.Advanced.LogFile != defaults.Advanced.LogFile {
		t.Errorf("LogFile = %q, want %q (default)", cfg.Advanced.LogFile, defaults.Advanced.LogFile)
	}
	if cfg.Output.Color != defaults.Output.Color {
		t.Errorf("Color = %v, want %v (default)", cfg.Output.Color, defaults.Output.Color)
	}
	if cfg.Naming.Mode != defaults.Naming.Mode {
		t.Errorf("Naming.Mode = %q, want %q (default)", cfg.Naming.Mode, defaults.Naming.Mode)
	}
	if cfg.Update.LastCheck != defaults.Update.LastCheck {
		t.Errorf("Update.LastCheck = %q, want %q (default)", cfg.Update.LastCheck, defaults.Update.LastCheck)
	}
}

func TestLoad_InvalidNamingMode(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"$schema":"cardbot-config-v1","naming":{"mode":"banana"}}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, warnings, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Naming.Mode != NamingOriginal {
		t.Errorf("Naming.Mode = %q, want %q", cfg.Naming.Mode, NamingOriginal)
	}
	if len(warnings) == 0 {
		t.Fatal("expected warning for invalid naming mode")
	}
}

func TestNormalizeNamingMode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want string
	}{
		{"", NamingOriginal},
		{"original", NamingOriginal},
		{"ORIGINAL", NamingOriginal},
		{"timestamp", NamingTimestamp},
		{" TIMESTAMP ", NamingTimestamp},
		{"nope", NamingOriginal},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("input_%q", tt.in), func(t *testing.T) {
			if got := NormalizeNamingMode(tt.in); got != tt.want {
				t.Errorf("NormalizeNamingMode(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	t.Parallel()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/Pictures", filepath.Join(home, "Pictures")},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~bob/path", "~bob/path"}, // ~user syntax not expanded
	}

	for _, tt := range tests {
		got, err := ExpandPath(tt.input)
		if err != nil {
			t.Errorf("ExpandPath(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestContractPath(t *testing.T) {
	t.Parallel()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{filepath.Join(home, "Pictures"), "~/Pictures"},
		{filepath.Join(home, ""), "~"},
		{home + "/", "~"}, // trailing slash normalized
		{"/other/path", "/other/path"},
	}

	for _, tt := range tests {
		got := ContractPath(tt.input)
		if got != tt.want {
			t.Errorf("ContractPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExpandContractRoundTrip(t *testing.T) {
	t.Parallel()
	original := "~/Pictures/CardBot"
	expanded, err := ExpandPath(original)
	if err != nil {
		t.Fatal(err)
	}
	contracted := ContractPath(expanded)
	if contracted != original {
		t.Errorf("round-trip: %q → %q → %q", original, expanded, contracted)
	}
}

func TestSave_CreatesParentDirs(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "a", "b", "c", "config.json")
	cfg := Defaults()

	if err := Save(cfg, path); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Error("config file should exist after Save")
	}
}
