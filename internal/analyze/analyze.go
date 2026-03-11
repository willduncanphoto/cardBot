// Package analyze handles card content analysis.
// Walks the DCIM tree, groups files by date, collects per-date stats,
// extracts camera model and star ratings via EXIF/XMP.
package analyze

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/evanoberholster/imagemeta"
)

// supportedExif lists extensions we attempt EXIF extraction on (uppercase, no dot).
var supportedExif = map[string]bool{
	"JPG":  true,
	"JPEG": true,
	"NEF":  true,
	"NRW":  true,
	"CR2":  true,
	"CR3":  true,
	"CRW":  true,
	"ARW":  true,
	"SRF":  true,
	"SR2":  true,
	"RAF":  true,
	"ORF":  true,
	"RW2":  true,
	"DNG":  true,
	"PEF":  true,
	"TIFF": true,
	"TIF":  true,
	"HEIC": true,
	"HEIF": true,
}

// DateGroup holds stats for all files on a single date.
type DateGroup struct {
	Date       string   // YYYY-MM-DD
	Size       int64    // Total bytes
	FileCount  int      // Number of files
	Extensions []string // Unique extensions, sorted alphabetically, uppercase
}

// Result contains analysis data for a memory card.
type Result struct {
	Groups     []DateGroup // Newest first
	TotalSize  int64       // Sum of all file sizes
	FileCount  int         // Total number of files
	PhotoSize  int64       // Total bytes of photo files
	PhotoCount int         // Number of photo files
	VideoSize  int64       // Total bytes of video files
	VideoCount int         // Number of video files
	Gear       string      // Camera make + model (e.g. "Nikon Z 9"), empty if unknown
	Starred    int         // Count of files with star rating > 0
}

// videoExts lists extensions classified as video.
var videoExts = map[string]bool{
	"MOV":  true,
	"MP4":  true,
	"AVI":  true,
	"MXF":  true,
	"MTS":  true,
	"M2TS": true,
	"R3D":  true,
	"BRAW": true,
}

// photoExts lists extensions classified as photos (RAW + compressed images).
var photoExts = map[string]bool{
	"NEF":  true,
	"NRW":  true,
	"CR2":  true,
	"CR3":  true,
	"CRW":  true,
	"ARW":  true,
	"SRF":  true,
	"SR2":  true,
	"RAF":  true,
	"ORF":  true,
	"RW2":  true,
	"DNG":  true,
	"PEF":  true,
	"3FR":  true,
	"IIQ":  true,
	"JPG":  true,
	"JPEG": true,
	"TIF":  true,
	"TIFF": true,
	"HEIC": true,
	"HEIF": true,
	"PNG":  true,
}

// ProgressFunc is called periodically during analysis with the current file count.
type ProgressFunc func(count int)

// Analyzer scans a card's DCIM directory.
type Analyzer struct {
	cardPath   string
	onProgress ProgressFunc
}

// New creates a new analyzer for the given card path.
func New(cardPath string) *Analyzer {
	return &Analyzer{cardPath: cardPath}
}

// OnProgress sets a callback invoked during the walk with the running file count.
func (a *Analyzer) OnProgress(fn ProgressFunc) {
	a.onProgress = fn
}

// Analyze walks the DCIM tree and returns content stats grouped by date.
// Extracts camera model from the first supported image, counts star ratings,
// and uses DateTimeOriginal for date grouping when available.
// Returns an empty Result (not nil) if the card has no files.
// Returns an error only if DCIM cannot be read at all.
func (a *Analyzer) Analyze() (*Result, error) {
	dcim := filepath.Join(a.cardPath, "DCIM")
	if _, err := os.Stat(dcim); err != nil {
		return nil, err
	}

	groups := make(map[string]*dateAccumulator)
	var totalSize, photoSize, videoSize int64
	var fileCount, photoCount, videoCount int
	var gear string
	var starred int

	// Reusable buffer for XMP rating scans. Allocated once, reused across all files.
	// Avoids ~256KB allocation per file (was ~762MB total on a 3048-file card).
	xmpBuf := make([]byte, xmpBufSize)

	err := filepath.WalkDir(dcim, func(path string, d os.DirEntry, err error) error {
		if err != nil {
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

		info, err := d.Info()
		if err != nil {
			return nil
		}

		size := info.Size()
		ext := normalizeExt(filepath.Ext(d.Name()))

		// Skip files that aren't photos or videos.
		if ext == "" || (!photoExts[ext] && !videoExts[ext]) {
			return nil
		}

		// Default to file mtime; overwrite with EXIF date if available.
		date := info.ModTime().Format("2006-01-02")

		if supportedExif[ext] {
			if exifDate, exifGear, rating, ok := readExif(path, xmpBuf); ok {
				if !exifDate.IsZero() {
					date = exifDate.Format("2006-01-02")
				}
				if gear == "" && exifGear != "" {
					gear = exifGear
				}
				if rating > 0 {
					starred++
				}
			}
		}

		acc, ok := groups[date]
		if !ok {
			acc = &dateAccumulator{exts: make(map[string]bool)}
			groups[date] = acc
		}
		acc.size += size
		acc.count++
		acc.exts[ext] = true

		totalSize += size
		fileCount++
		if a.onProgress != nil {
			a.onProgress(fileCount)
		}
		if videoExts[ext] {
			videoSize += size
			videoCount++
		} else if photoExts[ext] {
			photoSize += size
			photoCount++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &Result{
		Groups:     buildGroups(groups),
		TotalSize:  totalSize,
		FileCount:  fileCount,
		PhotoSize:  photoSize,
		PhotoCount: photoCount,
		VideoSize:  videoSize,
		VideoCount: videoCount,
		Gear:       gear,
		Starred:    starred,
	}, nil
}

// xmpBufSize is the size of the reusable buffer for XMP rating scans.
// XMP is typically embedded in the first 256KB of RAW files (NEF, CR2, ARW, etc.).
const xmpBufSize = 256 * 1024

// readExif opens a file and extracts date, camera model, and star rating.
// The caller-provided xmpBuf is reused across calls to avoid per-file allocations.
// The file is read once: first xmpBufSize bytes are read into xmpBuf for XMP scanning,
// then the file is seeked back to 0 for EXIF decoding.
// Returns ok=false if EXIF cannot be read (not an error — file is still counted).
func readExif(path string, xmpBuf []byte) (date time.Time, gear string, rating int, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, "", 0, false
	}
	defer f.Close()

	// Read the head of the file for XMP scanning before EXIF decode.
	// This avoids a second read/seek after imagemeta.Decode consumes the reader.
	n, _ := f.Read(xmpBuf)
	xmpData := xmpBuf[:n]

	// Seek back for EXIF decode.
	f.Seek(0, io.SeekStart)

	exif, err := imagemeta.Decode(f)
	if err != nil {
		return time.Time{}, "", 0, false
	}

	// Camera model: combine Make + Model, dedup if Model already contains Make.
	make_ := strings.TrimSpace(exif.Make)
	model := strings.TrimSpace(exif.Model)
	if make_ != "" && model != "" {
		if strings.HasPrefix(strings.ToLower(model), strings.ToLower(make_)) {
			gear = model
		} else {
			gear = make_ + " " + model
		}
	} else if model != "" {
		gear = model
	}

	// Star rating: check EXIF tag first, then scan XMP from the already-read buffer.
	rating = int(exif.Rating)
	if rating == 0 {
		rating = scanXMPRating(xmpData)
	}

	dto := exif.DateTimeOriginal()
	return dto, gear, rating, true
}

// xmpRatingPrefix is the byte sequence before the rating digit in XMP.
var xmpRatingPrefix = []byte("<xmp:Rating>")

// scanXMPRating searches a byte slice for an embedded XMP rating.
// The Nikon Z9 (and others) store star ratings in XMP embedded in the file,
// not in the EXIF Rating tag. Format: <xmp:Rating>N</xmp:Rating>
// Returns 0 if no rating found or rating is 0.
func scanXMPRating(buf []byte) int {
	idx := bytes.Index(buf, xmpRatingPrefix)
	if idx < 0 {
		return 0
	}
	pos := idx + len(xmpRatingPrefix)
	if pos >= len(buf) {
		return 0
	}
	ch := buf[pos]
	if ch >= '1' && ch <= '5' {
		return int(ch - '0')
	}
	return 0
}

// dateAccumulator collects stats while walking.
type dateAccumulator struct {
	size  int64
	count int
	exts  map[string]bool
}

// buildGroups converts the accumulator map into a sorted slice (newest first).
func buildGroups(m map[string]*dateAccumulator) []DateGroup {
	groups := make([]DateGroup, 0, len(m))
	for date, acc := range m {
		exts := make([]string, 0, len(acc.exts))
		for ext := range acc.exts {
			exts = append(exts, ext)
		}
		sort.Strings(exts)

		groups = append(groups, DateGroup{
			Date:       date,
			Size:       acc.size,
			FileCount:  acc.count,
			Extensions: exts,
		})
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Date > groups[j].Date
	})
	return groups
}

// isHidden reports whether a filename should be skipped.
func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

// normalizeExt returns the uppercase extension without the leading dot.
func normalizeExt(ext string) string {
	if ext == "" {
		return ""
	}
	return strings.ToUpper(ext[1:])
}


