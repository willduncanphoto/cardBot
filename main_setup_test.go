package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/illwill/cardbot/internal/app"
	"github.com/illwill/cardbot/internal/config"
)

func TestPromptDestinationReadlineIO_UsesProvidedReader(t *testing.T) {
	t.Parallel()

	in := bufio.NewReader(strings.NewReader("~/Pictures/Ingest\n"))
	var out bytes.Buffer

	got := promptDestinationReadlineIO("~/Pictures/CardBot", in, &out)
	if got != "~/Pictures/Ingest" {
		t.Fatalf("promptDestinationReadlineIO() = %q, want %q", got, "~/Pictures/Ingest")
	}
	if !strings.Contains(out.String(), "Destination [~/Pictures/CardBot]:") {
		t.Fatalf("missing destination prompt, got:\n%s", out.String())
	}
}

func TestSetupInput_SequentialAcrossDestinationAndPrompts(t *testing.T) {
	t.Parallel()

	in := bufio.NewReader(strings.NewReader("~/Pictures/Ingest\n2\ny\ny\n3\n~/Code\n"))
	var out bytes.Buffer

	dest := promptDestinationReadlineIO("~/Pictures/CardBot", in, &out)
	if dest != "~/Pictures/Ingest" {
		t.Fatalf("destination = %q, want %q", dest, "~/Pictures/Ingest")
	}

	prompter := app.NewSetupPrompter(in, &out)
	if mode := prompter.PromptNamingMode(config.NamingOriginal); mode != config.NamingTimestamp {
		t.Fatalf("PromptNamingMode = %q, want %q", mode, config.NamingTimestamp)
	}
	if enabled := prompter.PromptDaemonEnabled(false); !enabled {
		t.Fatal("PromptDaemonEnabled = false, want true")
	}
	if startAtLogin := prompter.PromptDaemonStartAtLogin(false); !startAtLogin {
		t.Fatal("PromptDaemonStartAtLogin = false, want true")
	}
	if appName := prompter.PromptDaemonTerminalApp("Terminal"); appName != "Ghostty" {
		t.Fatalf("PromptDaemonTerminalApp = %q, want %q", appName, "Ghostty")
	}
	if dir := prompter.PromptDaemonWorkingDirectory("~"); dir != "~/Code" {
		t.Fatalf("PromptDaemonWorkingDirectory = %q, want %q", dir, "~/Code")
	}
}
