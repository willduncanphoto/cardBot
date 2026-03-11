package analyze

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// createTestCard builds a fake DCIM structure in a temp directory.
// Returns the card root path. Caller should defer os.RemoveAll.
func createTestCard(t *testing.T, files map[string]testFile) string {
	t.Helper()
	root := t.TempDir()
	dcim := filepath.Join(root, "DCIM", "100NIKON")
	if err := os.MkdirAll(dcim, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, tf := range files {
		path := filepath.Join(dcim, name)
		if err := os.WriteFile(path, make([]byte, tf.size), 0o644); err != nil {
			t.Fatal(err)
		}
		if !tf.mtime.IsZero() {
			os.Chtimes(path, tf.mtime, tf.mtime)
		}
	}
	return root
}

type testFile struct {
	size  int
	mtime time.Time
}

func date(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 12, 0, 0, 0, time.UTC)
}

func TestAnalyze_MultiDay(t *testing.T) {
	root := createTestCard(t, map[string]testFile{
		"DSC_0001.NEF": {size: 50000, mtime: date(2025, 3, 8)},
		"DSC_0002.NEF": {size: 30000, mtime: date(2025, 3, 8)},
		"DSC_0003.JPG": {size: 10000, mtime: date(2025, 3, 8)},
		"DSC_0004.MOV": {size: 90000, mtime: date(2025, 3, 7)},
		"DSC_0005.NEF": {size: 40000, mtime: date(2025, 3, 7)},
	})

	result, err := New(root).Analyze()
	if err != nil {
		t.Fatal(err)
	}

	if result.FileCount != 5 {
		t.Errorf("FileCount = %d, want 5", result.FileCount)
	}
	if result.TotalSize != 220000 {
		t.Errorf("TotalSize = %d, want 220000", result.TotalSize)
	}
	if len(result.Groups) != 2 {
		t.Fatalf("len(Groups) = %d, want 2", len(result.Groups))
	}

	// Newest first.
	g0 := result.Groups[0]
	if g0.Date != "2025-03-08" {
		t.Errorf("Groups[0].Date = %q, want 2025-03-08", g0.Date)
	}
	if g0.FileCount != 3 {
		t.Errorf("Groups[0].FileCount = %d, want 3", g0.FileCount)
	}
	if g0.Size != 90000 {
		t.Errorf("Groups[0].Size = %d, want 90000", g0.Size)
	}
	// Extensions sorted alphabetically.
	wantExts := []string{"JPG", "NEF"}
	if len(g0.Extensions) != len(wantExts) {
		t.Errorf("Groups[0].Extensions = %v, want %v", g0.Extensions, wantExts)
	} else {
		for i, ext := range g0.Extensions {
			if ext != wantExts[i] {
				t.Errorf("Groups[0].Extensions[%d] = %q, want %q", i, ext, wantExts[i])
			}
		}
	}

	g1 := result.Groups[1]
	if g1.Date != "2025-03-07" {
		t.Errorf("Groups[1].Date = %q, want 2025-03-07", g1.Date)
	}
	if g1.FileCount != 2 {
		t.Errorf("Groups[1].FileCount = %d, want 2", g1.FileCount)
	}
	if g1.Size != 130000 {
		t.Errorf("Groups[1].Size = %d, want 130000", g1.Size)
	}
	wantExts1 := []string{"MOV", "NEF"}
	if len(g1.Extensions) != len(wantExts1) {
		t.Errorf("Groups[1].Extensions = %v, want %v", g1.Extensions, wantExts1)
	}

}

func TestAnalyze_SkipsHiddenFiles(t *testing.T) {
	root := createTestCard(t, map[string]testFile{
		"DSC_0001.NEF": {size: 1000, mtime: date(2025, 3, 8)},
		".DS_Store":    {size: 500, mtime: date(2025, 3, 8)},
		"._DSC_0001.NEF": {size: 300, mtime: date(2025, 3, 8)},
	})

	result, err := New(root).Analyze()
	if err != nil {
		t.Fatal(err)
	}

	if result.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1 (hidden files should be skipped)", result.FileCount)
	}
	if result.TotalSize != 1000 {
		t.Errorf("TotalSize = %d, want 1000", result.TotalSize)
	}
}

func TestAnalyze_EmptyCard(t *testing.T) {
	root := t.TempDir()
	dcim := filepath.Join(root, "DCIM")
	if err := os.MkdirAll(dcim, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := New(root).Analyze()
	if err != nil {
		t.Fatal(err)
	}

	if result.FileCount != 0 {
		t.Errorf("FileCount = %d, want 0", result.FileCount)
	}
	if result.TotalSize != 0 {
		t.Errorf("TotalSize = %d, want 0", result.TotalSize)
	}
	if len(result.Groups) != 0 {
		t.Errorf("len(Groups) = %d, want 0", len(result.Groups))
	}
}

func TestAnalyze_NoDCIM(t *testing.T) {
	root := t.TempDir()

	_, err := New(root).Analyze()
	if err == nil {
		t.Error("expected error for missing DCIM, got nil")
	}
}

func TestAnalyze_HiddenDirectory(t *testing.T) {
	root := t.TempDir()
	dcim := filepath.Join(root, "DCIM", "100NIKON")
	hidden := filepath.Join(root, "DCIM", ".Trashes")
	if err := os.MkdirAll(dcim, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(hidden, 0o755); err != nil {
		t.Fatal(err)
	}
	// Visible file.
	os.WriteFile(filepath.Join(dcim, "DSC_0001.NEF"), make([]byte, 1000), 0o644)
	// File inside hidden directory — should be skipped entirely.
	os.WriteFile(filepath.Join(hidden, "junk.dat"), make([]byte, 500), 0o644)

	result, err := New(root).Analyze()
	if err != nil {
		t.Fatal(err)
	}

	if result.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1 (files in hidden dirs should be skipped)", result.FileCount)
	}
}

func TestAnalyze_ExtensionNormalization(t *testing.T) {
	root := createTestCard(t, map[string]testFile{
		"photo.nef":  {size: 100, mtime: date(2025, 3, 8)},
		"photo.Nef":  {size: 100, mtime: date(2025, 3, 8)},
		"photo2.NEF": {size: 100, mtime: date(2025, 3, 8)},
	})

	result, err := New(root).Analyze()
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Groups) != 1 {
		t.Fatalf("len(Groups) = %d, want 1", len(result.Groups))
	}
	g := result.Groups[0]
	if len(g.Extensions) != 1 || g.Extensions[0] != "NEF" {
		t.Errorf("Extensions = %v, want [NEF] (case should be normalized)", g.Extensions)
	}
}

func TestNormalizeExt(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{".nef", "NEF"},
		{".NEF", "NEF"},
		{".Jpg", "JPG"},
		{"", ""},
	}
	for _, tt := range tests {
		got := normalizeExt(tt.input)
		if got != tt.want {
			t.Errorf("normalizeExt(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsHidden(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{".DS_Store", true},
		{"._resource", true},
		{".Trashes", true},
		{".hidden", true},
		{"DSC_0001.NEF", false},
		{"100NIKON", false},
	}
	for _, tt := range tests {
		got := isHidden(tt.name)
		if got != tt.want {
			t.Errorf("isHidden(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
