package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/illwill/cardbot/internal/config"
	"github.com/illwill/cardbot/internal/detect"
	cblog "github.com/illwill/cardbot/internal/log"
	"github.com/illwill/cardbot/internal/pick"
)

const version = "0.2.9"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "self-update" {
		os.Exit(runSelfUpdate())
	}

	// --- CLI flags ---
	var (
		flagVersion = flag.Bool("version", false, "print version and exit")
		flagDest    = flag.String("dest", "", "destination path for copied cards")
		flagDryRun  = flag.Bool("dry-run", false, "scan cards but do not copy files")
		flagReset   = flag.Bool("reset", false, "clear saved config and exit")
		flagSetup   = flag.Bool("setup", false, "re-run destination setup")
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
			fmt.Fprintf(os.Stderr, "Warning: %s — using defaults\n", friendlyErr(err))
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
		cfg.Destination.Path = config.ContractPath(promptDestination(cfg.Destination.Path))
		if cfgPath != "" {
			if saveErr := config.Save(cfg, cfgPath); saveErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not save config: %s\n", friendlyErr(saveErr))
			}
		}
		fmt.Println()
	}

	// --- Set up logger ---
	var logger *cblog.Logger
	if cfg.Advanced.LogFile != "" {
		logPath, expandErr := config.ExpandPath(cfg.Advanced.LogFile)
		if expandErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not expand log path: %s\n", friendlyErr(expandErr))
		} else {
			logger, err = cblog.Open(logPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not open log file: %s\n", friendlyErr(err))
			} else {
				defer logger.Close()
			}
		}
	}

	// --- Signal handling ---
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	inputDone := make(chan struct{})

	// --- Build app ---
	inputChan := make(chan string, 10)
	a := &app{
		cardQueue:   make([]*detect.Card, 0),
		cfg:         cfg,
		logger:      logger,
		inputChan:   inputChan,
		sigChan:     sigChan,
		inputDone:   inputDone,
		dryRun:      *flagDryRun,
		copiedModes: make(map[string]bool),
	}

	// Print any config warnings now that logging is ready.
	for _, w := range cfgWarnings {
		a.printf("[%s] Warning: %s\n", ts(), w)
	}

	// Print version, animate dots over ~1 second, then continue.
	fmt.Printf("[%s] Starting CardBot %s", ts(), version)
	if a.logger != nil {
		a.logger.Raw(fmt.Sprintf("[%s] Starting CardBot %s...", ts(), version))
	}
	for i := 0; i < 3; i++ {
		time.Sleep(300 * time.Millisecond)
		fmt.Print(".")
	}
	fmt.Println()

	a.printf("[%s] Copy path %s\n", ts(), config.ContractPath(cfg.Destination.Path))
	a.printf("[%s] Keep original filenames\n", ts())

	if latest, ok := maybeCheckForUpdate(cfg, cfgPath, logger); ok {
		a.printf("[%s] Update available: %s (you have %s)\n", ts(), latest, version)
		a.printf("[%s] Run: cardbot self-update\n", ts())
	}

	if a.dryRun {
		a.printf("[%s] Dry-run mode — no files will be copied\n", ts())
	}

	a.printf("[%s] Scanning  ", ts())
	a.startSpinner()

	a.detector = detect.NewDetector()
	if err := a.detector.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer a.detector.Stop()

	go readInput(a.inputChan, a.inputDone)

	for {
		select {
		case card := <-a.detector.Events():
			a.handleCardEvent(card)

		case path := <-a.detector.Removals():
			a.handleRemoval(path)

		case input := <-a.inputChan:
			a.handleInput(input)

		case <-a.sigChan:
			fmt.Println("\nShutting down...")
			a.logf("Shutting down")
			close(a.inputDone)
			return
		}
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
