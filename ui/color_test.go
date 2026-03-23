package ui

import "testing"

func TestBrandColor_KnownBrands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		brand string
		want  string
	}{
		{"Nikon", "\033[33m"},
		{"Canon", "\033[31m"},
		{"Sony", "\033[37m"},
		{"Fujifilm", "\033[32m"},
		{"Panasonic", "\033[34m"},
		{"Olympus", "\033[36m"},
		{"OM System", "\033[36m"},
	}
	for _, tt := range tests {
		got := BrandColor(tt.brand)
		if got != tt.want {
			t.Errorf("BrandColor(%q) = %q, want %q", tt.brand, got, tt.want)
		}
	}
}

func TestBrandColor_Unknown(t *testing.T) {
	t.Parallel()
	// Unknown brands should return white.
	for _, brand := range []string{"Unknown", "Leica", "Hasselblad", ""} {
		got := BrandColor(brand)
		if got != "\033[37m" {
			t.Errorf("BrandColor(%q) = %q, want white (\\033[37m)", brand, got)
		}
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
	if Reset != "\033[0m" {
		t.Errorf("Reset = %q, want \\033[0m", Reset)
	}
}
