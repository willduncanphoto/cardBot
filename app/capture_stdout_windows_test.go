//go:build windows

package app

import (
	"bytes"
	"io"
	"os"
	"sync"
	"testing"
)

var captureStdoutMu sync.Mutex

func captureStdoutFD(t *testing.T, fn func()) string {
	t.Helper()

	captureStdoutMu.Lock()
	defer captureStdoutMu.Unlock()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}
