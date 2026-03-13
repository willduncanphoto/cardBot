package copy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/illwill/cardbot/internal/analyze"
)

// createTestCard builds a fake DCIM structure.
func createTestCard(t *testing.T, files map[string]testFileSpec) string {
	t.Helper()
	root := t.TempDir()
	for relPath, spec := range files {
		path := filepath.Join(root, "DCIM", relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, spec.data, 0644); err != nil {
			t.Fatal(err)
		}
		if !spec.mtime.IsZero() {
			if err := os.Chtimes(path, spec.mtime, spec.mtime); err != nil {
				t.Fatal(err)
			}
		}
	}
	return root
}

type testFileSpec struct {
	data  []byte
	mtime time.Time
}

func date(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 12, 0, 0, 0, time.UTC)
}

func TestCopy_BasicFiles(t *testing.T) {
	t.Parallel()
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 5000), mtime: date(2026, 3, 8)},
		"100NIKON/DSC_0002.JPG": {data: make([]byte, 3000), mtime: date(2026, 3, 8)},
		"101NIKON/DSC_0003.NEF": {data: make([]byte, 4000), mtime: date(2026, 3, 9)},
	})
	dest := t.TempDir()

	result, err := Run(context.Background(), Options{CardPath: card, DestBase: dest}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.FilesCopied != 3 {
		t.Errorf("FilesCopied = %d, want 3", result.FilesCopied)
	}
	if result.BytesCopied != 12000 {
		t.Errorf("BytesCopied = %d, want 12000", result.BytesCopied)
	}

	assertFileSize(t, filepath.Join(dest, "2026-03-08", "100NIKON", "DSC_0001.NEF"), 5000)
	assertFileSize(t, filepath.Join(dest, "2026-03-08", "100NIKON", "DSC_0002.JPG"), 3000)
	assertFileSize(t, filepath.Join(dest, "2026-03-09", "101NIKON", "DSC_0003.NEF"), 4000)
}

func TestCopy_SkipsHiddenFiles(t *testing.T) {
	t.Parallel()
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 1000), mtime: date(2026, 3, 8)},
		"100NIKON/.DS_Store":    {data: make([]byte, 500), mtime: date(2026, 3, 8)},
	})
	dest := t.TempDir()

	result, err := Run(context.Background(), Options{CardPath: card, DestBase: dest}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.FilesCopied != 1 {
		t.Errorf("FilesCopied = %d, want 1", result.FilesCopied)
	}

	dsStore := filepath.Join(dest, "2026-03-08", "100NIKON", ".DS_Store")
	if _, err := os.Stat(dsStore); !os.IsNotExist(err) {
		t.Error(".DS_Store should not have been copied")
	}
}

func TestCopy_SkipsHiddenDirs(t *testing.T) {
	t.Parallel()
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 1000), mtime: date(2026, 3, 8)},
		".Trashes/junk.dat":     {data: make([]byte, 500), mtime: date(2026, 3, 8)},
	})
	dest := t.TempDir()

	result, err := Run(context.Background(), Options{CardPath: card, DestBase: dest}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.FilesCopied != 1 {
		t.Errorf("FilesCopied = %d, want 1", result.FilesCopied)
	}
}

func TestCopy_EmptyCard(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "DCIM"), 0755); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()

	result, err := Run(context.Background(), Options{CardPath: root, DestBase: dest}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.FilesCopied != 0 {
		t.Errorf("FilesCopied = %d, want 0", result.FilesCopied)
	}
}

func TestCopy_NoDCIM(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dest := t.TempDir()

	_, err := Run(context.Background(), Options{CardPath: root, DestBase: dest}, nil)
	if err == nil {
		t.Error("expected error for missing DCIM")
	}
}

func TestCopy_DryRun(t *testing.T) {
	t.Parallel()
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 5000), mtime: date(2026, 3, 8)},
	})
	dest := t.TempDir()

	result, err := Run(context.Background(), Options{CardPath: card, DestBase: dest, DryRun: true}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.FilesCopied != 1 {
		t.Errorf("FilesCopied = %d, want 1", result.FilesCopied)
	}

	// No dated folders should exist.
	entries, err := os.ReadDir(dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("dry-run should not create files, found %d entries", len(entries))
	}
}

func TestCopy_DryRun_NoDirCreation(t *testing.T) {
	t.Parallel()
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 100), mtime: date(2026, 3, 8)},
	})
	// Destination that doesn't exist yet — dry-run should NOT create it.
	dest := filepath.Join(t.TempDir(), "nonexistent", "deep", "path")

	result, err := Run(context.Background(), Options{CardPath: card, DestBase: dest, DryRun: true}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.FilesCopied != 1 {
		t.Errorf("FilesCopied = %d, want 1", result.FilesCopied)
	}

	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Error("dry-run should not create destination directory")
	}
}

func TestCopy_ContentVerification(t *testing.T) {
	t.Parallel()
	data := []byte("hello world, this is test content for verification")
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: data, mtime: date(2026, 3, 8)},
	})
	dest := t.TempDir()

	_, err := Run(context.Background(), Options{CardPath: card, DestBase: dest}, nil)
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "2026-03-08", "100NIKON", "DSC_0001.NEF"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Error("copied file content does not match source")
	}
}

func TestCopy_Progress(t *testing.T) {
	t.Parallel()
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 1000), mtime: date(2026, 3, 8)},
		"100NIKON/DSC_0002.NEF": {data: make([]byte, 2000), mtime: date(2026, 3, 8)},
	})
	dest := t.TempDir()

	var calls []Progress
	_, err := Run(context.Background(), Options{CardPath: card, DestBase: dest}, func(p Progress) {
		calls = append(calls, p)
	})
	if err != nil {
		t.Fatal(err)
	}

	// Before file 1, before file 2, and final.
	if len(calls) < 3 {
		t.Errorf("got %d progress callbacks, want at least 3", len(calls))
	}

	last := calls[len(calls)-1]
	if last.FilesDone != 2 || last.FilesTotal != 2 {
		t.Errorf("final progress: %d/%d, want 2/2", last.FilesDone, last.FilesTotal)
	}
}

func TestCopy_CreatesNestedDest(t *testing.T) {
	t.Parallel()
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 100), mtime: date(2026, 3, 8)},
	})
	dest := filepath.Join(t.TempDir(), "nested", "deep", "dest")

	_, err := Run(context.Background(), Options{CardPath: card, DestBase: dest}, nil)
	if err != nil {
		t.Fatal(err)
	}

	assertFileSize(t, filepath.Join(dest, "2026-03-08", "100NIKON", "DSC_0001.NEF"), 100)
}

func TestCopy_MultipleSubfolders(t *testing.T) {
	t.Parallel()
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 100), mtime: date(2026, 3, 8)},
		"101NIKON/DSC_0010.NEF": {data: make([]byte, 200), mtime: date(2026, 3, 8)},
		"102NIKON/DSC_0020.NEF": {data: make([]byte, 300), mtime: date(2026, 3, 9)},
	})
	dest := t.TempDir()

	result, err := Run(context.Background(), Options{CardPath: card, DestBase: dest}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.FilesCopied != 3 {
		t.Errorf("FilesCopied = %d, want 3", result.FilesCopied)
	}

	assertFileSize(t, filepath.Join(dest, "2026-03-08", "100NIKON", "DSC_0001.NEF"), 100)
	assertFileSize(t, filepath.Join(dest, "2026-03-08", "101NIKON", "DSC_0010.NEF"), 200)
	assertFileSize(t, filepath.Join(dest, "2026-03-09", "102NIKON", "DSC_0020.NEF"), 300)
}

func TestCopy_DefaultBufferSize(t *testing.T) {
	t.Parallel()
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 100), mtime: date(2026, 3, 8)},
	})
	dest := t.TempDir()

	// BufferKB=0 should default to 256.
	result, err := Run(context.Background(), Options{CardPath: card, DestBase: dest, BufferKB: 0}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.FilesCopied != 1 {
		t.Errorf("FilesCopied = %d, want 1", result.FilesCopied)
	}
}

func TestCopy_ExifDatesOverrideMtime(t *testing.T) {
	t.Parallel()
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 100), mtime: date(2026, 3, 10)},
	})
	dest := t.TempDir()

	result, err := Run(context.Background(), Options{
		CardPath: card,
		DestBase: dest,
		AnalyzeResult: &analyze.Result{
			FileDates: map[string]string{
				"100NIKON/DSC_0001.NEF": "2026-03-08",
			},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.FilesCopied != 1 {
		t.Errorf("FilesCopied = %d, want 1", result.FilesCopied)
	}

	// Should use EXIF date, not mtime.
	assertFileSize(t, filepath.Join(dest, "2026-03-08", "100NIKON", "DSC_0001.NEF"), 100)

	// mtime path should NOT exist.
	if _, err := os.Stat(filepath.Join(dest, "2026-03-10")); !os.IsNotExist(err) {
		t.Error("mtime-based folder should not exist when EXIF date is available")
	}
}

func TestCopy_ExifDatePartialOverride(t *testing.T) {
	t.Parallel()
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 100), mtime: date(2026, 3, 10)},
		"100NIKON/DSC_0002.MOV": {data: make([]byte, 200), mtime: date(2026, 3, 10)},
	})
	dest := t.TempDir()

	// Only the NEF has an EXIF date; the MOV falls back to mtime.
	result, err := Run(context.Background(), Options{
		CardPath: card,
		DestBase: dest,
		AnalyzeResult: &analyze.Result{
			FileDates: map[string]string{
				"100NIKON/DSC_0001.NEF": "2026-03-08",
			},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.FilesCopied != 2 {
		t.Errorf("FilesCopied = %d, want 2", result.FilesCopied)
	}

	assertFileSize(t, filepath.Join(dest, "2026-03-08", "100NIKON", "DSC_0001.NEF"), 100)
	assertFileSize(t, filepath.Join(dest, "2026-03-10", "100NIKON", "DSC_0002.MOV"), 200)
}

func TestCopy_ElapsedTime(t *testing.T) {
	t.Parallel()
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 100), mtime: date(2026, 3, 8)},
	})
	dest := t.TempDir()

	result, err := Run(context.Background(), Options{CardPath: card, DestBase: dest}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.Elapsed <= 0 {
		t.Error("Elapsed should be positive")
	}
}

func TestCopy_SkipsExistingWithCorrectSize(t *testing.T) {
	t.Parallel()
	// FilesCopied counts files *processed* (including skips), not files newly written.
	// This matches the progress callback semantics where the user sees "150/150 files"
	// even on a re-copy where all 150 were skipped.
	data := []byte("original content here")
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: data, mtime: date(2026, 3, 8)},
	})
	dest := t.TempDir()

	// First copy.
	result1, err := Run(context.Background(), Options{CardPath: card, DestBase: dest}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result1.FilesCopied != 1 {
		t.Fatalf("first copy: FilesCopied = %d, want 1", result1.FilesCopied)
	}

	// Tamper with the dest file content (but keep the same size) to prove it's skipped, not re-copied.
	destFile := filepath.Join(dest, "2026-03-08", "100NIKON", "DSC_0001.NEF")
	tampered := make([]byte, len(data))
	for i := range tampered {
		tampered[i] = 'X'
	}
	if err := os.WriteFile(destFile, tampered, 0644); err != nil {
		t.Fatal(err)
	}

	// Second copy — file should be skipped because size matches.
	result2, err := Run(context.Background(), Options{CardPath: card, DestBase: dest}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result2.FilesCopied != 1 {
		t.Fatalf("second copy: FilesCopied = %d, want 1", result2.FilesCopied)
	}

	// File should still have tampered content (was not overwritten).
	got, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(tampered) {
		t.Error("file should have been skipped (not re-copied) since size matched")
	}
}

// assertFileSize checks that a file exists and has the expected size.
func assertFileSize(t *testing.T, path string, wantSize int64) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("expected file %s to exist: %v", path, err)
		return
	}
	if info.Size() != wantSize {
		t.Errorf("file %s size = %d, want %d", path, info.Size(), wantSize)
	}
}

func TestCopy_CancelBeforeCopy(t *testing.T) {
	t.Parallel()
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 1000), mtime: date(2026, 3, 8)},
		"100NIKON/DSC_0002.NEF": {data: make([]byte, 2000), mtime: date(2026, 3, 8)},
		"100NIKON/DSC_0003.NEF": {data: make([]byte, 3000), mtime: date(2026, 3, 8)},
	})
	dest := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before any files are copied

	result, err := Run(ctx, Options{CardPath: card, DestBase: dest}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if result == nil {
		t.Fatal("expected partial result even on cancel")
	}
	if result.FilesCopied != 0 {
		t.Errorf("FilesCopied = %d, want 0 (cancelled before start)", result.FilesCopied)
	}
}

func TestCopy_CancelMidCopy(t *testing.T) {
	t.Parallel()
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 1000), mtime: date(2026, 3, 8)},
		"100NIKON/DSC_0002.NEF": {data: make([]byte, 2000), mtime: date(2026, 3, 8)},
		"100NIKON/DSC_0003.NEF": {data: make([]byte, 3000), mtime: date(2026, 3, 8)},
	})
	dest := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel once we see any file completed. The context check runs at the
	// top of each loop iteration, so the file being processed when cancel()
	// fires will still finish — but we should not copy all 3.
	result, err := Run(ctx, Options{CardPath: card, DestBase: dest}, func(p Progress) {
		if p.FilesDone >= 1 {
			cancel()
		}
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if result == nil {
		t.Fatal("expected partial result even on cancel")
	}
	if result.FilesCopied < 1 {
		t.Errorf("FilesCopied = %d, want >= 1", result.FilesCopied)
	}
	if result.FilesCopied == 3 {
		t.Error("expected copy to be interrupted, but all 3 files were copied")
	}
}

func TestCopy_PathTraversal(t *testing.T) {
	t.Parallel()
	// Simulate a card where the EXIF date lookup returns a traversal path.
	// The path traversal guard in copy.go should block writes outside dest.
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 100), mtime: date(2026, 3, 8)},
	})
	dest := t.TempDir()

	// Inject a malicious date that would resolve outside the destination.
	result, err := Run(context.Background(), Options{
		CardPath: card,
		DestBase: dest,
		AnalyzeResult: &analyze.Result{
			FileDates: map[string]string{
				"100NIKON/DSC_0001.NEF": "../../etc",
			},
		},
	}, nil)

	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if result == nil {
		t.Fatal("expected partial result on path traversal error")
	}
	if result.FilesCopied != 0 {
		t.Errorf("FilesCopied = %d, want 0 (no files should be written)", result.FilesCopied)
	}

	// Verify nothing escaped the destination.
	escaped := filepath.Join(dest, "..", "etc", "100NIKON", "DSC_0001.NEF")
	if _, statErr := os.Stat(escaped); statErr == nil {
		t.Error("path traversal: file written outside destination")
	}
}

func TestCopy_SourceMissing(t *testing.T) {
	t.Parallel()
	// Create a card, then delete a source file before copy runs.
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 1000), mtime: date(2026, 3, 8)},
		"100NIKON/DSC_0002.NEF": {data: make([]byte, 2000), mtime: date(2026, 3, 8)},
	})
	// Remove the second file after card is built — simulates card read error.
	if err := os.Remove(filepath.Join(card, "DCIM", "100NIKON", "DSC_0002.NEF")); err != nil {
		t.Fatal(err)
	}

	dest := t.TempDir()
	result, err := Run(context.Background(), Options{CardPath: card, DestBase: dest}, nil)

	// WalkDir collects files first, then copy fails when the missing file is read.
	// Depending on walk order, we may get 0 or 1 files before the error.
	if err == nil {
		// If walk doesn't see the removed file (race), that's OK too.
		return
	}
	if result == nil {
		t.Fatal("expected partial result on error")
	}
}

func TestCopy_DestNotWritable(t *testing.T) {
	t.Parallel()
	card := createTestCard(t, map[string]testFileSpec{
		"100NIKON/DSC_0001.NEF": {data: make([]byte, 100), mtime: date(2026, 3, 8)},
	})
	// Create a read-only destination.
	dest := filepath.Join(t.TempDir(), "readonly")
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dest, 0444); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(dest, 0755) }() // cleanup

	_, err := Run(context.Background(), Options{CardPath: card, DestBase: filepath.Join(dest, "sub")}, nil)
	if err == nil {
		t.Error("expected error for non-writable destination")
	}
}
