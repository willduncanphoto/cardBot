package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/illwill/cardbot/internal/config"
)

// promptNamingMode asks the user how filenames should be written on copy.
func promptNamingMode(defaultMode string) string {
	return promptNamingModeIO(os.Stdin, os.Stdout, defaultMode)
}

func promptNamingModeIO(in io.Reader, out io.Writer, defaultMode string) string {
	mode := config.NormalizeNamingMode(defaultMode)
	reader := bufio.NewReader(in)

	for {
		fmt.Fprintln(out, "────────────────────────────────────────")
		fmt.Fprintln(out, "File Naming")
		fmt.Fprintln(out, "────────────────────────────────────────")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Camera filenames reset every 10,000 shots.")
		fmt.Fprintln(out, "This can cause duplicates when copying multiple cards.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "[1] Keep camera filenames")
		fmt.Fprintln(out, "    DSC_0001.NEF, DSC_0002.NEF ...")
		fmt.Fprintln(out, "    Use if you rely on camera numbering.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "[2] Timestamp + sequence")
		fmt.Fprintln(out, "    260314T143052_001.NEF, _002.NEF ...")
		fmt.Fprintln(out, "    Use for automatic order across all cards.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "You can change this later with cardbot --setup.")
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Choice [%s]: ", namingChoiceDefault(mode))

		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(out)
			return mode
		}
		line = strings.TrimSpace(line)
		if line == "" {
			fmt.Fprintf(out, "Naming set to: %s\n", namingModeLabel(mode))
			return mode
		}

		if chosen, ok := parseNamingChoice(line); ok {
			fmt.Fprintf(out, "Naming set to: %s\n", namingModeLabel(chosen))
			return chosen
		}

		fmt.Fprintln(out, "Please enter 1 or 2.")
		fmt.Fprintln(out)
	}
}

func parseNamingChoice(input string) (string, bool) {
	switch strings.TrimSpace(strings.ToLower(input)) {
	case "1", "o", "original":
		return config.NamingOriginal, true
	case "2", "t", "timestamp":
		return config.NamingTimestamp, true
	default:
		return "", false
	}
}

func namingChoiceDefault(mode string) string {
	if config.NormalizeNamingMode(mode) == config.NamingTimestamp {
		return "2"
	}
	return "1"
}

func namingModeLabel(mode string) string {
	if config.NormalizeNamingMode(mode) == config.NamingTimestamp {
		return "Timestamp + sequence"
	}
	return "Camera original"
}

func namingStartupLine(mode string) string {
	if config.NormalizeNamingMode(mode) == config.NamingTimestamp {
		return "Naming: Timestamp + sequence (001-999)"
	}
	return "Naming: Camera original (DSC_xxxx.NEF)"
}

func namingDisplayLine(mode string, fileCount int) string {
	_ = fileCount // reserved for future per-date digit detection
	if config.NormalizeNamingMode(mode) != config.NamingTimestamp {
		return "Camera original (DSC_0001.NEF)"
	}
	return "Timestamp + sequence (001-999)"
}
