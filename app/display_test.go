package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/illwill/cardbot/analyze"
	"github.com/illwill/cardbot/config"
	"github.com/illwill/cardbot/detect"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	return captureStdoutFD(t, fn)
}

func TestFriendlyErr(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"no space left on device", "destination disk is full"},
		{"permission denied", "permission denied — check folder permissions"},
		{"read-only file system", "destination is read-only"},
		{"input/output error", "I/O error — card may be damaged"},
		{"something else", "something else"},
	}

	for _, tt := range tests {
		if got := FriendlyErr(assertErr(tt.in)); got != tt.want {
			t.Fatalf("FriendlyErr(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCardIsReadOnly(t *testing.T) {
	if cardIsReadOnly(t.TempDir()) {
		t.Fatal("expected writable temp dir")
	}

	missing := filepath.Join(t.TempDir(), "missing")
	if !cardIsReadOnly(missing) {
		t.Fatal("expected missing dir to be treated as read-only")
	}
}

func TestPrintCardInfo(t *testing.T) {
	cfg := config.Defaults()
	cfg.Output.Color = false
	a := &App{cfg: cfg, copiedModes: make(map[string]bool)}
	card := &detect.Card{
		Path:       t.TempDir(),
		Brand:      "Nikon",
		TotalBytes: 1024 * 1024,
		UsedBytes:  512 * 1024,
	}

	result := &analyze.Result{
		Gear:       "Nikon Z 9",
		Starred:    1,
		PhotoCount: 2,
		VideoCount: 1,
		TotalSize:  123456,
		FileCount:  3,
		Groups: []analyze.DateGroup{
			{Date: "2026-03-10", Size: 100000, FileCount: 2, Extensions: []string{"JPG", "NEF"}},
			{Date: "2026-03-09", Size: 23456, FileCount: 1, Extensions: []string{"MOV"}},
		},
	}

	out := captureStdout(t, func() {
		a.printCardInfo(card, result)
	})

	for _, want := range []string{
		"Status:", "Path:", "Storage:", "Camera:", "Starred:", "Content:",
		"Total:", "Copy to:", "Naming:", "[a] Copy All",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\n%s", want, out)
		}
	}
}

func TestPrintInvalidCardInfo(t *testing.T) {
	cfg := config.Defaults()
	cfg.Output.Color = false
	a := &App{cfg: cfg, copiedModes: make(map[string]bool)}
	card := &detect.Card{Path: t.TempDir(), Brand: "Unknown"}

	out := captureStdout(t, func() {
		a.printInvalidCardInfo(card)
	})

	if !strings.Contains(out, "(no DCIM — not a camera card)") {
		t.Fatalf("missing invalid-card message\n%s", out)
	}
	if !strings.Contains(out, "[e] Eject") {
		t.Fatalf("missing prompt\n%s", out)
	}
}

func TestShowHelp(t *testing.T) {
	a := &App{}
	out := captureStdout(t, func() { a.showHelp() })
	if !strings.Contains(out, "[a]  Copy All") || !strings.Contains(out, "[?]  Help") {
		t.Fatalf("unexpected help output:\n%s", out)
	}
}

func TestShowHardwareInfo(t *testing.T) {
	cfg := config.Defaults()
	cfg.Output.Color = false
	a := &App{cfg: cfg, copiedModes: make(map[string]bool)}

	t.Run("nil hardware", func(t *testing.T) {
		card := &detect.Card{}
		out := captureStdout(t, func() { a.showHardwareInfo(card) })
		if !strings.Contains(out, "Hardware info unavailable") {
			t.Fatalf("unexpected output:\n%s", out)
		}
	})

	t.Run("with hardware", func(t *testing.T) {
		card := &detect.Card{}
		card.SetHW(&detect.HardwareInfo{})
		out := captureStdout(t, func() { a.showHardwareInfo(card) })
		if strings.Contains(out, "Hardware info unavailable") {
			t.Fatalf("expected hardware path, got:\n%s", out)
		}
		if !strings.Contains(out, "[a] Copy All") {
			t.Fatalf("missing prompt after hardware info:\n%s", out)
		}
	})
}

func TestPrintf(t *testing.T) {
	a := &App{}
	out := captureStdout(t, func() {
		a.Printf("hello %s", "world")
	})
	if out != "hello world" {
		t.Fatalf("Printf output = %q, want %q", out, "hello world")
	}
}

func TestStartAndStopScanning(t *testing.T) {
	cfg := config.Defaults()
	a := New(Config{Cfg: cfg})

	a.StartScanning()
	if a.spinner == nil {
		t.Fatal("expected spinner to be created")
	}
	if got := a.currentPhase(); got != phaseScanning {
		t.Fatalf("phase = %v, want %v", got, phaseScanning)
	}

	a.stopScanning()
	if a.spinner != nil {
		t.Fatal("expected spinner to be cleared")
	}

	// No-op second stop must be safe.
	a.stopScanning()
}

func TestTsPrefix_FirstCall_ReturnsTimestamp(t *testing.T) {
	a := New(Config{Cfg: config.Defaults()})
	prefix := a.TsPrefix()
	// Should contain ANSI dim codes and a bracketed timestamp.
	if !strings.Contains(prefix, "\033[2m") {
		t.Fatalf("expected dim ANSI code, got %q", prefix)
	}
	if !strings.Contains(prefix, "\033[0m") {
		t.Fatalf("expected reset ANSI code, got %q", prefix)
	}
	if !strings.Contains(prefix, "[") || !strings.Contains(prefix, "]") {
		t.Fatalf("expected bracketed timestamp, got %q", prefix)
	}
}

func TestTsPrefix_SameSecond_ReturnsIndent(t *testing.T) {
	a := New(Config{Cfg: config.Defaults()})
	first := a.TsPrefix()
	second := a.TsPrefix()
	if first == tsIndent {
		t.Fatal("first call should not return indent")
	}
	if second != tsIndent {
		t.Fatalf("second call in same second should return indent, got %q", second)
	}
}

func TestSetLastTS_SyncsWithTsPrefix(t *testing.T) {
	a := New(Config{Cfg: config.Defaults()})
	now := Ts()
	a.SetLastTS(now)
	// TsPrefix should return indent since we set the same second.
	got := a.TsPrefix()
	if got != tsIndent {
		t.Fatalf("TsPrefix after SetLastTS with current second should indent, got %q", got)
	}
}

func TestSetLastTS_DifferentSecond_ShowsTimestamp(t *testing.T) {
	a := New(Config{Cfg: config.Defaults()})
	a.SetLastTS("2000-01-01T00:00:00")
	got := a.TsPrefix()
	// Should return a new timestamp, not indent.
	if got == tsIndent {
		t.Fatal("TsPrefix after SetLastTS with old timestamp should not indent")
	}
}

func TestDimTS(t *testing.T) {
	got := dimTS("2026-03-22T12:00:00")
	want := "\033[2m[2026-03-22T12:00:00]\033[0m"
	if got != want {
		t.Fatalf("dimTS = %q, want %q", got, want)
	}
}

func TestTsIndent_Width(t *testing.T) {
	// tsIndent must match the visible width of "[2006-01-02T15:04:05]" (21 chars).
	if len(tsIndent) != 21 {
		t.Fatalf("tsIndent length = %d, want 21", len(tsIndent))
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func assertErr(s string) error { return errString(s) }
