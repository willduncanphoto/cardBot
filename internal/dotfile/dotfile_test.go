package dotfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteRead_RoundTrip(t *testing.T) {
	t.Parallel()
	card := t.TempDir()

	err := Write(WriteOptions{
		CardPath:       card,
		Destination:    "/Users/test/Pictures/CardBot",
		Mode:           "all",
		FilesCopied:    150,
		BytesCopied:    5000000,
		Verified:       true,
		CardbotVersion: "0.1.5",
	})
	if err != nil {
		t.Fatal(err)
	}

	status := Read(card)
	if !status.Copied {
		t.Error("expected Copied=true")
	}
	if status.CopiedAt.IsZero() {
		t.Error("expected CopiedAt to be set")
	}
	if status.CopiedDest != "/Users/test/Pictures/CardBot" {
		t.Errorf("CopiedDest = %q, want /Users/test/Pictures/CardBot", status.CopiedDest)
	}
}

func TestRead_MissingFile(t *testing.T) {
	t.Parallel()
	status := Read(t.TempDir())
	if status.Copied {
		t.Error("expected Copied=false for missing dotfile")
	}
}

func TestRead_MalformedJSON(t *testing.T) {
	t.Parallel()
	card := t.TempDir()
	os.WriteFile(filepath.Join(card, ".cardbot"), []byte("{bad json"), 0644)

	status := Read(card)
	if status.Copied {
		t.Error("expected Copied=false for malformed JSON")
	}
}

func TestRead_MissingTimestamp(t *testing.T) {
	t.Parallel()
	card := t.TempDir()
	os.WriteFile(filepath.Join(card, ".cardbot"), []byte(`{"$schema":"cardbot-dotfile-v1"}`), 0644)

	status := Read(card)
	if status.Copied {
		t.Error("expected Copied=false when last_copied is missing")
	}
}

func TestWrite_AtomicRename(t *testing.T) {
	t.Parallel()
	card := t.TempDir()

	err := Write(WriteOptions{
		CardPath:       card,
		Destination:    "/dest",
		Mode:           "all",
		FilesCopied:    1,
		BytesCopied:    100,
		CardbotVersion: "0.1.5",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Temp file should not remain.
	tmp := filepath.Join(card, ".cardbot.tmp")
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Error(".cardbot.tmp should not exist after successful write")
	}

	// Final file should be valid JSON.
	data, err := os.ReadFile(filepath.Join(card, ".cardbot"))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Errorf("dotfile is not valid JSON: %v", err)
	}
}

func TestWrite_Schema(t *testing.T) {
	t.Parallel()
	card := t.TempDir()

	Write(WriteOptions{CardPath: card, Destination: "/dest", Mode: "all", CardbotVersion: "0.1.5"})

	data, _ := os.ReadFile(filepath.Join(card, ".cardbot"))
	var df dotfileSchema
	json.Unmarshal(data, &df)

	if df.Schema != "cardbot-dotfile-v1" {
		t.Errorf("Schema = %q, want cardbot-dotfile-v1", df.Schema)
	}
	if df.Mode != "all" {
		t.Errorf("Mode = %q, want all", df.Mode)
	}
}

func TestWrite_Overwrite(t *testing.T) {
	t.Parallel()
	card := t.TempDir()

	// First write.
	Write(WriteOptions{CardPath: card, Destination: "/first", Mode: "all", FilesCopied: 1, CardbotVersion: "0.1.5"})
	// Second write should overwrite.
	Write(WriteOptions{CardPath: card, Destination: "/second", Mode: "all", FilesCopied: 99, CardbotVersion: "0.1.5"})

	status := Read(card)
	if status.CopiedDest != "/second" {
		t.Errorf("CopiedDest = %q, want /second (should overwrite)", status.CopiedDest)
	}
}

func TestFormatStatus(t *testing.T) {
	t.Parallel()
	t.Run("new", func(t *testing.T) {
		s := FormatStatus(Status{})
		if s != "New" {
			t.Errorf("FormatStatus(empty) = %q, want New", s)
		}
	})

	t.Run("copied", func(t *testing.T) {
		card := t.TempDir()
		Write(WriteOptions{CardPath: card, Destination: "/dest", Mode: "all", CardbotVersion: "0.1.5"})
		status := Read(card)

		s := FormatStatus(status)
		if s == "New" {
			t.Error("expected 'Copied on ...' not 'New'")
		}
		if len(s) < 20 {
			t.Errorf("FormatStatus too short: %q", s)
		}
	})
}
