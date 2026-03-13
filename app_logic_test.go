package main

import (
	"testing"

	"github.com/illwill/cardbot/internal/analyze"
)

func TestParseInputAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		hasCard bool
		want    inputAction
	}{
		{"help no card", "?", false, actionHelp},
		{"help with card", "?", true, actionHelp},
		{"empty no card", "", false, actionNone},
		{"empty with card", "", true, actionNone},
		{"copy all", "a", true, actionCopyAll},
		{"copy selects", "s", true, actionCopySelects},
		{"copy photos", "p", true, actionCopyPhotos},
		{"copy videos", "v", true, actionCopyVideos},
		{"eject", "e", true, actionEject},
		{"exit", "x", true, actionExitCard},
		{"info", "i", true, actionHardwareInfo},
		{"speed", "t", true, actionSpeedTest},
		{"uppercase + spaces", "  A  ", true, actionCopyAll},
		{"unknown with card", "zzz", true, actionUnknown},
		{"input but no card", "a", false, actionNoCardMessage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseInputAction(tt.input, tt.hasCard); got != tt.want {
				t.Errorf("parseInputAction(%q, %v) = %v, want %v", tt.input, tt.hasCard, got, tt.want)
			}
		})
	}
}

func TestModeDisplayName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode string
		want string
	}{
		{"all", "All"},
		{"selects", "Selects"},
		{"photos", "Photos"},
		{"videos", "Videos"},
		{"custom", "Custom"},
		{"étoiles", "Étoiles"},
		{"", "Copy"},
	}

	for _, tt := range tests {
		if got := modeDisplayName(tt.mode); got != tt.want {
			t.Errorf("modeDisplayName(%q) = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestCopyBlockReason(t *testing.T) {
	t.Parallel()

	result := &analyze.Result{Starred: 0, PhotoCount: 0, VideoCount: 0}

	tests := []struct {
		name       string
		mode       string
		invalid    bool
		copiedAll  bool
		copiedMode bool
		result     *analyze.Result
		want       string
	}{
		{"invalid card", "all", true, false, false, nil, "No media found on this card."},
		{"all already copied", "all", false, true, false, nil, "Already copied."},
		{"photos blocked by all", "photos", false, true, false, nil, "Photos already copied."},
		{"mode already copied", "videos", false, false, true, nil, "Videos already copied."},
		{"no starred", "selects", false, false, false, result, "No starred files found on this card."},
		{"no photos", "photos", false, false, false, result, "No photo files found on this card."},
		{"no videos", "videos", false, false, false, result, "No video files found on this card."},
		{"allowed all", "all", false, false, false, result, ""},
		{"allowed nil result", "all", false, false, false, nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := copyBlockReason(tt.mode, tt.invalid, tt.copiedAll, tt.copiedMode, tt.result)
			if got != tt.want {
				t.Errorf("copyBlockReason() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPromptText(t *testing.T) {
	t.Parallel()

	if got := promptText(true, false); got != "[e] Eject  [x] Exit  [?]  > " {
		t.Errorf("invalid prompt = %q", got)
	}
	if got := promptText(false, true); got != "[e] Eject  [x] Done  [?]  > " {
		t.Errorf("copied prompt = %q", got)
	}
	if got := promptText(false, false); got != "[a] Copy All  [e] Eject  [x] Exit  [?]  > " {
		t.Errorf("default prompt = %q", got)
	}
}

func TestShouldResumeScanning(t *testing.T) {
	t.Parallel()

	if !shouldResumeScanning(true, 0) {
		t.Error("expected idle state to resume scanning")
	}
	if shouldResumeScanning(false, 0) {
		t.Error("should not resume scanning when a current card exists")
	}
	if shouldResumeScanning(true, 1) {
		t.Error("should not resume scanning when queue is non-empty")
	}
}
