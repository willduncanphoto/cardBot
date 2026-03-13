//go:build darwin || linux

package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectBrand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		folders []string
		want    string
	}{
		{"Nikon NIKON", []string{"100NIKON"}, "Nikon"},
		{"Nikon Z9", []string{"100NCZ_9"}, "Nikon"},
		{"Nikon Z8", []string{"100NCZ_8"}, "Nikon"},
		{"Nikon NZ_", []string{"100NZ_6"}, "Nikon"},
		{"Nikon D850", []string{"100ND850"}, "Nikon"},
		{"Canon", []string{"100CANON"}, "Canon"},
		{"Canon EOS", []string{"100EOS5D"}, "Canon"},
		{"Sony", []string{"100MSDCF"}, "Sony"},
		{"Sony explicit", []string{"101SONY"}, "Sony"},
		{"Fujifilm", []string{"100_FUJI"}, "Fujifilm"},
		{"Panasonic PANA", []string{"100_PANA"}, "Panasonic"},
		{"Panasonic LUMIX", []string{"100LUMIX"}, "Panasonic"},
		{"Olympus", []string{"100OLYMP"}, "Olympus"},
		{"Unknown", []string{"100ABCDE"}, "Unknown"},
		{"Empty DCIM", []string{}, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			dcim := filepath.Join(root, "DCIM")
			if err := os.MkdirAll(dcim, 0755); err != nil {
				t.Fatal(err)
			}
			for _, folder := range tt.folders {
				if err := os.MkdirAll(filepath.Join(dcim, folder), 0755); err != nil {
					t.Fatal(err)
				}
			}

			got := detectBrand(root)
			if got != tt.want {
				t.Errorf("detectBrand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectBrand_NoDCIM(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	got := detectBrand(root)
	if got != "Unknown" {
		t.Errorf("detectBrand(no DCIM) = %q, want Unknown", got)
	}
}

func TestContainsNDModel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"100ND850", true},
		{"100ND750", true},
		{"100ND5", true},
		{"STANDARD", false},
		{"ANDROID", false},
		{"ND", false},
		{"NDx", false},
	}

	for _, tt := range tests {
		got := containsNDModel(tt.input)
		if got != tt.want {
			t.Errorf("containsNDModel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
