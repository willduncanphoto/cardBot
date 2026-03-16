package app

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/illwill/cardbot/internal/analyze"
	"github.com/illwill/cardbot/internal/detect"
)

func (a *App) handleCardEvent(card *detect.Card) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Ignore if already being processed or queued
	if a.isTracked(card.Path) {
		return
	}

	a.stopScanning()

	if a.currentCard == nil {
		a.currentCard = card
		// Format: "/Volumes/NAME" [disk identifier]
		hw := card.GetHW()
		diskID := ""
		if hw != nil && hw.DiskID() != "" {
			diskID = " [" + hw.DiskID() + "]"
		}
		fmt.Printf("[%s] \"%s\"%s detected\n", ts(), strings.TrimSpace(card.Path), diskID)
		a.logf("Card detected: %s", card.Path)
		ctx, cancel := context.WithCancel(context.Background())
		a.scanCancel = cancel
		go a.displayCard(ctx, card.Path)
	} else {
		a.cardQueue = append(a.cardQueue, card)
		a.printQueueNotice(card)
	}
}

func (a *App) isTracked(path string) bool {
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

func (a *App) printQueueNotice(card *detect.Card) {
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

func (a *App) displayCard(ctx context.Context, path string) {
	// Card may have been removed or replaced before this goroutine runs.
	if ctx.Err() != nil {
		return
	}

	fmt.Printf("[%s] Scanning", ts())
	a.logf("Reading %s", path)
	scanStart := time.Now()
	analyzer := analyze.New(path)
	analyzer.SetWorkers(a.cfg.Advanced.ExifWorkers)
	analyzer.OnProgress(func(count int) {
		if count%100 == 0 {
			fmt.Printf("\r[%s] Scanning %d files", ts(), count)
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
	fmt.Printf("\r[%s] Scanning %d files ✓\n", ts(), total)
	fmt.Printf("[%s] Scan complete (%ds)\n", ts(), secs)
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

func (a *App) finishCard() {
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

	a.StartScanning()
}

// resumeScanningIfIdle starts the scanning spinner only if
// no current card is active and no queued cards are waiting.
func (a *App) resumeScanningIfIdle() {
	a.mu.Lock()
	shouldStart := shouldResumeScanning(a.currentCard == nil, len(a.cardQueue))
	a.mu.Unlock()
	if !shouldStart {
		return
	}
	a.StartScanning()
}

func (a *App) handleRemoval(path string) {
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
				a.resumeScanningIfIdle()
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

func (a *App) handleInput(input string) {
	a.mu.Lock()
	card := a.currentCard
	hasCard := card != nil
	a.mu.Unlock()

	action := parseInputAction(input, hasCard)

	switch action {
	case actionNone:
		return
	case actionHelp:
		a.showHelp()
		if hasCard {
			a.printPrompt()
		}
		return
	case actionNoCardMessage:
		fmt.Printf("\nNo card inserted. Waiting for a memory card...\n")
		return
	}

	// Re-read current card to avoid acting on a stale pointer if card state changed.
	a.mu.Lock()
	card = a.currentCard
	a.mu.Unlock()
	if card == nil {
		fmt.Printf("\nNo card inserted. Waiting for a memory card...\n")
		return
	}

	switch action {
	case actionCopyAll:
		a.handleCopyCmd(card, "all")
	case actionCopySelects:
		a.handleCopyCmd(card, "selects")
	case actionCopyPhotos:
		a.handleCopyCmd(card, "photos")
	case actionCopyVideos:
		a.handleCopyCmd(card, "videos")
	case actionEject:
		a.ejectCard(card)
	case actionExitCard:
		a.cancelCard()
	case actionHardwareInfo:
		a.showHardwareInfo(card)
	case actionSpeedTest:
		a.runSpeedTest(card)
	case actionUnknown:
		fmt.Printf("\nUnknown command %q. Press [?] for help.\n", input)
		a.printPrompt()
	}
}

func (a *App) ejectCard(card *detect.Card) {
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

func (a *App) cancelCard() {
	fmt.Println("\nCancelled.")
	a.logf("Card cancelled")
	a.finishCard()
}

func (a *App) handleCopyCmd(card *detect.Card, mode string) {
	a.mu.Lock()
	invalid := a.cardInvalid
	copiedAll := a.copiedModes["all"]
	copiedMode := a.copiedModes[mode]
	analyzeResult := a.lastResult
	a.mu.Unlock()

	if reason := copyBlockReason(mode, invalid, copiedAll, copiedMode, analyzeResult); reason != "" {
		fmt.Printf("\n[%s] %s\n", ts(), reason)
		a.printPrompt()
		return
	}

	a.copyFiltered(card, mode)
}

// newStdinReader creates a buffered reader for stdin.
func newStdinReader() *bufio.Reader {
	return bufio.NewReader(os.Stdin)
}
