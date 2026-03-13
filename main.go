package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/illwill/cardbot/internal/analyze"
	"github.com/illwill/cardbot/internal/config"
	cardcopy "github.com/illwill/cardbot/internal/copy"
	"github.com/illwill/cardbot/internal/detect"
	"github.com/illwill/cardbot/internal/dotfile"
	cblog "github.com/illwill/cardbot/internal/log"
	"github.com/illwill/cardbot/internal/pick"
	"github.com/illwill/cardbot/internal/speedtest"
	"github.com/illwill/cardbot/internal/ui"
)

const version = "0.1.7"

// UX delays — gives the user time to read each startup line before the next appears.
const (
	removalDelay = 2 * time.Second // Pause after card removal so message is visible
)

// ts returns the current timestamp formatted for log output.
func ts() string {
	return time.Now().Format("2006-01-02T15:04:05")
}

type app struct {
	detector    *detect.Detector
	currentCard *detect.Card
	lastResult  *analyze.Result // analysis result for currentCard
	cardQueue   []*detect.Card
	mu          sync.Mutex
	printMu     sync.Mutex // serialises concurrent stdout writes during copy
	cfg         *config.Config
	logger      *cblog.Logger
	inputChan   chan string    // buffered input from stdin
	sigChan     chan os.Signal // SIGINT/SIGTERM
	inputDone   chan struct{}  // closed on shutdown to stop readInput
	dryRun      bool
	copied      bool          // true after successful copy of current card
	cardInvalid bool          // true when current card has no DCIM directory
	spinStop    chan struct{} // signals spinner goroutine to stop
	spinDone    chan struct{} // closed when spinner goroutine exits
}

// drainInput discards any buffered input keystrokes.
// Called after blocking operations (copy, speed test) to prevent
// queued commands from firing on the next prompt.
func (a *app) drainInput() {
	for {
		select {
		case <-a.inputChan:
		default:
			return
		}
	}
}

// spinnerFrames are the braille animation frames for the scanning spinner.
// spinnerFrames are the classic spinner animation frames.
var spinnerFrames = []string{"|", "/", "-", "\\"}

// startSpinner starts the background spinner animation on the current line.
func (a *app) startSpinner() {
	a.stopSpinner() // ensure no stale spinner
	a.spinStop = make(chan struct{})
	a.spinDone = make(chan struct{})
	go func() {
		defer close(a.spinDone)
		i := 0
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-a.spinStop:
				fmt.Print("\b \b")
				return
			case <-ticker.C:
				fmt.Printf("\b%s", spinnerFrames[i%len(spinnerFrames)])
				i++
			}
		}
	}()
}

// stopSpinner stops the background spinner and waits for it to finish.
func (a *app) stopSpinner() {
	if a.spinStop != nil {
		close(a.spinStop)
		<-a.spinDone // wait for goroutine to clean up
		a.spinStop = nil
		a.spinDone = nil
	}
}

// logf writes to the log file if logging is enabled, and is a no-op otherwise.
func (a *app) logf(format string, args ...any) {
	if a.logger != nil {
		a.logger.Printf(format, args...)
	}
}

// printf prints to stdout and mirrors to the log file (without adding a second timestamp).
func (a *app) printf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Print(msg)
	if a.logger != nil {
		// Caller already includes [timestamp] in the message, so write raw.
		a.logger.Raw(strings.TrimRight(msg, "\n"))
	}
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

	// --- Signal handling ---
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	inputDone := make(chan struct{})

	// --- Build app ---
	inputChan := make(chan string, 10)
	a := &app{
		cardQueue: make([]*detect.Card, 0),
		cfg:       cfg,
		logger:    logger,
		inputChan: inputChan,
		sigChan:   sigChan,
		inputDone: inputDone,
		dryRun:    *flagDryRun,
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

func (a *app) handleCardEvent(card *detect.Card) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Ignore if already being processed or queued
	if a.isTracked(card.Path) {
		return
	}

	if a.currentCard == nil {
		a.currentCard = card
		a.stopSpinner()
		fmt.Println() // newline after spinner
		fmt.Printf("[%s] Card detected\n", ts())
		a.logf("Card detected: %s", card.Path)
		cardPath := card.Path // capture by value before releasing lock
		go func() {
			time.Sleep(500 * time.Millisecond)
			a.displayCard(cardPath)
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

func (a *app) displayCard(path string) {
	if !a.isCurrentCard(path) {
		return // Card was cancelled or removed while starting
	}

	fmt.Printf("[%s] Reading %s... ", ts(), path)
	a.logf("Reading %s", path)
	scanStart := time.Now()
	analyzer := analyze.New(path)
	analyzer.SetWorkers(a.cfg.Advanced.ExifWorkers)
	analyzer.OnProgress(func(count int) {
		if count%100 == 0 {
			fmt.Printf("\r[%s] Reading %s... %d files", ts(), path, count)
		}
	})

	result, err := analyzer.Analyze()

	// Re-check: card may have been removed or cancelled during analysis.
	if !a.isCurrentCard(path) {
		return
	}

	if err != nil {
		if os.IsNotExist(err) {
			a.mu.Lock()
			a.cardInvalid = true
			card := a.currentCard
			a.mu.Unlock()
			fmt.Printf("\r[%s] Card is invalid (no DCIM found)\n", ts())
			a.logf("Card invalid: no DCIM at %s", path)
			a.printInvalidCardInfo(card)
		} else {
			fmt.Printf("\r[%s] Error scanning card: %s\n", ts(), friendlyErr(err))
			a.logf("Error analyzing card %s: %v", path, err)
			a.finishCard()
		}
		return
	}

	elapsed := time.Since(scanStart)
	secs := int(elapsed.Round(time.Second).Seconds())

	total := 0
	if result != nil {
		total = result.FileCount
	}
	fmt.Printf("\r[%s] Reading %s... %d files ✓ (%ds)\n", ts(), path, total, secs)
	a.logf("Scan completed: %s — %d files in %ds", path, total, secs)
	fmt.Println()
	a.mu.Lock()
	a.lastResult = result
	card := a.currentCard
	a.mu.Unlock()
	a.printCardInfo(card, result)
}

func (a *app) isCurrentCard(path string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.currentCard != nil && a.currentCard.Path == path
}

func (a *app) printCardInfo(card *detect.Card, result *analyze.Result) {
	status := dotfile.Read(card.Path)
	fmt.Printf("  Status:   %s\n", dotfile.FormatStatus(status))
	fmt.Printf("  Path:     %s\n", card.Path)
	var pct int64
	if card.TotalBytes > 0 {
		pct = (card.UsedBytes * 100) / card.TotalBytes
	}
	fmt.Printf("  Storage:  %s / %s (%d%%)\n",
		detect.FormatBytes(card.UsedBytes),
		detect.FormatBytes(card.TotalBytes),
		pct)
	color, reset := "", ""
	if a.cfg.Output.Color {
		color = ui.BrandColor(card.Brand)
		reset = ui.Reset
	}
	if result != nil && result.Gear != "" {
		fmt.Printf("  Camera:   %s%s%s\n", color, result.Gear, reset)
	} else {
		fmt.Printf("  Camera:   %s%s (unknown model)%s\n", color, card.Brand, reset)
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
	a.printPrompt()
}

func (a *app) finishCard() {
	a.mu.Lock()
	a.currentCard = nil
	a.lastResult = nil
	a.copied = false
	a.cardInvalid = false

	if len(a.cardQueue) > 0 {
		nextCard := a.cardQueue[0]
		a.cardQueue = a.cardQueue[1:]
		a.currentCard = nextCard
		a.mu.Unlock()
		go a.displayCard(nextCard.Path)
		return
	}
	a.mu.Unlock()

	fmt.Printf("\n[%s] Scanning  ", ts())
	a.startSpinner()
}

func (a *app) handleRemoval(path string) {
	a.mu.Lock()
	wasCurrent := a.currentCard != nil && a.currentCard.Path == path

	if wasCurrent {
		a.currentCard = nil
		a.lastResult = nil
		a.copied = false
		a.cardInvalid = false
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
			go a.displayCard(nextCard.Path)
		} else {
			go func() {
				time.Sleep(removalDelay)
				fmt.Printf("\n[%s] Scanning  ", ts())
				a.startSpinner()
			}()
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
	cmd := strings.ToLower(input)

	// Help works regardless of card state.
	if cmd == "?" {
		a.showHelp()
		a.mu.Lock()
		hasCard := a.currentCard != nil
		a.mu.Unlock()
		if hasCard {
			a.printPrompt()
		}
		return
	}

	a.mu.Lock()
	card := a.currentCard
	a.mu.Unlock()

	if card == nil {
		if input != "" {
			fmt.Printf("\nNo card inserted. Waiting for a memory card...\n")
		}
		return
	}

	switch cmd {
	case "a":
		a.mu.Lock()
		alreadyCopied := a.copied
		invalid := a.cardInvalid
		a.mu.Unlock()
		if invalid {
			fmt.Printf("\n[%s] No media found on this card.\n", ts())
			a.printPrompt()
			return
		}
		if alreadyCopied {
			fmt.Printf("\n[%s] Already copied.\n", ts())
			a.printPrompt()
			return
		}
		a.copyAll(card)
	case "e":
		a.ejectCard(card)
	case "x":
		a.cancelCard()
	case "i":
		a.showHardwareInfo(card)
	case "t":
		a.runSpeedTest(card)
	case "s":
		fmt.Println("\nCopy Selects is not yet available.")
		a.printPrompt()
	case "p":
		fmt.Println("\nCopy Photos is not yet available.")
		a.printPrompt()
	case "v":
		fmt.Println("\nCopy Videos is not yet available.")
		a.printPrompt()
	default:
		if input != "" {
			fmt.Printf("\nUnknown command %q. Press [?] for help.\n", input)
			a.printPrompt()
		}
	}
}

func (a *app) ejectCard(card *detect.Card) {
	fmt.Printf("\nEjecting %s...\n", card.Name)
	a.logf("Ejecting %s", card.Path)
	if err := a.detector.Eject(card.Path); err != nil {
		fmt.Printf("Error: %s\n", friendlyErr(err))
		a.logf("Eject error: %v", err)
		a.printPrompt()
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

func (a *app) copyAll(card *detect.Card) {
	destBase, err := config.ExpandPath(a.cfg.Destination.Path)
	if err != nil {
		fmt.Printf("\nError: could not expand destination path: %v\n", err)
		a.printPrompt()
		return
	}

	if a.dryRun {
		fmt.Printf("\n[%s] Dry-run: would copy all files to %s\n", ts(), a.cfg.Destination.Path)
		a.printPrompt()
		return
	}

	// Warn if the card is write-protected — dotfile won't be written after copy.
	if cardIsReadOnly(card.Path) {
		fmt.Printf("\n[%s] Warning: card appears to be write-protected — copy status will not be saved to card\n", ts())
		a.logf("Card %s appears write-protected", card.Path)
	}

	fmt.Printf("\n[%s] Copying all files to %s\n", ts(), a.cfg.Destination.Path)
	fmt.Printf("[%s] Press [\\] to cancel\n", ts())
	a.logf("Copy starting: %s → %s", card.Path, destBase)

	a.mu.Lock()
	analyzeResult := a.lastResult
	a.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := cardcopy.Options{
		CardPath:      card.Path,
		DestBase:      destBase,
		BufferKB:      a.cfg.Advanced.BufferSizeKB,
		AnalyzeResult: analyzeResult,
	}

	type copyOutcome struct {
		result *cardcopy.Result
		err    error
	}
	doneCh := make(chan copyOutcome, 1)

	lastUpdate := time.Now()
	go func() {
		r, err := cardcopy.Run(ctx, opts, func(p cardcopy.Progress) {
			now := time.Now()
			if now.Sub(lastUpdate) < 2*time.Second && p.FilesDone < p.FilesTotal {
				return
			}
			lastUpdate = now
			pct := int64(0)
			if p.BytesTotal > 0 {
				pct = (p.BytesDone * 100) / p.BytesTotal
			}
			a.printMu.Lock()
			fmt.Printf("\r[%s] Copying... %d/%d files  %s/%s (%d%%)    ",
				ts(),
				p.FilesDone, p.FilesTotal,
				detect.FormatBytes(p.BytesDone),
				detect.FormatBytes(p.BytesTotal),
				pct)
			a.printMu.Unlock()
		})
		doneCh <- copyOutcome{r, err}
	}()

	var cardRemovedDuringCopy bool

	for {
		select {
		case outcome := <-doneCh:
			result, copyErr := outcome.result, outcome.err

			if errors.Is(copyErr, context.Canceled) {
				if cardRemovedDuringCopy {
					a.printMu.Lock()
					fmt.Printf("\n[%s] Copy stopped — card removed. %d files copied.\n",
						ts(), result.FilesCopied)
					a.printMu.Unlock()
					a.logf("Copy stopped: card removed. %d files copied.", result.FilesCopied)
					a.finishCard()
				} else {
					a.printMu.Lock()
					fmt.Printf("\n[%s] Copy cancelled — %d files copied.\n",
						ts(), result.FilesCopied)
					a.printMu.Unlock()
					a.logf("Copy cancelled. %d files copied.", result.FilesCopied)
					a.drainInput()
					a.printPrompt()
				}
				return
			}

			if copyErr != nil {
				a.printMu.Lock()
				fmt.Printf("\n[%s] Copy failed: %s\n", ts(), friendlyErr(copyErr))
				if result != nil && result.FilesCopied > 0 {
					fmt.Printf("[%s] %d files copied before failure.\n", ts(), result.FilesCopied)
				}
				a.printMu.Unlock()
				a.logf("Copy failed: %v", copyErr)
				a.drainInput()
				a.printPrompt()
				return
			}

			// --- Success ---
			elapsed := result.Elapsed.Round(time.Second)
			speed := float64(0)
			if result.Elapsed.Seconds() > 0 {
				speed = float64(result.BytesCopied) / result.Elapsed.Seconds() / (1024 * 1024)
			}
			a.printMu.Lock()
			fmt.Printf("\r[%s] Copy complete ✓                                          \n", ts())
			fmt.Printf("[%s] %d files, %s copied in %s (%.1f MB/s)\n",
				ts(),
				result.FilesCopied,
				detect.FormatBytes(result.BytesCopied),
				elapsed,
				speed)
			a.printMu.Unlock()
			a.logf("Copy complete: %d files, %s in %s (%.1f MB/s)",
				result.FilesCopied,
				detect.FormatBytes(result.BytesCopied),
				elapsed,
				speed)

			dotErr := dotfile.Write(dotfile.WriteOptions{
				CardPath:       card.Path,
				Destination:    destBase,
				Mode:           "all",
				FilesCopied:    result.FilesCopied,
				BytesCopied:    result.BytesCopied,
				Verified:       true,
				CardbotVersion: version,
			})
			if dotErr != nil {
				fmt.Printf("[%s] Warning: could not write .cardbot to card: %v\n", ts(), dotErr)
				a.logf("Dotfile write failed: %v", dotErr)
			} else {
				a.logf("Dotfile written to %s", card.Path)
			}

			a.mu.Lock()
			a.copied = true
			a.mu.Unlock()

			fmt.Println()
			a.drainInput()
			a.printPrompt()
			return

		case path := <-a.detector.Removals():
			if path == card.Path {
				// Current card pulled — cancel the copy and wait for doneCh.
				cardRemovedDuringCopy = true
				cancel()
			} else {
				// A queued card was removed; handle under printMu to avoid garbling progress output.
				a.printMu.Lock()
				a.handleRemoval(path)
				a.printMu.Unlock()
			}

		case newCard := <-a.detector.Events():
			// Queue any cards inserted while copying.
			a.printMu.Lock()
			a.handleCardEvent(newCard)
			a.printMu.Unlock()

		case input := <-a.inputChan:
			if strings.ToLower(input) == "\\" {
				cancel()
			}
			// All other input is silently ignored during copy.

		case <-a.sigChan:
			cancel()
			<-doneCh // wait for copy goroutine to finish
			fmt.Println("\nShutting down...")
			a.logf("Shutting down")
			close(a.inputDone)
			os.Exit(0)
		}
	}
}

// cardIsReadOnly probes the card path for write access.
// Returns true if a temp file cannot be created (write-protected card).
func cardIsReadOnly(path string) bool {
	probe := filepath.Join(path, ".cardbot_rw")
	f, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return true
	}
	f.Close()
	os.Remove(probe)
	return false
}

func (a *app) showHardwareInfo(card *detect.Card) {
	fmt.Println()
	hw := card.GetHW()
	if hw == nil {
		fmt.Println("Hardware info unavailable")
		fmt.Println()
		a.printPrompt()
		return
	}
	fmt.Println(detect.FormatHardwareInfo(hw))
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

	a.drainInput()
	a.printPrompt()
}

func (a *app) printPrompt() {
	a.mu.Lock()
	invalid := a.cardInvalid
	copied := a.copied
	a.mu.Unlock()

	switch {
	case invalid:
		fmt.Print("[e] Eject  [x] Exit  [?]  > ")
	case copied:
		fmt.Print("[e] Eject  [x] Done  [?]  > ")
	default:
		fmt.Print("[a] Copy All  [e] Eject  [x] Exit  [?]  > ")
	}
}

// printInvalidCardInfo shows basic card info when a card has no DCIM directory.
func (a *app) printInvalidCardInfo(card *detect.Card) {
	status := dotfile.Read(card.Path)
	fmt.Println()
	fmt.Printf("  Status:   %s\n", dotfile.FormatStatus(status))
	fmt.Printf("  Path:     %s\n", card.Path)
	var pct int64
	if card.TotalBytes > 0 {
		pct = (card.UsedBytes * 100) / card.TotalBytes
	}
	fmt.Printf("  Storage:  %s / %s (%d%%)\n",
		detect.FormatBytes(card.UsedBytes),
		detect.FormatBytes(card.TotalBytes),
		pct)
	color, reset := "", ""
	if a.cfg.Output.Color {
		color = ui.BrandColor(card.Brand)
		reset = ui.Reset
	}
	fmt.Printf("  Camera:   %s%s%s\n", color, card.Brand, reset)
	fmt.Printf("  Content:  (no DCIM — not a camera card)\n")
	fmt.Println("────────────────────────────────────────")
	a.printPrompt()
}

// showHelp prints all available commands.
func (a *app) showHelp() {
	fmt.Println()
	fmt.Println("  Commands:")
	fmt.Println("  [a]  Copy All     copy all files to destination")
	fmt.Println("  [s]  \033[9mCopy Selects\033[0m  copy starred/picked files only")
	fmt.Println("  [p]  \033[9mCopy Photos\033[0m   copy photos only")
	fmt.Println("  [v]  \033[9mCopy Videos\033[0m   copy videos only")
	fmt.Println("  [e]  Eject        safely eject this card")
	fmt.Println("  [x]  Exit         skip this card, move to next")
	fmt.Println("  [i]  Card Info    show hardware details")
	fmt.Println("  [t]  Speed Test   benchmark read/write speed")
	fmt.Println("  [\\]  Cancel Copy   cancel the copy in progress")
	fmt.Println("  [?]  Help         show this help")
	fmt.Println()
}

// friendlyErr returns a short, user-facing message for common OS-level errors.
func friendlyErr(err error) string {
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

func readInput(ch chan<- string, done <-chan struct{}) {
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		select {
		case ch <- strings.TrimSpace(line):
		case <-done:
			return
		}
	}
}
