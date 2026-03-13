package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/illwill/cardbot/internal/analyze"
	"github.com/illwill/cardbot/internal/config"
	"github.com/illwill/cardbot/internal/detect"
	cblog "github.com/illwill/cardbot/internal/log"
)

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
	lastResult  *analyze.Result     // analysis result for currentCard
	cardQueue   []*detect.Card
	mu          sync.Mutex
	printMu     sync.Mutex          // serialises concurrent stdout writes during copy
	cfg         *config.Config
	logger      *cblog.Logger
	inputChan   chan string          // buffered input from stdin
	sigChan     chan os.Signal       // SIGINT/SIGTERM
	inputDone   chan struct{}        // closed on shutdown to stop readInput
	dryRun      bool
	copiedModes map[string]bool     // modes completed this session
	cardInvalid bool                // true when current card has no DCIM directory
	spinStop    chan struct{}        // signals spinner goroutine to stop
	spinDone    chan struct{}        // closed when spinner goroutine exits
	scanCancel  context.CancelFunc  // cancels the current displayCard goroutine
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
		ctx, cancel := context.WithCancel(context.Background())
		a.scanCancel = cancel
		go a.displayCard(ctx, card.Path)
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

func (a *app) displayCard(ctx context.Context, path string) {
	// Card may have been removed or replaced before this goroutine runs.
	if ctx.Err() != nil {
		return
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

	result, err := analyzer.Analyze(ctx)

	// Cancelled — card was removed or replaced during analysis.
	if ctx.Err() != nil {
		return
	}

	if err != nil {
		if os.IsNotExist(err) {
			a.mu.Lock()
			if a.currentCard == nil || a.currentCard.Path != path {
				a.mu.Unlock()
				return
			}
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

	// Log any warnings from the scan.
	if result != nil {
		for _, w := range result.Warnings {
			a.logf("Scan warning: %s: %s", path, w)
		}
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
	if a.currentCard == nil || a.currentCard.Path != path {
		a.mu.Unlock()
		return
	}
	a.lastResult = result
	card := a.currentCard
	a.mu.Unlock()
	a.printCardInfo(card, result)
}

func (a *app) finishCard() {
	a.mu.Lock()
	if a.scanCancel != nil {
		a.scanCancel()
		a.scanCancel = nil
	}
	a.currentCard = nil
	a.lastResult = nil
	a.copiedModes = make(map[string]bool)
	a.cardInvalid = false

	if len(a.cardQueue) > 0 {
		nextCard := a.cardQueue[0]
		a.cardQueue = a.cardQueue[1:]
		a.currentCard = nextCard
		ctx, cancel := context.WithCancel(context.Background())
		a.scanCancel = cancel
		a.mu.Unlock()
		go a.displayCard(ctx, nextCard.Path)
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
		if a.scanCancel != nil {
			a.scanCancel()
			a.scanCancel = nil
		}
		a.currentCard = nil
		a.lastResult = nil
		a.copiedModes = make(map[string]bool)
		a.cardInvalid = false
		hasQueue := len(a.cardQueue) > 0
		var nextCard *detect.Card
		var nextCtx context.Context
		if hasQueue {
			nextCard = a.cardQueue[0]
			a.cardQueue = a.cardQueue[1:]
			a.currentCard = nextCard
			var cancel context.CancelFunc
			nextCtx, cancel = context.WithCancel(context.Background())
			a.scanCancel = cancel
		}
		a.mu.Unlock()

		fmt.Printf("\n[%s] Card removed: %s\n", ts(), path)
		a.logf("Card removed: %s", path)
		if hasQueue {
			go a.displayCard(nextCtx, nextCard.Path)
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
		a.handleCopyCmd(card, "all")
	case "s":
		a.handleCopyCmd(card, "selects")
	case "p":
		a.handleCopyCmd(card, "photos")
	case "v":
		a.handleCopyCmd(card, "videos")
	case "e":
		a.ejectCard(card)
	case "x":
		a.cancelCard()
	case "i":
		a.showHardwareInfo(card)
	case "t":
		a.runSpeedTest(card)
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

func (a *app) handleCopyCmd(card *detect.Card, mode string) {
	a.mu.Lock()
	invalid := a.cardInvalid
	copiedAll := a.copiedModes["all"]
	copiedMode := a.copiedModes[mode]
	analyzeResult := a.lastResult
	a.mu.Unlock()

	if invalid {
		fmt.Printf("\n[%s] No media found on this card.\n", ts())
		a.printPrompt()
		return
	}

	if copiedAll {
		fmt.Printf("\n[%s] Already copied.\n", ts())
		a.printPrompt()
		return
	}

	if copiedMode {
		modeName := "Selects"
		switch mode {
		case "photos": modeName = "Photos"
		case "videos": modeName = "Videos"
		}
		fmt.Printf("\n[%s] %s already copied.\n", ts(), modeName)
		a.printPrompt()
		return
	}

	if analyzeResult != nil {
		switch mode {
		case "selects":
			if analyzeResult.Starred == 0 {
				fmt.Printf("\n[%s] No starred files found on this card.\n", ts())
				a.printPrompt()
				return
			}
		case "photos":
			if analyzeResult.PhotoCount == 0 {
				fmt.Printf("\n[%s] No photo files found on this card.\n", ts())
				a.printPrompt()
				return
			}
		case "videos":
			if analyzeResult.VideoCount == 0 {
				fmt.Printf("\n[%s] No video files found on this card.\n", ts())
				a.printPrompt()
				return
			}
		}
	}

	a.copyFiltered(card, mode)
}
