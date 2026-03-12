package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const schemaVersion = "cardbot-config-v1"

// Config holds all CardBot configuration.
type Config struct {
	Schema      string      `json:"$schema"`
	Destination Destination `json:"destination"`
	Output      Output      `json:"output"`
	Advanced    Advanced    `json:"advanced"`
}

// Destination settings.
type Destination struct {
	Path string `json:"path"`
}

// Output settings.
type Output struct {
	Color bool `json:"color"`
	Quiet bool `json:"quiet"`
}

// Advanced settings.
type Advanced struct {
	BufferSizeKB int    `json:"buffer_size_kb"`
	ExifWorkers  int    `json:"exif_workers"`
	LogFile      string `json:"log_file"`
}

// Defaults returns a Config populated with built-in defaults.
func Defaults() *Config {
	return &Config{
		Schema: schemaVersion,
		Destination: Destination{
			Path: "~/Pictures/CardBot",
		},
		Output: Output{
			Color: true,
			Quiet: false,
		},
		Advanced: Advanced{
			BufferSizeKB: 256,
			ExifWorkers:  4,
			LogFile:      "~/.cardbot/cardbot.log",
		},
	}
}

// Path returns the default config file path.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "cardbot", "config.json"), nil
}

// Load reads the config file and merges it over defaults.
// Returns (config, warnings, error).
func Load(path string) (*Config, []string, error) {
	cfg := Defaults()
	var warnings []string

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil, nil
		}
		return cfg, nil, fmt.Errorf("reading config: %w", err)
	}

	// Parse schema field first to verify compatibility, then unmarshal into config.
	// Use a single json.Unmarshal into a struct that embeds both to avoid parsing twice.
	var probe struct {
		Schema string `json:"$schema"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return cfg, []string{"config file is malformed JSON, using defaults"}, nil
	}
	if probe.Schema != "" && probe.Schema != schemaVersion {
		warnings = append(warnings, fmt.Sprintf("unknown config schema %q, using defaults", probe.Schema))
		return cfg, warnings, nil
	}

	// Unmarshal into config (merges over defaults since cfg already has them).
	if err := json.Unmarshal(data, cfg); err != nil {
		warnings = append(warnings, "config file could not be parsed, using defaults")
		return Defaults(), warnings, nil
	}

	// Validate and clamp.
	if cfg.Advanced.BufferSizeKB < 64 {
		warnings = append(warnings, fmt.Sprintf("buffer_size_kb %d is below minimum 64, using 64", cfg.Advanced.BufferSizeKB))
		cfg.Advanced.BufferSizeKB = 64
	} else if cfg.Advanced.BufferSizeKB > 4096 {
		warnings = append(warnings, fmt.Sprintf("buffer_size_kb %d exceeds maximum 4096, using 4096", cfg.Advanced.BufferSizeKB))
		cfg.Advanced.BufferSizeKB = 4096
	}
	if cfg.Advanced.ExifWorkers < 1 {
		warnings = append(warnings, fmt.Sprintf("exif_workers %d is below minimum 1, using 1", cfg.Advanced.ExifWorkers))
		cfg.Advanced.ExifWorkers = 1
	} else if cfg.Advanced.ExifWorkers > 16 {
		warnings = append(warnings, fmt.Sprintf("exif_workers %d exceeds maximum 16, using 16", cfg.Advanced.ExifWorkers))
		cfg.Advanced.ExifWorkers = 16
	}

	return cfg, warnings, nil
}

// Save writes cfg to path, creating parent directories as needed.
// File is created with 0600 permissions.
func Save(cfg *Config, path string) error {
	cfg.Schema = schemaVersion
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// ExpandPath expands a leading ~ to the user's home directory.
func ExpandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path, err
	}
	return filepath.Join(home, path[1:]), nil
}

// ContractPath replaces the user's home directory prefix with ~.
// Reverses ExpandPath for display and storage.
func ContractPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	// Clean trailing slashes for consistent comparison.
	path = filepath.Clean(path)
	home = filepath.Clean(home)
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+string(filepath.Separator)) {
		return "~" + path[len(home):]
	}
	return path
}
