// Package log provides simple file-based logging with size-based rotation.
package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const maxSize = 5 * 1024 * 1024 // 5 MB

// Logger writes log lines to a file, rotating when it exceeds maxSize.
type Logger struct {
	mu      sync.Mutex
	path    string
	f       *os.File
	written int64 // bytes written since open, avoids Stat() on every line
}

// Open opens (or creates) the log file at path.
func Open(path string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("creating log directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}
	// Seed written from current file size so rotation works on restart.
	var written int64
	if info, err := f.Stat(); err == nil {
		written = info.Size()
	}
	return &Logger{path: path, f: f, written: written}, nil
}

// Printf writes a formatted log line with a timestamp prefix.
func (l *Logger) Printf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	line := fmt.Sprintf("[%s] %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		fmt.Sprintf(format, args...))

	if l.written >= maxSize {
		l.rotate()
	}

	n, _ := l.f.WriteString(line)
	l.written += int64(n)
}

// Raw writes a pre-formatted line to the log (no timestamp added).
func (l *Logger) Raw(line string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.written >= maxSize {
		l.rotate()
	}

	n, _ := l.f.WriteString(line + "\n")
	l.written += int64(n)
}

// Close flushes and closes the log file.
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f != nil {
		_ = l.f.Close()
		l.f = nil
	}
}

// rotate renames the current log to .old and opens a fresh file.
func (l *Logger) rotate() {
	_ = l.f.Close()
	_ = os.Rename(l.path, l.path+".old")
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		l.f = nil
		return
	}
	l.f = f
	l.written = 0
}
