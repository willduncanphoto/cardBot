package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/illwill/cardbot/internal/analyze"
	"github.com/illwill/cardbot/internal/detect"
)

const version = "0.1.3"

// UX delays — remove in 0.4.0 when real startup and analysis timings replace them.
const (
	removalDelay = 2 * time.Second // Pause after card removal so message is visible
)

type app struct {
	detector    *detect.Detector
	currentCard *detect.Card
	cardQueue   []*detect.Card
	mu          sync.Mutex
}

func main() {
	fmt.Printf("[%s] Starting CardBot %s...\n", time.Now().Format("2006-01-02 15:04:05"), version)
	fmt.Printf("[%s] Scanning for memory cards...", time.Now().Format("2006-01-02 15:04:05"))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	a := &app{
		cardQueue: make([]*detect.Card, 0),
	}

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
			return
		}
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
		fmt.Println("card found.")
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
		time.Now().Format("2006-01-02 15:04:05"),
		card.Brand,
		len(a.cardQueue),
		plural)
}

func (a *app) displayCard(card *detect.Card) {
	if !a.isCurrentCard(card.Path) {
		return // Card was cancelled or removed while starting
	}

	fmt.Printf("[%s] Scanning %s... ", time.Now().Format("2006-01-02 15:04:05"), card.Path)
	scanStart := time.Now()
	analyzer := analyze.New(card.Path)
	analyzer.OnProgress(func(count int) {
		fmt.Printf("\r[%s] Scanning %s... %d files", time.Now().Format("2006-01-02 15:04:05"), card.Path, count)
	})

	result, err := analyzer.Analyze()
	if err != nil {
		fmt.Printf("\nError analyzing card: %v\n", err)
		a.finishCard()
		return
	}

	elapsed := time.Since(scanStart)
	secs := int(elapsed.Round(time.Second).Seconds())

	total := 0
	if result != nil {
		total = result.FileCount
	}
	fmt.Printf("\r[%s] Scanning %s... %d files ✓\n", time.Now().Format("2006-01-02 15:04:05"), card.Path, total)
	secWord := "seconds"
	if secs == 1 {
		secWord = "second"
	}
	fmt.Printf("[%s] Scan completed in %d %s\n", time.Now().Format("2006-01-02 15:04:05"), secs, secWord)
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
		formatBytes(card.UsedBytes),
		formatBytes(card.TotalBytes),
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
				formatBytes(g.Size),
				countWidth,
				g.FileCount,
				strings.Join(g.Extensions, ", "))
		}
		fmt.Println()
		fmt.Printf("  Total:    %d photos, %d videos, %s\n", result.PhotoCount, result.VideoCount, formatBytes(result.TotalSize))
	} else {
		fmt.Println("  Content:  (empty)")
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

	fmt.Printf("\n[%s] Scanning for memory cards...", time.Now().Format("2006-01-02 15:04:05"))
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

		fmt.Printf("\n[%s] Card removed: %s\n", time.Now().Format("2006-01-02 15:04:05"), path)
		if hasQueue {
			go a.displayCard(nextCard)
		} else {
			time.Sleep(removalDelay)
			fmt.Printf("\n[%s] Scanning for memory cards...", time.Now().Format("2006-01-02 15:04:05"))
		}
		return
	}

	// Check queue
	for i, card := range a.cardQueue {
		if card.Path == path {
			a.cardQueue = append(a.cardQueue[:i], a.cardQueue[i+1:]...)
			a.mu.Unlock()
			fmt.Printf("\n[%s] Queued card removed: %s\n", time.Now().Format("2006-01-02 15:04:05"), path)
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
	}
}

func (a *app) ejectCard(card *detect.Card) {
	fmt.Printf("\nEjecting %s...\n", card.Name)
	if err := a.detector.Eject(card.Path); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	a.detector.Remove(card.Path)
	fmt.Printf("\n[%s] Card ejected: %s\n", time.Now().Format("2006-01-02 15:04:05"), card.Path)
	a.finishCard()
}

func (a *app) cancelCard() {
	fmt.Println("\nCancelled.")
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

func formatBytes(b int64) string {
	const unit = 1024
	if b <= 0 {
		return "0 B"
	}
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
