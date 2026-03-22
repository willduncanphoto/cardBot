// Package copy handles file copying from memory cards to the destination.
package copy

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/illwill/cardbot/internal/analyze"
	"github.com/illwill/cardbot/internal/detect"
)

// ProgressFunc is called periodically during the copy with current stats.
type ProgressFunc func(stats Progress)

// Progress holds real-time copy stats for the callback.
type Progress struct {
	FilesDone   int
	FilesTotal  int
	BytesDone   int64
	BytesTotal  int64
	CurrentFile string // relative destination path being copied
	SourceFile  string // relative source path (for dry-run rename preview)
}

// Result holds the final outcome of a copy operation.
// On a cancelled or failed copy, FilesCopied/BytesCopied reflect files that
// completed successfully before the interruption.
type Result struct {
	FilesCopied int
	BytesCopied int64
	Elapsed     time.Duration
	DestPath    string
	Warnings    []string // Non-fatal errors encountered during walk (permission, I/O)
}

// Options configures the copy operation.
type Options struct {
	CardPath      string                                // Source card mount point
	DestBase      string                                // Base destination directory (e.g. ~/Pictures/CardBot)
	BufferKB      int                                   // Copy buffer size in KB (default 256)
	DryRun        bool                                  // If true, walk and report but don't copy
	AnalyzeResult *analyze.Result                       // If provided, use EXIF dates/times for folder grouping and naming
	Filter        func(relPath string, ext string) bool // If provided, skip files where func returns false
	NamingMode    string                                // "original" (default) or "timestamp"
}

// fileEntry holds a file to be copied.
type fileEntry struct {
	srcPath     string // absolute source path on card
	relPath     string // relative path from DCIM (e.g. "100NIKON/DSC_0001.NEF")
	size        int64
	date        string    // YYYY-MM-DD for folder grouping
	captureTime time.Time // EXIF capture time (fallback: mtime)
}

// Run executes the copy operation.
// ctx may be cancelled to abort mid-copy; a partial *Result is always returned
// alongside any error so the caller knows how many files completed.
func Run(ctx context.Context, opts Options, onProgress ProgressFunc) (*Result, error) {
	if opts.BufferKB <= 0 {
		opts.BufferKB = 256
	}

	dcim := filepath.Join(opts.CardPath, "DCIM")
	if _, err := os.Stat(dcim); err != nil {
		return nil, fmt.Errorf("no DCIM folder found on card")
	}

	// Build EXIF lookups from analyze result if available.
	var exifDates map[string]string
	var exifDateTimes map[string]time.Time
	if opts.AnalyzeResult != nil {
		exifDates = opts.AnalyzeResult.FileDates
		exifDateTimes = opts.AnalyzeResult.FileDateTimes
	}

	// --- Phase 1: Collect files ---
	var files []fileEntry
	var totalBytes int64
	var walkWarnings []string

	err := filepath.WalkDir(dcim, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Log permission/IO errors but keep walking.
			// Broken symlinks and not-exist are silently skipped.
			if !os.IsNotExist(err) {
				rel, _ := filepath.Rel(dcim, path)
				walkWarnings = append(walkWarnings, fmt.Sprintf("%s: %v", rel, err))
			}
			return nil
		}
		if d.IsDir() {
			if isHidden(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if isHidden(d.Name()) {
			return nil
		}
		// Skip symlinks — only copy real files from the card.
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		rel, _ := filepath.Rel(dcim, path)

		// Apply selective copy filter if provided.
		if opts.Filter != nil {
			// Extract extension (uppercase, no dot)
			ext := filepath.Ext(d.Name())
			if len(ext) > 0 {
				ext = strings.ToUpper(ext[1:])
			}
			if !opts.Filter(rel, ext) {
				return nil
			}
		}

		// Use EXIF date/time if available, fall back to mtime.
		captureTime := info.ModTime()
		date := captureTime.Format("2006-01-02")
		if exifDate, ok := exifDates[rel]; ok {
			date = exifDate
		}
		if exifDateTime, ok := exifDateTimes[rel]; ok && !exifDateTime.IsZero() {
			captureTime = exifDateTime
		}

		files = append(files, fileEntry{
			srcPath:     path,
			relPath:     rel,
			size:        info.Size(),
			date:        date,
			captureTime: captureTime,
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

	// Sort files by capture time for chronological sequence numbering.
	// When shooting bursts, this ensures sequence numbers reflect actual shot order
	// even if files are scattered across multiple DCIM subfolders.
	sortFilesByCaptureTime(files)

	// Compute rename mappings for progress reporting and dry-run preview.
	namingMode := isTimestampMode(opts.NamingMode)
	seqDigits := SequenceDigits(len(files))
	if opts.AnalyzeResult != nil && opts.AnalyzeResult.FileCount > 0 {
		seqDigits = SequenceDigits(opts.AnalyzeResult.FileCount)
	}
	seqMax := sequenceMax(seqDigits)
	seq := 1

	// Pre-compute all destination paths for dry-run and progress reporting.
	destPaths := make([]string, len(files))
	for i := range files {
		f := &files[i]
		destRelPath := f.relPath
		if namingMode {
			destRelPath = renamedRelativePath(f.relPath, f.captureTime, seq, seqDigits)
			seq++
			if seq > seqMax {
				seq = 1 // Loop back to 0001 after 9999
			}
		}
		destPaths[i] = destRelPath
	}

	if opts.DryRun {
		// Report all mappings via progress callback for dry-run preview.
		if onProgress != nil {
			for i, f := range files {
				onProgress(Progress{
					FilesDone:   i,
					FilesTotal:  len(files),
					BytesDone:   0,
					BytesTotal:  totalBytes,
					CurrentFile: destPaths[i],
					SourceFile:  f.relPath, // For dry-run rename preview
				})
			}
		}
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

	// --- Disk space check ---
	// If we can query free space and it's clearly insufficient, fail fast.
	// Note: files already at the destination will be skipped during copy,
	// so this check may be conservative for re-copies.
	if free, ok := diskFreeBytes(opts.DestBase); ok && free < totalBytes {
		return nil, fmt.Errorf("not enough space on destination: need %s, only %s free",
			detect.FormatBytes(totalBytes), detect.FormatBytes(free))
	}

	// --- Phase 2: Copy ---
	buf := make([]byte, opts.BufferKB*1024)
	var bytesDone int64
	var filesDone int
	start := time.Now()
	madeDir := make(map[string]bool, 32)

	for i := range files {
		// Check for cancellation before each file.
		select {
		case <-ctx.Done():
			return partialResult(filesDone, bytesDone, start, opts.DestBase), ctx.Err()
		default:
		}

		f := &files[i]
		destPath := filepath.Join(opts.DestBase, f.date, destPaths[i])

		// Guard against path traversal via malicious card paths.
		destPath = filepath.Clean(destPath)
		if !strings.HasPrefix(destPath, filepath.Clean(opts.DestBase)+string(filepath.Separator)) {
			return partialResult(filesDone, bytesDone, start, opts.DestBase),
				fmt.Errorf("refusing to write outside destination: %s", destPath)
		}

		if onProgress != nil {
			onProgress(Progress{
				FilesDone:   filesDone,
				FilesTotal:  len(files),
				BytesDone:   bytesDone,
				BytesTotal:  totalBytes,
				CurrentFile: destPaths[i],
			})
		}

		if err := copyFile(destPath, f.srcPath, f.size, buf, madeDir); err != nil {
			return partialResult(filesDone, bytesDone, start, opts.DestBase),
				fmt.Errorf("copying %s: %w", f.relPath, err)
		}

		bytesDone += f.size
		filesDone++
	}

	// Final progress
	if onProgress != nil {
		onProgress(Progress{
			FilesDone:  filesDone,
			FilesTotal: len(files),
			BytesDone:  bytesDone,
			BytesTotal: totalBytes,
		})
	}

	return &Result{
		FilesCopied: filesDone,
		BytesCopied: bytesDone,
		Elapsed:     time.Since(start),
		DestPath:    opts.DestBase,
		Warnings:    walkWarnings,
	}, nil
}

// partialResult builds a Result from in-progress counters.
func partialResult(files int, bytes int64, start time.Time, dest string) *Result {
	return &Result{
		FilesCopied: files,
		BytesCopied: bytes,
		Elapsed:     time.Since(start),
		DestPath:    dest,
	}
}

// copyFile copies a single file with size verification and atomic rename.
// Writes to a temporary .part file, syncs, then renames to the final path.
// If the destination already exists with the correct size, it is skipped.
// madeDir caches directories already created to avoid redundant MkdirAll syscalls.
func copyFile(dst, src string, srcSize int64, buf []byte, madeDir map[string]bool) (err error) {
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

	// Write to a temporary .part file to avoid exposing half-written files.
	partPath := dst + ".part"
	df, err := os.Create(partPath)
	if err != nil {
		return err
	}
	defer func() {
		// On any error, close and remove the temp file.
		if err != nil {
			df.Close()
			os.Remove(partPath)
		}
	}()

	n, err := io.CopyBuffer(df, sf, buf)
	if err != nil {
		return err
	}

	if n != srcSize {
		return fmt.Errorf("size mismatch: wrote %d, expected %d", n, srcSize)
	}

	// Flush to stable storage before rename.
	if err := df.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	if err := df.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	// Atomic rename: .part → final path.
	if err := os.Rename(partPath, dst); err != nil {
		os.Remove(partPath)
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

// sortFilesByCaptureTime sorts files chronologically by capture time.
// Falls back to lexicographic path order if capture times are equal.
func sortFilesByCaptureTime(files []fileEntry) {
	sort.SliceStable(files, func(i, j int) bool {
		if files[i].captureTime.Equal(files[j].captureTime) {
			return files[i].relPath < files[j].relPath
		}
		return files[i].captureTime.Before(files[j].captureTime)
	})
}

// isHidden reports whether a filename should be skipped.
func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}
