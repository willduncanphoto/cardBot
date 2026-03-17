package app

import (
	"testing"
	"time"
)

func TestFormatElapsed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   time.Duration
		want string
	}{
		{100 * time.Millisecond, "0.1s"},
		{850 * time.Millisecond, "0.9s"},
		{time.Second, "1s"},
		{1500 * time.Millisecond, "2s"},
	}

	for _, tt := range tests {
		if got := formatElapsed(tt.in); got != tt.want {
			t.Fatalf("formatElapsed(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatDetectedVolume(t *testing.T) {
	t.Parallel()

	t.Run("with disk id", func(t *testing.T) {
		got := formatDetectedVolume("/Volumes/NIKON Z 9", "disk0s2")
		want := "\"/Volumes/NIKON Z 9\" (disk0s2) detected"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("without disk id", func(t *testing.T) {
		got := formatDetectedVolume("/Volumes/NIKON Z 9", "")
		want := "\"/Volumes/NIKON Z 9\" detected"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		got := formatDetectedVolume("  /Volumes/NIKON Z 9  ", "  disk0s2  ")
		want := "\"/Volumes/NIKON Z 9\" (disk0s2) detected"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}
