package app

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
		{"unknown with card", "z", true, actionUnknown},
		{"input but no card", "z", false, actionNoCardMessage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseInputAction(tt.input, tt.hasCard)
			if got != tt.want {
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
		{"", "Copy"},
		{"custom", "Custom"},
	}

	for _, tt := range tests {
		got := modeDisplayName(tt.mode)
		if got != tt.want {
			t.Errorf("modeDisplayName(%q) = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestCopyBlockReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mode       string
		invalid    bool
		copiedAll  bool
		copiedMode bool
		result     *analyze.Result
		wantEmpty  bool
	}{
		{"invalid card", "all", true, false, false, nil, false},
		{"all already copied", "all", false, true, false, nil, false},
		{"mode already copied", "photos", false, false, true, nil, false},
		{"selects no starred", "selects", false, false, false, &analyze.Result{Starred: 0}, false},
		{"selects has starred", "selects", false, false, false, &analyze.Result{Starred: 5}, true},
		{"photos no photos", "photos", false, false, false, &analyze.Result{PhotoCount: 0}, false},
		{"photos has photos", "photos", false, false, false, &analyze.Result{PhotoCount: 10}, true},
		{"videos no videos", "videos", false, false, false, &analyze.Result{VideoCount: 0}, false},
		{"videos has videos", "videos", false, false, false, &analyze.Result{VideoCount: 3}, true},
		{"all allowed", "all", false, false, false, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := copyBlockReason(tt.mode, tt.invalid, tt.copiedAll, tt.copiedMode, tt.result)
			if (got == "") != tt.wantEmpty {
				t.Errorf("copyBlockReason() = %q, wantEmpty = %v", got, tt.wantEmpty)
			}
		})
	}
}

func TestPromptText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		invalid   bool
		copiedAll bool
		contains  string
	}{
		{"normal", false, false, "Copy All"},
		{"invalid card", true, false, "Eject"},
		{"copied all", false, true, "Done"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := promptText(tt.invalid, tt.copiedAll)
			if !contains(got, tt.contains) {
				t.Errorf("promptText() = %q, should contain %q", got, tt.contains)
			}
		})
	}
}

func TestShouldResumeScanning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		noCurrentCard bool
		queueLen      int
		want          bool
	}{
		{"no card, empty queue", true, 0, true},
		{"has card", false, 0, false},
		{"no card, has queue", true, 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldResumeScanning(tt.noCurrentCard, tt.queueLen)
			if got != tt.want {
				t.Errorf("shouldResumeScanning(%v, %d) = %v, want %v", tt.noCurrentCard, tt.queueLen, got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
