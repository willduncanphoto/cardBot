package copy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// makeTestFiles creates n files of the given size in srcDir/DCIM/100TEST/.
func makeTestFiles(b *testing.B, srcDir string, count int, sizeBytes int64) {
	b.Helper()
	dir := filepath.Join(srcDir, "DCIM", "100TEST")
	if err := os.MkdirAll(dir, 0755); err != nil {
		b.Fatal(err)
	}
	data := make([]byte, sizeBytes)
	for i := range data {
		data[i] = byte(i % 251) // non-zero pattern
	}
	for i := 0; i < count; i++ {
		path := filepath.Join(dir, fmt.Sprintf("IMG_%04d.CR3", i))
		if err := os.WriteFile(path, data, 0644); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCopySmallFiles benchmarks copying many small files (like JPEGs).
// 500 x 5MB = 2.5 GB logical, tests per-file overhead (mkdir cache, stat skip, fsync).
func BenchmarkCopySmallFiles(b *testing.B) {
	srcDir := b.TempDir()
	makeTestFiles(b, srcDir, 500, 5*1024*1024) // 500 x 5MB

	for b.Loop() {
		dstDir := b.TempDir()
		opts := Options{
			CardPath: srcDir,
			DestBase: dstDir,
			BufferKB: 256,
		}
		_, err := Run(context.Background(), opts, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCopyLargeFiles benchmarks copying fewer large files (like RAW/video).
// 20 x 50MB = 1 GB logical, tests sustained throughput.
func BenchmarkCopyLargeFiles(b *testing.B) {
	srcDir := b.TempDir()
	makeTestFiles(b, srcDir, 20, 50*1024*1024) // 20 x 50MB

	for b.Loop() {
		dstDir := b.TempDir()
		opts := Options{
			CardPath: srcDir,
			DestBase: dstDir,
			BufferKB: 256,
		}
		_, err := Run(context.Background(), opts, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCopyFileSync isolates the cost of fsync per file.
// Copies a single 10MB file repeatedly to measure fsync overhead.
func BenchmarkCopyFileSync(b *testing.B) {
	srcDir := b.TempDir()
	srcFile := filepath.Join(srcDir, "test.dat")
	data := make([]byte, 10*1024*1024)
	for i := range data {
		data[i] = byte(i % 251)
	}
	if err := os.WriteFile(srcFile, data, 0644); err != nil {
		b.Fatal(err)
	}

	buf := make([]byte, 256*1024)
	dstDir := b.TempDir()

	for b.Loop() {
		dst := filepath.Join(dstDir, fmt.Sprintf("out_%d.dat", b.N))
		madeDir := map[string]bool{dstDir: true}
		if err := copyFile(dst, srcFile, int64(len(data)), buf, madeDir); err != nil {
			b.Fatal(err)
		}
		os.Remove(dst) // clean up for next iteration
	}
}

// BenchmarkBufferSize compares different buffer sizes for copy throughput.
func BenchmarkBufferSize(b *testing.B) {
	srcDir := b.TempDir()
	makeTestFiles(b, srcDir, 10, 50*1024*1024) // 10 x 50MB = 500MB

	for _, kb := range []int{64, 128, 256, 512, 1024, 4096} {
		b.Run(fmt.Sprintf("%dKB", kb), func(b *testing.B) {
			for b.Loop() {
				dstDir := b.TempDir()
				opts := Options{
					CardPath: srcDir,
					DestBase: dstDir,
					BufferKB: kb,
				}
				_, err := Run(context.Background(), opts, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
