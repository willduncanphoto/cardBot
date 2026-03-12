package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/illwill/cardbot/internal/analyze"
	"github.com/illwill/cardbot/internal/config"
	"github.com/illwill/cardbot/internal/detect"
	cblog "github.com/illwill/cardbot/internal/log"
	"github.com/illwill/cardbot/internal/pick"
	"github.com/illwill/cardbot/internal/speedtest"
)

const version = "0.1.4"

// UX delays — remove in 0.4.0 when real startup and analysis timings replace them.
const (
	removalDelay = 2 * time.Second // Pause after card removal so message is visible
)

// ts returns the current timestamp formatted for log output.
func ts() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

type app struct {
	detector    *detect.Detector
	currentCard *detect.Card
	cardQueue   []*detect.Card
	mu          sync.Mutex
	cfg         *config.Config
	logger      *cblog.Logger
	dryRun      bool
}

// logf writes to the log file if logging is enabled, and is a no-op otherwise.
func (a *app) logf(format string, args ...any) {
	if a.logger != nil {
		a.logger.Printf(format, args...)
	}
}

// printf prints to stdout and mirrors to the log file.
func (a *app) printf(format string, args ...any) {
	fmt.Printf(format, args...)
	a.logf(strings.TrimRight(fmt.Sprintf(format, args...), "\n"))
}

func main() {
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
			fmt.Fprintf(os.Stderr, "Warning: %v — using defaults\n", err)
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
				fmt.Fprintf(os.Stderr, "Warning: could not save config: %v\n", saveErr)
			}
		}
		fmt.Println()
	} else if *flagDest == "" {
		// Destination already configured — confirm before scanning.
		newPath := config.ContractPath(confirmDestination(cfg.Destination.Path))
		if newPath != cfg.Destination.Path {
			cfg.Destination.Path = newPath
			if cfgPath != "" {
				if saveErr := config.Save(cfg, cfgPath); saveErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not save config: %v\n", saveErr)
				}
			}
		}
		fmt.Println()
	}

	// --- Set up logger ---
	var logger *cblog.Logger
	if cfg.Advanced.LogFile != "" {
		logPath, expandErr := config.ExpandPath(cfg.Advanced.LogFile)
		if expandErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not expand log path: %v\n", expandErr)
		} else {
			logger, err = cblog.Open(logPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not open log file: %v\n", err)
			} else {
				defer logger.Close()
			}
		}
	}

	// --- Build app ---
	a := &app{
		cardQueue: make([]*detect.Card, 0),
		cfg:       cfg,
		logger:    logger,
		dryRun:    *flagDryRun,
	}

	// Print any config warnings now that logging is ready.
	for _, w := range cfgWarnings {
		a.printf("[%s] Warning: %s\n", ts(), w)
	}

	a.printf("[%s] Starting CardBot %s...\n", ts(), version)
	a.printf("[%s] Copy location is set to %s\n", ts(), cfg.Destination.Path)

	if a.dryRun {
		a.printf("[%s] Dry-run mode — no files will be copied\n", ts())
	}

	a.printf("[%s] Scanning for memory cards...", ts())

	// --- Signal handling ---
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	a.detector = detect.NewDetector()
	if err := a.detector.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer a.detector.Stop()

	inputChan := make(chan string, 10)
	go readInput(inputChan)

	for {
		select {
		case card := <-a.detector.Events():
			a.handleCardEvent(card)

		case path := <-a.detector.Removals():
			a.handleRemoval(path)

		case input := <-inputChan:
			a.handleInput(input)

		case <-sigChan:
			fmt.Println("\nShutting down...")
			a.logf("Shutting down")
			return
		}
	}
}

// confirmDestination shows the saved destination and lets the user confirm or change it.
func confirmDestination(savedPath string) string {
	expanded, err := config.ExpandPath(savedPath)
	if err != nil {
		expanded = savedPath
	}

	fmt.Printf("Destination: %s\n", expanded)
	fmt.Print("[Enter] Continue  [c] Change  > ")

	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))

	if line == "c" {
		return promptDestination(savedPath)
	}
	return savedPath
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

func (a *app) handleCardEvent(card *detect.Card) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Ignore if already being processed or queued
	if a.isTracked(card.Path) {
		return
	}

	if a.currentCard == nil {
		a.currentCard = card
		fmt.Println("card found.")
		a.logf("Card detected: %s", card.Path)
		go func() {
			time.Sleep(500 * time.Millisecond)
			a.displayCard(card)
		}()
	} else {
		a.cardQueue = append(a.cardQueue, card)
		a.printQueueNotice(card)
	}
}

func (a *app) isTracked(path string) bool {
	if a.currentCard != nil && a.currentCard.Path == path {
		return true
	}
	for _, c := range a.cardQueue {
		if c.Path == path {
			return true
		}
	}
	return false
}

func (a *app) printQueueNotice(card *detect.Card) {
	plural := ""
	if len(a.cardQueue) > 1 {
		plural = "s"
	}
	fmt.Printf("\n[%s] %s detected (%d card%s in queue)\n",
		ts(),
		card.Brand,
		len(a.cardQueue),
		plural)
	a.logf("%s detected (%d card%s in queue)", card.Brand, len(a.cardQueue), plural)
}

func (a *app) displayCard(card *detect.Card) {
	if !a.isCurrentCard(card.Path) {
		return // Card was cancelled or removed while starting
	}

	fmt.Printf("[%s] Scanning %s... ", ts(), card.Path)
	a.logf("Scanning %s", card.Path)
	scanStart := time.Now()
	analyzer := analyze.New(card.Path)
	analyzer.OnProgress(func(count int) {
		if count%100 == 0 {
			fmt.Printf("\r[%s] Scanning %s... %d files", ts(), card.Path, count)
		}
	})

	result, err := analyzer.Analyze()
	if err != nil {
		fmt.Printf("\nError analyzing card: %v\n", err)
		a.logf("Error analyzing card %s: %v", card.Path, err)
		a.finishCard()
		return
	}

	elapsed := time.Since(scanStart)
	secs := int(elapsed.Round(time.Second).Seconds())

	total := 0
	if result != nil {
		total = result.FileCount
	}
	fmt.Printf("\r[%s] Scanning %s... %d files ✓\n", ts(), card.Path, total)
	secWord := "seconds"
	if secs == 1 {
		secWord = "second"
	}
	fmt.Printf("[%s] Scan completed in %d %s\n", ts(), secs, secWord)
	a.logf("Scan completed: %s — %d files in %d %s", card.Path, total, secs, secWord)
	fmt.Println()
	a.printCardInfo(card, result)
}

func (a *app) isCurrentCard(path string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.currentCard != nil && a.currentCard.Path == path
}

func (a *app) printCardInfo(card *detect.Card, result *analyze.Result) {
	fmt.Printf("  Path:     %s\n", card.Path)
	var pct int64
	if card.TotalBytes > 0 {
		pct = (card.UsedBytes * 100) / card.TotalBytes
	}
	fmt.Printf("  Storage:  %s / %s (%d%%)\n",
		detect.FormatBytes(card.UsedBytes),
		detect.FormatBytes(card.TotalBytes),
		pct)
	fmt.Printf("  Brand:    %s\n", card.Brand)
	if result != nil && result.Gear != "" {
		fmt.Printf("  Camera:   %s\n", result.Gear)
	}

	if result != nil && result.Starred > 0 {
		fmt.Printf("  Starred:  %d\n", result.Starred)
	}

	if result != nil && result.FileCount > 0 {
		// Find max file count width for alignment.
		maxCount := 0
		for _, g := range result.Groups {
			if g.FileCount > maxCount {
				maxCount = g.FileCount
			}
		}
		countWidth := len(fmt.Sprintf("%d", maxCount))

		for i, g := range result.Groups {
			if i == 0 {
				fmt.Printf("  Content:  ")
			} else {
				fmt.Printf("            ")
			}
			fmt.Printf("%s   %10s   %*d   %s\n",
				g.Date,
				detect.FormatBytes(g.Size),
				countWidth,
				g.FileCount,
				strings.Join(g.Extensions, ", "))
		}
		fmt.Println()
		fmt.Printf("  Total:    %d photos, %d videos, %s\n", result.PhotoCount, result.VideoCount, detect.FormatBytes(result.TotalSize))
	} else {
		fmt.Println("  Content:  (empty)")
	}

	if a.dryRun {
		fmt.Printf("  Dest:     %s (dry-run)\n", a.cfg.Destination.Path)
	}

	a.mu.Lock()
	queueLen := len(a.cardQueue)
	a.mu.Unlock()

	if queueLen > 0 {
		plural := ""
		if queueLen > 1 {
			plural = "s"
		}
		fmt.Printf("  Queue:    %d card%s waiting\n", queueLen, plural)
	}

	fmt.Println("────────────────────────────────────────")
	fmt.Print("[e] Eject  [c] Cancel  > ")
}

func (a *app) finishCard() {
	a.mu.Lock()
	a.currentCard = nil

	if len(a.cardQueue) > 0 {
		nextCard := a.cardQueue[0]
		a.cardQueue = a.cardQueue[1:]
		a.currentCard = nextCard
		a.mu.Unlock()
		go a.displayCard(nextCard)
		return
	}
	a.mu.Unlock()

	fmt.Printf("\n[%s] Scanning for memory cards...", ts())
}

func (a *app) handleRemoval(path string) {
	a.mu.Lock()
	wasCurrent := a.currentCard != nil && a.currentCard.Path == path

	if wasCurrent {
		a.currentCard = nil
		hasQueue := len(a.cardQueue) > 0
		var nextCard *detect.Card
		if hasQueue {
			nextCard = a.cardQueue[0]
			a.cardQueue = a.cardQueue[1:]
			a.currentCard = nextCard
		}
		a.mu.Unlock()

		fmt.Printf("\n[%s] Card removed: %s\n", ts(), path)
		a.logf("Card removed: %s", path)
		if hasQueue {
			go a.displayCard(nextCard)
		} else {
			time.Sleep(removalDelay)
			fmt.Printf("\n[%s] Scanning for memory cards...", ts())
		}
		return
	}

	// Check queue
	for i, card := range a.cardQueue {
		if card.Path == path {
			a.cardQueue = append(a.cardQueue[:i], a.cardQueue[i+1:]...)
			a.mu.Unlock()
			fmt.Printf("\n[%s] Queued card removed: %s\n", ts(), path)
			a.logf("Queued card removed: %s", path)
			return
		}
	}
	a.mu.Unlock()
}

func (a *app) handleInput(input string) {
	a.mu.Lock()
	card := a.currentCard
	a.mu.Unlock()

	if card == nil {
		return
	}

	switch strings.ToLower(input) {
	case "e":
		a.ejectCard(card)
	case "c":
		a.cancelCard()
	case "i":
		a.showHardwareInfo(card)
	case "t":
		a.runSpeedTest(card)
	}
}

func (a *app) ejectCard(card *detect.Card) {
	fmt.Printf("\nEjecting %s...\n", card.Name)
	a.logf("Ejecting %s", card.Path)
	if err := a.detector.Eject(card.Path); err != nil {
		fmt.Printf("Error: %v\n", err)
		a.logf("Eject error: %v", err)
		return
	}
	a.detector.Remove(card.Path)
	fmt.Printf("\n[%s] Card ejected: %s\n", ts(), card.Path)
	a.logf("Card ejected: %s", card.Path)
	a.finishCard()
}

func (a *app) cancelCard() {
	fmt.Println("\nCancelled.")
	a.logf("Card cancelled")
	a.finishCard()
}

func (a *app) showHardwareInfo(card *detect.Card) {
	fmt.Println()
	if card.Hardware == nil {
		fmt.Println("Hardware info unavailable")
		return
	}
	fmt.Println(detect.FormatHardwareInfo(card.Hardware))
	fmt.Println()
	a.printPrompt()
}

func (a *app) runSpeedTest(card *detect.Card) {
	fmt.Println()
	fmt.Printf("[%s] Speed test starting (256 MB)...\n", ts())
	a.logf("Speed test starting on %s", card.Path)

	result, err := speedtest.Run(card.Path, func(phase string, mbps float64) {
		fmt.Printf("\r[%s] %s... %.1f MB/s    ", ts(), phase, mbps)
	})
	fmt.Println()

	if err != nil {
		fmt.Printf("Speed test failed: %v\n", err)
		a.logf("Speed test failed: %v", err)
	} else {
		fmt.Println()
		fmt.Printf("  Write:  %.1f MB/s\n", result.WriteSpeed)
		fmt.Printf("  Read:   %.1f MB/s\n", result.ReadSpeed)
		a.logf("Speed test complete — write: %.1f MB/s, read: %.1f MB/s", result.WriteSpeed, result.ReadSpeed)
		fmt.Println()
	}

	a.printPrompt()
}

func (a *app) printPrompt() {
	fmt.Print("[e] Eject  [c] Cancel  > ")
}

func readInput(ch chan<- string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		ch <- strings.TrimSpace(line)
	}
}
