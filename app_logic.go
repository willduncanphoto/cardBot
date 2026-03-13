package main

import (
	"strings"

	"github.com/illwill/cardbot/internal/analyze"
)

// inputAction is the parsed intent of a user command.
type inputAction int

const (
	actionNone inputAction = iota
	actionHelp
	actionCopyAll
	actionCopySelects
	actionCopyPhotos
	actionCopyVideos
	actionEject
	actionExitCard
	actionHardwareInfo
	actionSpeedTest
	actionNoCardMessage
	actionUnknown
)

// parseInputAction normalizes stdin input into a high-level action.
func parseInputAction(input string, hasCard bool) inputAction {
	cmd := strings.ToLower(strings.TrimSpace(input))

	if cmd == "?" {
		return actionHelp
	}

	if !hasCard {
		if cmd == "" {
			return actionNone
		}
		return actionNoCardMessage
	}

	switch cmd {
	case "":
		return actionNone
	case "a":
		return actionCopyAll
	case "s":
		return actionCopySelects
	case "p":
		return actionCopyPhotos
	case "v":
		return actionCopyVideos
	case "e":
		return actionEject
	case "x":
		return actionExitCard
	case "i":
		return actionHardwareInfo
	case "t":
		return actionSpeedTest
	default:
		return actionUnknown
	}
}

// modeDisplayName returns a user-facing mode label.
func modeDisplayName(mode string) string {
	switch mode {
	case "all":
		return "All"
	case "selects":
		return "Selects"
	case "photos":
		return "Photos"
	case "videos":
		return "Videos"
	default:
		if mode == "" {
			return "Copy"
		}
		r := []rune(mode)
		if len(r) == 0 {
			return "Copy"
		}
		return strings.ToUpper(string(r[0])) + string(r[1:])
	}
}

// copyBlockReason returns a user-facing reason that a copy command should be blocked.
// Empty string means the copy is allowed.
func copyBlockReason(mode string, invalid, copiedAll, copiedMode bool, result *analyze.Result) string {
	if invalid {
		return "No media found on this card."
	}

	if copiedAll {
		if mode == "all" {
			return "Already copied."
		}
		return modeDisplayName(mode) + " already copied."
	}

	if copiedMode {
		return modeDisplayName(mode) + " already copied."
	}

	if result == nil {
		return ""
	}

	switch mode {
	case "selects":
		if result.Starred == 0 {
			return "No starred files found on this card."
		}
	case "photos":
		if result.PhotoCount == 0 {
			return "No photo files found on this card."
		}
	case "videos":
		if result.VideoCount == 0 {
			return "No video files found on this card."
		}
	}

	return ""
}

// promptText returns the command prompt for the current card state.
func promptText(invalid, copiedAll bool) string {
	switch {
	case invalid:
		return "[e] Eject  [x] Exit  [?]  > "
	case copiedAll:
		return "[e] Eject  [x] Done  [?]  > "
	default:
		return "[a] Copy All  [e] Eject  [x] Exit  [?]  > "
	}
}

// shouldResumeScanning reports whether the scanner spinner should restart.
func shouldResumeScanning(noCurrentCard bool, queueLen int) bool {
	return noCurrentCard && queueLen == 0
}
