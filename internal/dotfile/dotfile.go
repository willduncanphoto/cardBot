// Package dotfile reads and writes .cardbot files on memory cards.
package dotfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const fileName = ".cardbot"

// Status represents the copy state of a card.
type Status struct {
	Copied     bool      // Whether the card has been copied
	CopiedAt   time.Time // When the last copy completed
	CopiedDest string    // Where files were copied to
}

// dotfileSchema is the on-disk JSON structure.
type dotfileSchema struct {
	Schema         string `json:"$schema"`
	LastCopied     string `json:"last_copied"`
	Mode           string `json:"mode"`
	Destination    string `json:"destination"`
	FilesCopied    int    `json:"files_copied,omitempty"`
	BytesCopied    int64  `json:"bytes_copied,omitempty"`
	Verified       bool   `json:"verified,omitempty"`
	CardbotVersion string `json:"cardbot_version,omitempty"`
}

// Read checks for a .cardbot file on the card and returns its status.
// Returns a "New" status (Copied=false) if the file doesn't exist or can't be parsed.
func Read(cardPath string) Status {
	data, err := os.ReadFile(filepath.Join(cardPath, fileName))
	if err != nil {
		return Status{}
	}

	var df dotfileSchema
	if err := json.Unmarshal(data, &df); err != nil {
		return Status{}
	}

	t, err := time.Parse(time.RFC3339, df.LastCopied)
	if err != nil {
		return Status{}
	}

	return Status{
		Copied:     true,
		CopiedAt:   t,
		CopiedDest: df.Destination,
	}
}

// WriteOptions configures what gets written to the dotfile.
type WriteOptions struct {
	CardPath       string
	Destination    string
	Mode           string // "all", "photos", "videos", "selects"
	FilesCopied    int
	BytesCopied    int64
	Verified       bool
	CardbotVersion string
}

// Write creates or overwrites the .cardbot dotfile on the card.
// Uses atomic write (temp file + rename) to avoid corruption.
// Returns nil if writing fails (e.g. read-only card) — callers should warn but not fail.
func Write(opts WriteOptions) error {
	df := dotfileSchema{
		Schema:         "cardbot-dotfile-v1",
		LastCopied:     time.Now().Format(time.RFC3339),
		Mode:           opts.Mode,
		Destination:    opts.Destination,
		FilesCopied:    opts.FilesCopied,
		BytesCopied:    opts.BytesCopied,
		Verified:       opts.Verified,
		CardbotVersion: opts.CardbotVersion,
	}

	data, err := json.MarshalIndent(df, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	target := filepath.Join(opts.CardPath, fileName)
	tmp := target + ".tmp"

	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmp, target)
}

// FormatStatus returns a display string for the card status.
//
//	"New"
//	"Copied on 2026-03-08 15:04"
func FormatStatus(s Status) string {
	if !s.Copied {
		return "New"
	}
	return "Copied on " + s.CopiedAt.Format("2006-01-02 15:04")
}
