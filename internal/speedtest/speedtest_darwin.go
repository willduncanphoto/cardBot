//go:build darwin

package speedtest

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	testFileSize = 256 * 1024 * 1024 // 256 MB
	bufferSize   = 256 * 1024        // 256 KB chunks
	testFileName = ".cardbot_speedtest"
)

// Result holds the measured read and write speeds.
type Result struct {
	WriteSpeed float64 // MB/s
	ReadSpeed  float64 // MB/s
}

// Run performs a write then read speed test on the given mount path.
// onProgress is called periodically with current MB/s and phase ("Writing" / "Reading").
func Run(mountPath string, onProgress func(phase string, mbps float64)) (*Result, error) {
	testFile := filepath.Join(mountPath, testFileName)

	// Ensure cleanup regardless of outcome.
	defer os.Remove(testFile)

	// --- Write test ---
	writeSpeed, err := measureWrite(testFile, onProgress)
	if err != nil {
		return nil, fmt.Errorf("write test failed: %w", err)
	}

	// --- Read test ---
	readSpeed, err := measureRead(testFile, onProgress)
	if err != nil {
		return nil, fmt.Errorf("read test failed: %w", err)
	}

	return &Result{
		WriteSpeed: writeSpeed,
		ReadSpeed:  readSpeed,
	}, nil
}

func measureWrite(path string, onProgress func(string, float64)) (float64, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	buf := make([]byte, bufferSize)
	// Fill buffer with non-zero pattern so it can't be optimised away.
	for i := range buf {
		buf[i] = 0xAB
	}

	written := 0
	start := time.Now()
	lastReport := start

	for written < testFileSize {
		toWrite := bufferSize
		if written+toWrite > testFileSize {
			toWrite = testFileSize - written
		}
		n, err := f.Write(buf[:toWrite])
		if err != nil {
			return 0, err
		}
		written += n

		if onProgress != nil && time.Since(lastReport) >= 250*time.Millisecond {
			elapsed := time.Since(start).Seconds()
			if elapsed > 0 {
				onProgress("Writing", float64(written)/elapsed/1024/1024)
			}
			lastReport = time.Now()
		}
	}

	// Flush to physical device before stopping the clock.
	if err := f.Sync(); err != nil {
		return 0, err
	}

	elapsed := time.Since(start).Seconds()
	return float64(written) / elapsed / 1024 / 1024, nil
}

func measureRead(path string, onProgress func(string, float64)) (float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	buf := make([]byte, bufferSize)
	read := 0
	start := time.Now()
	lastReport := start

	for {
		n, err := f.Read(buf)
		read += n

		if onProgress != nil && time.Since(lastReport) >= 250*time.Millisecond {
			elapsed := time.Since(start).Seconds()
			if elapsed > 0 {
				onProgress("Reading", float64(read)/elapsed/1024/1024)
			}
			lastReport = time.Now()
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	elapsed := time.Since(start).Seconds()
	return float64(read) / elapsed / 1024 / 1024, nil
}
