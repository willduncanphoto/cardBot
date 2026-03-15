package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/illwill/cardbot/internal/analyze"
	"github.com/illwill/cardbot/internal/config"
	cardcopy "github.com/illwill/cardbot/internal/copy"
	"github.com/illwill/cardbot/internal/detect"
	"github.com/illwill/cardbot/internal/dotfile"
	"github.com/illwill/cardbot/internal/speedtest"
)

const dryRunPreviewLimit = 200

func (a *app) copyFiltered(card *detect.Card, mode string) {
	destBase, err := config.ExpandPath(a.cfg.Destination.Path)
	if err != nil {
		fmt.Printf("\n[%s] Error: %s\n", ts(), friendlyErr(err))
		a.printPrompt()
		return
	}

	// Validate destination path.
	if destBase == "" {
		fmt.Printf("\n[%s] Error: no destination configured — run cardbot --setup\n", ts())
		a.printPrompt()
		return
	}

	isDryRun := a.dryRun

	// Warn if the card is write-protected — dotfile won't be written after copy.
	// (Skip warning in dry-run since we're not writing anyway.)
	if !isDryRun && cardIsReadOnly(card.Path) {
		fmt.Printf("\n[%s] Warning: card appears to be write-protected — copy status will not be saved to card\n", ts())
		a.logf("Card %s appears write-protected", card.Path)
	}

	// Human-readable mode name for output.
	modeStr := mode
	if mode == "selects" {
		modeStr = "starred"
	}
	if isDryRun {
		fmt.Printf("\n[%s] Dry-run: would copy %s files to %s\n", ts(), modeStr, a.cfg.Destination.Path)
	} else {
		fmt.Printf("\n[%s] Copying %s files to %s\n", ts(), modeStr, a.cfg.Destination.Path)
		fmt.Printf("[%s] Press [\\] to cancel\n", ts())
	}
	a.logf("Copy %s starting: %s → %s", mode, card.Path, destBase)

	a.mu.Lock()
	analyzeResult := a.lastResult
	a.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var filter func(relPath, ext string) bool
	switch mode {
	case "photos":
		filter = func(relPath, ext string) bool { return analyze.IsPhoto(ext) }
	case "videos":
		filter = func(relPath, ext string) bool { return analyze.IsVideo(ext) }
	case "selects":
		filter = func(relPath, ext string) bool {
			return analyzeResult != nil && analyzeResult.FileRatings != nil && analyzeResult.FileRatings[relPath] > 0
		}
	}
	opts := cardcopy.Options{
		CardPath:      card.Path,
		DestBase:      destBase,
		BufferKB:      a.cfg.Advanced.BufferSizeKB,
		DryRun:        isDryRun,
		AnalyzeResult: analyzeResult,
		Filter:        filter,
		NamingMode:    a.cfg.Naming.Mode,
	}

	type copyOutcome struct {
		result *cardcopy.Result
		err    error
	}
	doneCh := make(chan copyOutcome, 1)

	lastUpdate := time.Now()
	previewPrinted := 0
	previewHidden := 0
	go func() {
		r, err := cardcopy.Run(ctx, opts, func(p cardcopy.Progress) {
			// In dry-run mode, print rename mappings (capped for large cards).
			if isDryRun {
				if p.SourceFile == "" {
					return
				}
				a.printMu.Lock()
				if previewPrinted < dryRunPreviewLimit {
					if p.SourceFile != p.CurrentFile {
						fmt.Printf("  %s → %s\n", p.SourceFile, p.CurrentFile)
					} else {
						fmt.Printf("  %s (unchanged)\n", p.SourceFile)
					}
					previewPrinted++
				} else {
					previewHidden++
				}
				a.printMu.Unlock()
				return
			}
			// Normal progress display during actual copy.
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

			// Log any walk warnings.
			if result != nil {
				for _, w := range result.Warnings {
					a.logf("Copy warning: %s", w)
				}
			}

			if errors.Is(copyErr, context.Canceled) {
				copied := 0
				if result != nil {
					copied = result.FilesCopied
				}
				if cardRemovedDuringCopy {
					a.printMu.Lock()
					fmt.Printf("\n[%s] Copy stopped — card removed. %d files copied.\n",
						ts(), copied)
					a.printMu.Unlock()
					a.logf("Copy stopped: card removed. %d files copied.", copied)
					a.finishCard()
				} else {
					a.printMu.Lock()
					fmt.Printf("\n[%s] Copy cancelled — %d files copied.\n",
						ts(), copied)
					a.printMu.Unlock()
					a.logf("Copy cancelled. %d files copied.", copied)
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
			a.handleCopySuccess(card, mode, destBase, result, isDryRun, previewHidden)
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

func (a *app) handleCopySuccess(card *detect.Card, mode, destBase string, result *cardcopy.Result, isDryRun bool, previewHidden int) {
	if result == nil {
		result = &cardcopy.Result{}
	}

	elapsed := result.Elapsed.Round(time.Second)
	speed := float64(0)
	if result.Elapsed.Seconds() > 0 {
		speed = float64(result.BytesCopied) / result.Elapsed.Seconds() / (1024 * 1024)
	}

	if isDryRun {
		a.printMu.Lock()
		fmt.Printf("[%s] Dry-run complete ✓\n", ts())
		fmt.Printf("[%s] %d files, %s would be copied\n",
			ts(),
			result.FilesCopied,
			detect.FormatBytes(result.BytesCopied))
		if previewHidden > 0 {
			fmt.Printf("[%s] ... +%d more files (preview capped at %d)\n", ts(), previewHidden, dryRunPreviewLimit)
		}
		a.printMu.Unlock()
		a.logf("Dry-run complete: %d files, %s would be copied", result.FilesCopied, detect.FormatBytes(result.BytesCopied))
		return
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
		Mode:           mode,
		FilesCopied:    result.FilesCopied,
		BytesCopied:    result.BytesCopied,
		Verified:       true,
		CardbotVersion: version,
	})
	if dotErr != nil {
		fmt.Printf("[%s] Warning: could not write .cardbot to card: %s\n", ts(), friendlyErr(dotErr))
		a.logf("Dotfile write failed: %v", dotErr)
	} else {
		a.logf("Dotfile written to %s", card.Path)
	}

	a.mu.Lock()
	a.copiedModes[mode] = true
	a.mu.Unlock()
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
		fmt.Printf("Speed test failed: %s\n", friendlyErr(err))
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
