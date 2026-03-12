# CardBot Development Plan

**Generated:** 2026-03-12  
**Version:** 0.1.5 → 0.1.6 (Copy Robustness)  
**Focus:** Code quality, stability, and user experience improvements

---

## Executive Summary

This document consolidates findings from a comprehensive codebase review covering architecture, code quality, testing, performance, security, and user experience. The plan prioritizes critical stability issues before adding new features.

**Current State:**
- 3,500 lines Go across 20 files
- 27.7% test coverage (main.go: 0%, detect: 11.5%)
- 4 critical race conditions/resource leaks
- Solid architecture with good package separation
- Zero external runtime dependencies

---

## Critical Issues (Must Fix Before 0.1.6)

### 1. Race Condition in displayCard
**File:** `main.go:276-279`
**Severity:** HIGH

```go
// PROBLEM: Goroutine accesses shared state without proper sync
go func() {
    time.Sleep(500 * time.Millisecond)
    a.displayCard(card)  // Card may be removed during sleep
}()
```

**Impact:** TOCTOU race between card detection and display
**Fix:** Pass card data by value or use context cancellation

### 2. File Handle Leak in Copy
**File:** `internal/copy/copy.go:214-222`
**Severity:** HIGH

```go
// PROBLEM: Source file not closed if CopyBuffer fails early
n, err := io.CopyBuffer(df, sf, buf)
if closeErr := df.Close(); err == nil {
    err = closeErr
}
if err != nil {
    os.Remove(dst)
    return err  // sf not closed!
}
```

**Impact:** Resource exhaustion on repeated copy failures
**Fix:** Both files need defer close with proper error tracking

### 3. Goroutine Leak - Input Reader
**File:** `main.go:698-707`
**Severity:** HIGH

```go
// PROBLEM: No cancellation mechanism
func readInput(ch chan<- string) {
    reader := bufio.NewReader(os.Stdin)
    for {
        line, err := reader.ReadString('\n')  // Blocks forever
        // ...
    }
}
```

**Impact:** Goroutine never terminates on app shutdown
**Fix:** Add done channel or context cancellation

### 4. Path Traversal Vulnerability
**File:** `internal/copy/copy.go:150`
**Severity:** HIGH

```go
// PROBLEM: No sanitization of card-derived paths
dstDir := filepath.Join(opts.DestBase, f.date, filepath.Dir(f.relPath))
```

**Impact:** Malicious card could write outside destination
**Fix:** Add `filepath.Clean()` and validate final path

---

## High Priority Issues

### 5. Global Detector Race Condition
**File:** `internal/detect/detect_darwin.go:87-90`
**Severity:** HIGH

C callbacks race with `Start()`/`Stop()`. Nil dereference possible if detector stopped during callback.

### 6. Unbounded Hardware Info Goroutines
**File:** `internal/detect/shared.go:36-40`
**Severity:** MEDIUM

Each card detection spawns a goroutine for hardware info without lifecycle management.

### 7. Copy Buffer Too Small
**File:** `internal/copy/copy.go:57-58`
**Severity:** MEDIUM

Default 256KB buffer is suboptimal for modern storage (CFexpress/SSD).

### 8. No Disk Space Validation
**File:** `internal/copy/copy.go`
**Severity:** MEDIUM

Copy starts without checking destination space availability.

---

## 0.1.6 Implementation Phases

### Phase 1: Critical Fixes (Priority: P0)
**Timeline:** Week 1

- [ ] **Race Condition Fix**
  - Add context cancellation pattern to `displayCard`
  - Ensure atomic state checks
  - File: `main.go`

- [ ] **File Handle Leak Fix**
  - Restructure copy loop with proper defer closes
  - Track separate errors for src/dst
  - File: `internal/copy/copy.go`

- [ ] **Input Reader Cleanup**
  - Add `done` channel to `readInput`
  - Proper goroutine termination on shutdown
  - File: `main.go`

- [ ] **Path Sanitization**
  - Add `filepath.Clean()` validation
  - Verify paths stay within destination
  - File: `internal/copy/copy.go`

### Phase 2: Copy Robustness (Priority: P1)
**Timeline:** Week 2

- [ ] **Disk Space Pre-flight Check**
  - Calculate total card size before copy
  - Show helpful error if insufficient space
  - File: `internal/copy/copy.go`

- [ ] **Cancel During Copy**
  - Accept 'q' or Ctrl+C during copy
  - Clean up partial files on cancel
  - File: `main.go`, `internal/copy/copy.go`

- [ ] **Handle Card Removal Mid-Copy**
  - Detect card removal during copy operation
  - Graceful cleanup and error reporting
  - File: `internal/copy/copy.go`

- [ ] **File Collision Logic**
  - Skip files if dest exists and size matches
  - Log skipped files
  - File: `internal/copy/copy.go`

- [ ] **Read-Only Card Handling**
  - Detect read-only cards before copy
  - Warn that dotfile can't be written
  - Files: `main.go`, `internal/dotfile/dotfile.go`

### Phase 3: UX Improvements (Priority: P2)
**Timeline:** Week 3

- [ ] **Add Help Command**
  - `[?]` shows all available commands
  - Include hidden commands (i, t)
  - File: `main.go`

- [ ] **Better Error Messages**
  - Add "What to do next" guidance
  - Replace technical jargon ("walking DCIM")
  - File: `main.go`, all error returns

- [ ] **Empty Card Handling**
  - Show friendly message for no-DCIM cards
  - Don't offer "Copy All" on empty cards
  - File: `main.go`, `internal/analyze/analyze.go`

- [ ] **Copy Progress Improvements**
  - Show current filename being copied *(deferred to renaming milestone — filenames are meaningless until rename patterns are implemented)*
  - Reduce update interval to 500ms or per-file
  - Pre-scan for existing files
  - File: `main.go`, `internal/copy/copy.go`

- [ ] **Input Feedback**
  - Show message for invalid commands
  - "Unknown command 'x'. Press ? for help."
  - File: `main.go`

### Phase 4: Testing (Priority: P2)
**Timeline:** Week 4

- [ ] **Extract Testable Functions from main.go**
  - Separate CLI parsing from app logic
  - Create `runApp(cfg)` function
  - File: `main.go` → new `internal/app` package

- [ ] **Add Concurrency Tests**
  - Test concurrent card events with queue
  - Test `copied` flag race conditions
  - Test `drainInput` behavior
  - File: `main_test.go`

- [ ] **Mock-Based Hardware Tests**
  - Create `HardwareQuerier` interface
  - Test with mock `diskutil`/`system_profiler` output
  - File: `internal/detect/hardware_test.go`

- [ ] **Add Error Condition Tests**
  - Copy with permission denied
  - Analyze with corrupt EXIF
  - Run with card removed mid-operation

---

## Architecture Improvements

### Short Term (0.1.6-0.1.7)

1. **Extract Interfaces**
   ```go
   // internal/detect/detector.go
   type Detector interface {
       Start() error
       Stop()
       Events() <-chan *Card
       Removals() <-chan string
       Eject(path string) error
   }
   ```

2. **Decouple Analyze from Copy**
   - Pass date mappings as simple map instead of `*analyze.Result`
   - Reduces package coupling

3. **Create Types Package**
   ```go
   // internal/types/card.go
   type Card struct { ... }
   type HardwareInfo interface { ... }
   ```

4. **Move FormatBytes**
   - From `internal/detect/shared.go` to `internal/util/format.go`
   - Used by multiple packages

### Medium Term (0.2.0+)

5. **Split main.go**
   - Extract UI/display logic to `internal/ui/prompt.go`
   - Extract event loop to `internal/app/eventloop.go`
   - Target: Reduce main.go from 707 to <300 lines

6. **Platform Feature Parity**
   - Implement `speedtest` for Linux
   - Implement `pick.Folder` for Linux (zenity/kdialog)
   - Document platform limitations clearly

7. **Windows Support**
   - Create `detect_windows.go` with WMI
   - Create `hardware_windows.go` with Win32 APIs
   - Build tags already support this pattern

---

## Security Checklist

### 0.1.6 Must-Have

- [ ] **Path Validation**
  - [ ] Validate `f.date` and `f.relPath` don't contain `..`
  - [ ] Verify final path is within `opts.DestBase`
  - [ ] Validate card path is in known mount points

- [ ] **Command Execution**
  - [ ] Use absolute paths for all external commands (`/usr/bin/diskutil`)
  - [ ] Fix AppleScript escaping in `pick_darwin.go`

- [ ] **Resource Limits**
  - [ ] Add maximum file count check before collection
  - [ ] Validate EXIF workers against reasonable maximum (already clamped to 16)

### 0.1.7 Nice-to-Have

- [ ] **Logging Controls**
  - [ ] Add `--no-log` flag
  - [ ] Document `.cardbot` file contains destination path
  - [ ] Consider path obfuscation for sensitive locations

---

## Testing Roadmap

### Current Coverage

| Package | Lines | Coverage |
|---------|-------|----------|
| dotfile | 108 | 91.3% |
| log | 90 | 85.7% |
| copy | 230 | 84.1% |
| config | 169 | 80.3% |
| analyze | 461 | 66.2% |
| detect | ~700 | 11.5% |
| main | 707 | 0.0% |

### Priority Test Additions

**P0 - Critical**
1. Concurrency tests for `app` struct (4 hours)
2. Error condition tests for copy (2 hours)
3. Extract and test main flag parsing (2 hours)

**P1 - Important**
4. Mock-based hardware info tests (3 hours)
5. Test `app.finishCard()` queue handling (1 hour)
6. Add table-driven tests for `ui.BrandColor` (15 min)

**P2 - Nice to Have**
7. Integration tests for copy workflow (6 hours)
8. Platform-specific detection tests (4 hours)

---

## Performance Optimizations

### Critical

1. **Fix Goroutine Leaks**
   - Input reader: Add cancellation
   - Hardware info: Add context with timeout
   - Deferred scan: Add context cancellation

2. **Buffer Pooling**
   - EXIF workers: Use `sync.Pool` for 256KB buffers
   - Copy operations: Use `sync.Pool` for copy buffers

### High Priority

3. **Buffer Size Optimization**
   - Increase default copy buffer from 256KB to 1-4MB
   - Add validation for minimum buffer size

4. **Context Cancellation**
   - Add context to all long-running operations
   - Enable graceful shutdown and cancellation

### Low Priority

5. **Remove Artificial Delays**
   - Replace UX sleeps with actual ready-state detection
   - Make delays configurable (zero in production)

6. **Sequential vs Parallel Copy**
   - Consider worker pool for small files
   - Keep large files sequential for disk efficiency

---

## UX Improvements

### 0.1.6

- [ ] Add `[?] Help` command showing all options
- [ ] Show feedback for invalid input
- [ ] Handle empty/no-DCIM cards as empty state, not error
- [ ] Add disk space check with clear error message
- [ ] Improve error messages with "What to do next"
- [ ] Add `--no-color` flag and `NO_COLOR` support
- [ ] Show current filename during copy progress
- [ ] Reduce progress update interval to 500ms

### 0.1.7

- [ ] Add `[q] View Queue` command
- [ ] Show copy confirmation for large operations
- [ ] Add TTY detection before emitting color codes
- [ ] Colorblind-friendly indicators (not just color)

---

## Documentation Updates

### README.md

- [ ] Update roadmap with 0.1.6 features
- [ ] Document hidden commands (i, t)
- [ ] Add troubleshooting section

### New Files

- [ ] `docs/SECURITY.md` - Security considerations
- [ ] `docs/TESTING.md` - Testing guidelines
- [ ] `docs/PERFORMANCE.md` - Performance tuning

### Agent Documentation

- [ ] Update `agent/AGENT.md` with current version (0.1.5)
- [ ] Update `agent/ARCHITECTURE.md` with new packages
- [ ] Mark completed roadmap items in `docs/TODO.md`

---

## Risk Assessment

### High Risk

- **Race conditions in main.go** - Could cause crashes or data corruption
- **Path traversal vulnerability** - Security risk with malicious cards
- **No cancel during copy** - Poor UX, potential data loss on force quit

### Medium Risk

- **Low test coverage** - Regressions likely without tests
- **Goroutine leaks** - Resource exhaustion over long runtime
- **No disk space check** - Mid-copy failures waste user time

### Low Risk

- **Platform feature gaps** - Linux users have reduced functionality
- **Missing help command** - Discoverability issues
- **Performance optimizations** - Nice to have, not critical

---

## Success Criteria for 0.1.6

### Functional

- [ ] No race conditions detected by `go test -race`
- [ ] All file handles properly closed (verified via /proc/PID/fd)
- [ ] Cancel during copy works cleanly
- [ ] Card removal during copy handled gracefully
- [ ] Path traversal attempts blocked

### Quality

- [ ] Test coverage increased to 40%+ (from 27.7%)
- [ ] No goroutine leaks on shutdown (verified via pprof)
- [ ] All critical code paths have error handling

### UX

- [ ] Help command shows all options
- [ ] Empty cards show friendly message
- [ ] Error messages are actionable
- [ ] Copy progress shows current file

---

## Notes for Future Development

### 0.1.7 Ideas

- Single-key input (no Enter required)
- Startup under 100ms
- ETA during copy
- Selective copy (photos only, videos only, starred only)

### 0.2.0 Ideas

- File renaming on copy
- Resume interrupted copies
- Video metadata parsing
- Auto-update mechanism

### Windows Support

- WMI for volume monitoring
- Win32 APIs for hardware info
- Consider WSL detection for Linux compatibility layer

---

## Appendix: Quick Reference

### Key Files

| File | Lines | Purpose |
|------|-------|---------|
| `main.go` | 707 | Entry point, event loop, UI |
| `internal/copy/copy.go` | 230 | File copy with verification |
| `internal/analyze/analyze.go` | 461 | DCIM walking, EXIF/XMP parsing |
| `internal/detect/detect_darwin.go` | ~350 | macOS native detection |
| `internal/detect/detect_linux.go` | ~250 | Linux polling detection |

### Build Tags

```go
//go:build darwin && cgo       // macOS with Xcode
//go:build darwin && !cgo      // macOS without Xcode
//go:build linux               // Linux
//go:build !darwin && !linux   // Unsupported
```

### Dependencies

- `github.com/evanoberholster/imagemeta` - EXIF/XMP parsing
- `github.com/cespare/xxhash/v2` - File hashing

---

*This plan is a living document. Update as priorities shift and work progresses.*
