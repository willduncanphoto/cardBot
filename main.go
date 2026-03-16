package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/illwill/cardbot/internal/app"
	"github.com/illwill/cardbot/internal/config"
	cblog "github.com/illwill/cardbot/internal/log"
	"github.com/illwill/cardbot/internal/pick"
)

const version = "0.4.2"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "self-update" {
		os.Exit(app.RunSelfUpdate(version))
	}

	// --- CLI flags ---
	var (
		flagVersion = flag.Bool("version", false, "print version and exit")
		flagDest    = flag.String("dest", "", "destination path for copied cards")
		flagDryRun  = flag.Bool("dry-run", false, "scan cards but do not copy files")
		flagReset   = flag.Bool("reset", false, "clear saved config and exit")
		flagSetup   = flag.Bool("setup", false, "re-run first-time setup (destination and naming)")
	)
	flag.Parse()

	if *flagVersion {
		fmt.Printf("cardbot %s\n", version)
		os.Exit(0)
	}

	if *flagReset {
		cfgPath, err := config.Path()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not determine config path: %v\n", err)
			os.Exit(1)
		}
		if err := os.Remove(cfgPath); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: could not remove config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Config cleared. Please restart CardBot.")
		os.Exit(0)
	}

	// --- Load config ---
	cfgPath, err := config.Path()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not determine config path: %v\n", err)
		cfgPath = ""
	}

	var cfg *config.Config
	var cfgWarnings []string

	if cfgPath != "" {
		cfg, cfgWarnings, err = config.Load(cfgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %s — using defaults\n", errMsg(err))
			cfg = config.Defaults()
		}
	} else {
		cfg = config.Defaults()
	}

	// --- CLI flags override config ---
	if *flagDest != "" {
		cfg.Destination.Path = *flagDest
	}

	// --- First-run or --setup: prompt for destination, then continue into the app ---
	needsSetup := *flagSetup
	if cfgPath != "" {
		if _, statErr := os.Stat(cfgPath); os.IsNotExist(statErr) {
			needsSetup = true
		}
	}
	if needsSetup {
		if saveErr := app.RunSetup(cfg, cfgPath, promptDestination, app.PromptNamingMode); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save config: %s\n", errMsg(saveErr))
		}
		fmt.Println()
	}

	// --- Set up logger ---
	var logger *cblog.Logger
	if cfg.Advanced.LogFile != "" {
		logPath, expandErr := config.ExpandPath(cfg.Advanced.LogFile)
		if expandErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not expand log path: %s\n", errMsg(expandErr))
		} else {
			logger, err = cblog.Open(logPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not open log file: %s\n", errMsg(err))
			} else {
				defer logger.Close()
			}
		}
	}

	// --- Build app ---
	a := app.New(app.Config{
		Cfg:     cfg,
		Logger:  logger,
		DryRun:  *flagDryRun,
		Version: version,
	})

	// Print any config warnings now that logging is ready.
	for _, w := range cfgWarnings {
		a.Printf("[%s] Warning: %s\n", app.Ts(), w)
	}

	a.Printf("[%s] CardBot %s\n", app.Ts(), version)

	// Startup update-check with spinner, then update the same line with result.
	ts := app.Ts()
	prefix := fmt.Sprintf("[%s] ", ts)
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Prefix = prefix + "Checking for updates "
	s.Start()
	latest, updateErr := app.MaybeCheckForUpdate(logger, version)
	time.Sleep(500 * time.Millisecond) // ensure user sees activity
	s.Stop()

	// Update the same line with the result.
	clearEOL := "\033[K"
	var msg string
	if updateErr != nil {
		msg = "NO SIGNAL"
	} else if latest != "" {
		msg = fmt.Sprintf("UPDATE AVAILABLE (%s)", latest)
	} else {
		msg = "CardBot is up to date"
	}
	fmt.Printf("\r%s%s%s\n", prefix, msg, clearEOL)

	// Print action line if update available.
	if latest != "" && updateErr == nil {
		fmt.Printf("%sRun: cardbot self-update\n", prefix)
	}

	if *flagDryRun {
		a.Printf("[%s] Dry-run mode — no files will be copied\n", app.Ts())
	}

	a.StartScanning()

	if err := a.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// promptDestination asks the user to pick a destination path.
// On macOS, opens the native folder picker. Falls back to readline on Linux.
func promptDestination(defaultPath string) string {
	fmt.Println("Welcome to CardBot!")
	fmt.Println()
	fmt.Println("Where should CardBot copy your work?")
	fmt.Println()

	expanded, err := config.ExpandPath(defaultPath)
	if err != nil {
		expanded = defaultPath
	}

	picked, err := pick.Folder(expanded)
	if err == nil && picked != "" {
		fmt.Printf("Destination: %s\n", picked)
		return picked
	}

	// Fallback: readline with tab completion.
	return promptDestinationReadline(expanded)
}

// promptDestinationReadline is the fallback path prompt using stdlib.
func promptDestinationReadline(defaultPath string) string {
	fmt.Printf("Destination [%s]: ", defaultPath)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultPath
	}
	return line
}

// errMsg returns a short, user-facing message for common OS-level errors.
func errMsg(err error) string {
	s := err.Error()
	switch {
	case strings.Contains(s, "no space left"):
		return "destination disk is full"
	case strings.Contains(s, "permission denied"):
		return "permission denied — check folder permissions"
	case strings.Contains(s, "read-only file system"):
		return "destination is read-only"
	case strings.Contains(s, "input/output error"):
		return "I/O error — card may be damaged"
	default:
		return s
	}
}
