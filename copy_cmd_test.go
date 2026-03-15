package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	cardcopy "github.com/illwill/cardbot/internal/copy"
	"github.com/illwill/cardbot/internal/detect"
	"github.com/illwill/cardbot/internal/dotfile"
)

func TestHandleCopySuccess_DryRun_NoSideEffects(t *testing.T) {
	t.Parallel()

	cardPath := t.TempDir()
	a := &app{copiedModes: make(map[string]bool)}
	card := &detect.Card{Path: cardPath}

	a.handleCopySuccess(card, "all", t.TempDir(), &cardcopy.Result{
		FilesCopied: 42,
		BytesCopied: 1024,
		Elapsed:     2 * time.Second,
	}, true, 10)

	if a.copiedModes["all"] {
		t.Fatal("dry-run must not mark mode as copied")
	}

	dotPath := filepath.Join(cardPath, ".cardbot")
	if _, err := os.Stat(dotPath); !os.IsNotExist(err) {
		t.Fatalf("dry-run must not write dotfile, got err=%v", err)
	}
}

func TestHandleCopySuccess_RealCopy_WritesDotfileAndMarksMode(t *testing.T) {
	t.Parallel()

	cardPath := t.TempDir()
	a := &app{copiedModes: make(map[string]bool)}
	card := &detect.Card{Path: cardPath}
	dest := t.TempDir()

	a.handleCopySuccess(card, "all", dest, &cardcopy.Result{
		FilesCopied: 7,
		BytesCopied: 2048,
		Elapsed:     3 * time.Second,
	}, false, 0)

	if !a.copiedModes["all"] {
		t.Fatal("real copy should mark mode as copied")
	}

	status := dotfile.Read(cardPath)
	if !status.Copied {
		t.Fatal("expected dotfile copied status")
	}
	if len(status.Entries) == 0 {
		t.Fatal("expected at least one dotfile entry")
	}
	if status.Entries[0].Mode != "all" {
		t.Fatalf("dotfile mode = %q, want %q", status.Entries[0].Mode, "all")
	}
	if status.Entries[0].Destination != dest {
		t.Fatalf("dotfile destination = %q, want %q", status.Entries[0].Destination, dest)
	}
}
