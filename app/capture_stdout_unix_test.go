//go:build !windows

package app

import (
	"bytes"
	"io"
	"os"
	"sync"
	"syscall"
	"testing"
)

var captureStdoutMu sync.Mutex

func captureStdoutFD(t *testing.T, fn func()) string {
	t.Helper()

	captureStdoutMu.Lock()
	defer captureStdoutMu.Unlock()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()

	stdoutFD := int(os.Stdout.Fd())
	oldFD, err := syscall.Dup(stdoutFD)
	if err != nil {
		t.Fatalf("dup stdout fd: %v", err)
	}
	restored := false
	defer func() {
		if restored {
			return
		}
		_ = syscall.Dup2(oldFD, stdoutFD)
		_ = syscall.Close(oldFD)
	}()

	if err := syscall.Dup2(int(w.Fd()), stdoutFD); err != nil {
		_ = syscall.Close(oldFD)
		t.Fatalf("redirect stdout fd: %v", err)
	}

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	_ = w.Close()
	if err := syscall.Dup2(oldFD, stdoutFD); err != nil {
		_ = syscall.Close(oldFD)
		t.Fatalf("restore stdout fd: %v", err)
	}
	restored = true
	_ = syscall.Close(oldFD)

	return <-done
}
