package app

import "testing"

func TestFormatDetectedVolume(t *testing.T) {
	t.Parallel()

	t.Run("with disk id", func(t *testing.T) {
		got := formatDetectedVolume("/Volumes/NIKON Z 9", "disk0s2")
		want := "\"/Volumes/NIKON Z 9\" [disk0s2] detected"
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
		want := "\"/Volumes/NIKON Z 9\" [disk0s2] detected"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}
