package main

import (
	"fmt"
	"strings"

	"github.com/illwill/cardbot/internal/analyze"
	"github.com/illwill/cardbot/internal/detect"
	"github.com/illwill/cardbot/internal/dotfile"
	"github.com/illwill/cardbot/internal/ui"
)

// printCardHeader renders the shared header lines used by both printCardInfo
// and printInvalidCardInfo: Status, Path, Storage, Camera.
func (a *app) printCardHeader(card *detect.Card, cameraDisplay string) {
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
	fmt.Printf("  Camera:   %s%s%s\n", color, cameraDisplay, reset)
}

func (a *app) printCardInfo(card *detect.Card, result *analyze.Result) {
	camera := card.Brand + " (unknown model)"
	if result != nil && result.Gear != "" {
		camera = result.Gear
	}
	a.printCardHeader(card, camera)

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

// printInvalidCardInfo shows basic card info when a card has no DCIM directory.
func (a *app) printInvalidCardInfo(card *detect.Card) {
	fmt.Println()
	a.printCardHeader(card, card.Brand)
	fmt.Printf("  Content:  (no DCIM — not a camera card)\n")
	fmt.Println("────────────────────────────────────────")
	a.printPrompt()
}

func (a *app) printPrompt() {
	a.mu.Lock()
	invalid := a.cardInvalid
	copiedAll := a.copiedModes["all"]
	a.mu.Unlock()

	switch {
	case invalid:
		fmt.Print("[e] Eject  [x] Exit  [?]  > ")
	case copiedAll:
		fmt.Print("[e] Eject  [x] Done  [?]  > ")
	default:
		fmt.Print("[a] Copy All  [e] Eject  [x] Exit  [?]  > ")
	}
}

// showHelp prints all available commands.
func (a *app) showHelp() {
	fmt.Println()
	fmt.Println("  Commands:")
	fmt.Println("  [a]  Copy All     copy all files to destination")
	fmt.Println("  [s]  Copy Selects copy starred/picked files only")
	fmt.Println("  [p]  Copy Photos  copy photos only")
	fmt.Println("  [v]  Copy Videos  copy videos only")
	fmt.Println("  [e]  Eject        safely eject this card")
	fmt.Println("  [x]  Exit         skip this card, move to next")
	fmt.Println("  [i]  Card Info    show hardware details")
	fmt.Println("  [t]  Speed Test   benchmark read/write speed")
	fmt.Println("  [\\]  Cancel Copy  cancel the copy in progress")
	fmt.Println("  [?]  Help         show this help")
	fmt.Println()
}

// showHardwareInfo displays hardware details for the current card.
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
