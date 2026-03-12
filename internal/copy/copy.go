// Package copy handles file copying from memory cards to the destination.
package copy

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/illwill/cardbot/internal/analyze"
)

// ProgressFunc is called periodically during the copy with current stats.
type ProgressFunc func(stats Progress)

// Progress holds real-time copy stats for the callback.
type Progress struct {
	FilesDone   int
	FilesTotal  int
	BytesDone   int64
	BytesTotal  int64
	CurrentFile string // relative path being copied
}

// Result holds the final outcome of a copy operation.
type Result struct {
	FilesCopied int
	BytesCopied int64
	Elapsed     time.Duration
	DestPath    string
}

// Options configures the copy operation.
type Options struct {
	CardPath    string           // Source card mount point
	DestBase    string           // Base destination directory (e.g. ~/Pictures/CardBot)
	BufferKB    int              // Copy buffer size in KB (default 256)
	DryRun      bool             // If true, walk and report but don't copy
	AnalyzeResult *analyze.Result // If provided, use EXIF dates for folder grouping
}

// fileEntry holds a file to be copied.
type fileEntry struct {
	srcPath string // absolute source path on card
	relPath string // relative path from DCIM (e.g. "100NIKON/DSC_0001.NEF")
	size    int64
	date    string // YYYY-MM-DD for folder grouping
}

// Run executes the copy operation.
// It walks DCIM, groups files by date into destination folders,
// copies with buffered I/O, verifies sizes, and returns a summary.
func Run(opts Options, onProgress ProgressFunc) (*Result, error) {
	if opts.BufferKB <= 0 {
		opts.BufferKB = 256
	}

	dcim := filepath.Join(opts.CardPath, "DCIM")
	if _, err := os.Stat(dcim); err != nil {
		return nil, fmt.Errorf("no DCIM folder found on card")
	}

	// Build EXIF date lookup from analyze result if available.
	var exifDates map[string]string
	if opts.AnalyzeResult != nil {
		exifDates = opts.AnalyzeResult.FileDates
	}

	// --- Phase 1: Collect files ---
	var files []fileEntry
	var totalBytes int64

	err := filepath.WalkDir(dcim, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		rel, _ := filepath.Rel(dcim, path)

		// Use EXIF date if available, fall back to mtime.
		date := info.ModTime().Format("2006-01-02")
		if exifDate, ok := exifDates[rel]; ok {
			date = exifDate
		}

		files = append(files, fileEntry{
			srcPath: path,
			relPath: rel,
			size:    info.Size(),
			date:    date,
		})
		totalBytes += info.Size()
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking DCIM: %w", err)
	}

	if len(files) == 0 {
		return &Result{DestPath: opts.DestBase}, nil
	}

	if opts.DryRun {
		return &Result{
			FilesCopied: len(files),
			BytesCopied: totalBytes,
			DestPath:    opts.DestBase,
		}, nil
	}

	// Verify destination is writable.
	// Skip the probe if the directory already exists (we've written here before).
	if _, err := os.Stat(opts.DestBase); os.IsNotExist(err) {
		if err := os.MkdirAll(opts.DestBase, 0755); err != nil {
			return nil, fmt.Errorf("cannot create destination %s: %w", opts.DestBase, err)
		}
		probe := filepath.Join(opts.DestBase, ".cardbot_probe")
		if f, err := os.Create(probe); err != nil {
			return nil, fmt.Errorf("destination %s is not writable: %w", opts.DestBase, err)
		} else {
			f.Close()
			os.Remove(probe)
		}
	}

	// --- Phase 2: Copy ---
	buf := make([]byte, opts.BufferKB*1024)
	var bytesDone int64
	start := time.Now()
	madeDir := make(map[string]bool, 32)

	for i := range files {
		f := &files[i]
		destPath := filepath.Join(opts.DestBase, f.date, f.relPath)
		// Guard against path traversal via malicious card paths.
		destPath = filepath.Clean(destPath)
		if !strings.HasPrefix(destPath, filepath.Clean(opts.DestBase)+string(filepath.Separator)) {
			return nil, fmt.Errorf("refusing to write outside destination: %s", destPath)
		}

		if onProgress != nil {
			onProgress(Progress{
				FilesDone:   i,
				FilesTotal:  len(files),
				BytesDone:   bytesDone,
				BytesTotal:  totalBytes,
				CurrentFile: f.relPath,
			})
		}

		if err := copyFile(destPath, f.srcPath, f.size, buf, madeDir); err != nil {
			return nil, fmt.Errorf("copying %s: %w", f.relPath, err)
		}

		bytesDone += f.size
	}

	// Final progress
	if onProgress != nil {
		onProgress(Progress{
			FilesDone:  len(files),
			FilesTotal: len(files),
			BytesDone:  bytesDone,
			BytesTotal: totalBytes,
		})
	}

	return &Result{
		FilesCopied: len(files),
		BytesCopied: bytesDone,
		Elapsed:     time.Since(start),
		DestPath:    opts.DestBase,
	}, nil
}

// copyFile copies a single file with size verification.
// If the destination already exists with the correct size, it is skipped.
// madeDir caches directories already created to avoid redundant MkdirAll syscalls.
func copyFile(dst, src string, srcSize int64, buf []byte, madeDir map[string]bool) error {
	// Skip if destination already exists with correct size (re-copy / resume).
	if info, err := os.Stat(dst); err == nil && info.Size() == srcSize {
		return nil
	}
	dir := filepath.Dir(dst)
	if !madeDir[dir] {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		madeDir[dir] = true
	}

	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()

	df, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := df.Close(); err == nil {
			err = closeErr
		}
	}()

	n, err := io.CopyBuffer(df, sf, buf)
	if err != nil {
		os.Remove(dst)
		return err
	}

	if n != srcSize {
		os.Remove(dst)
		return fmt.Errorf("size mismatch: wrote %d, expected %d", n, srcSize)
	}

	return err
}
