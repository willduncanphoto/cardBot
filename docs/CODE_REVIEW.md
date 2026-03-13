# Code Review Notes — v0.1.7

Review date: 2026-03-12. Notes for a future cleanup pass.

---

## main.go (941 lines) — Highest Priority

### Split needed
`main.go` is doing too much: CLI flags, config loading, signal handling, event loop, card display, copy orchestration, prompts, help, input handling, eject, speed test. Should be 3–4 files:

- **`main.go`** — flag parsing, config, logger setup, signal handling, `main()` only (~100 lines)
- **`app.go`** — `app` struct, event loop, card/queue management, `handleInput`
- **`display.go`** — `printCardInfo`, `printInvalidCardInfo`, `printPrompt`, `showHelp`, `showHardwareInfo`, `friendlyErr`
- **`copy_cmd.go`** — `copyAll` method (the 120-line mini event loop)

### `copyAll` is too long (120+ lines)
The copy mini event loop in `copyAll` handles: progress, cancellation, card removal, new card events, input, SIGINT. This is the most complex function in the codebase. Extract the select loop into a helper or use a struct to carry the copy session state.

### `printf` vs `fmt.Printf` inconsistency
Some output goes through `a.printf()` (prints + logs), some through raw `fmt.Printf` (prints only). No clear pattern for which is used when. Decision needed: either log everything meaningful, or drop `printf` and use explicit `fmt.Print` + `a.logf` pairs.

### Duplicate card info rendering
`printCardInfo` (~50 lines) and `printInvalidCardInfo` (~20 lines) share Status/Path/Storage/Camera display code. Extract a `printCardHeader(card)` helper.

### `time.Sleep` in startup
Three `150ms` sleeps in main for visual effect (total 450ms). These slow down startup and will conflict with the "startup <100ms" goal in 0.1.7. Gate behind a `--fast` flag or remove entirely.

### `displayCard` goroutine has no context
`displayCard` runs analysis in a goroutine but can't be cancelled if the card is removed mid-scan. The `isCurrentCard` checks are polling-style guards. Should take a `context.Context` so analysis can be cancelled cleanly.

### Progress callback captures mutable `lastUpdate`
In `copyAll`, the progress callback goroutine captures `lastUpdate` (a `time.Time` on the stack) and mutates it from inside the copy goroutine. This is technically a race since `lastUpdate` is on the main goroutine's stack but only written from the copy goroutine. Works in practice but is fragile. Move into an `atomic.Value` or pass a throttled callback.

---

## internal/copy/copy.go (285 lines) — Clean

### `fmtBytes` duplicates `detect.FormatBytes`
Two identical formatters exist. The copy package avoids importing detect to stay decoupled, which is fine, but they could drift. Options:
1. Extract a tiny `internal/format` package with one function
2. Accept the duplication (it's 10 lines)

### No `fsync` after copy
`copyFile` doesn't call `df.Sync()` before close. On macOS this is usually fine (unified buffer cache), but on Linux with removable media, data could be in the page cache when we report "copy complete". Consider adding `Sync()` before the size check for correctness.

### Walk errors silently swallowed
Both the analyze and copy `WalkDir` callbacks return `nil` on errors (permission denied, broken symlinks, etc.). Files are silently skipped. Consider collecting skipped-file warnings and surfacing them in the result.

---

## internal/analyze/analyze.go (461 lines) — Solid

### EXIF worker pool could use `errgroup`
Manual `sync.WaitGroup` + channel pattern works but `errgroup` would be cleaner and support cancellation. Low priority since it works.

### `readExif` reads 256KB for XMP unconditionally
Even JPEG files (which rarely have XMP) get the 256KB buffer read. Could check extension first and skip XMP scan for non-RAW files. Minor perf optimization.

### `gear` captures first camera model only
If a card has files from multiple cameras (e.g. two-body shooters with shared cards), only the first detected model is shown. Could collect all unique models. Low priority — edge case.

---

## internal/detect/ — Platform-Specific

### `shared.go` has `//go:build darwin || linux`
This means `FormatBytes`, `buildCard`, and `detectBrand` don't compile on Windows/other. `FormatBytes` is used by `main.go` for display — this will break if we ever add Windows support. Move pure utility functions to an unguarded file.

### `detect` coverage is 11.5%
Lowest coverage. Most code is platform-specific (DiskArbitration, IOKit, sysfs). Hard to unit test without mocks. Consider:
- Extract `detectBrand` and `FormatBytes` tests (these are pure functions)
- Hardware functions need integration tests or build-tag-guarded tests

### `containsNDModel` is clever but undocumented inline
The loop-based substring search works but isn't obvious. A one-line comment explaining "rejects STANDARD, ANDROID" would help future readers. (Already in the func doc, just not at the loop level.)

---

## internal/config/config.go (169 lines) — Clean

### No validation of destination path on load
Config accepts any string for `destination.path`. A path like `/dev/null` or empty string would cause confusing errors at copy time. Consider validating at load or at least at copy time with a clear message.

---

## internal/dotfile/dotfile.go (108 lines) — Clean

### Atomic write on FAT32?
The temp-file + rename pattern may not be truly atomic on FAT32/exFAT (common card filesystems). If power is lost mid-write, the temp file could be left behind. Not a real risk (card is mounted and powered by the reader), but worth a note.

---

## internal/log/log.go (90 lines) — Clean

### Only keeps `.old` — one rotation
If the log rotates twice in a session, the first `.old` is lost. For a personal tool this is fine. If needed later, switch to numbered rotation (`cardbot.log.1`, `.2`, etc.).

---

## internal/ui/color.go (25 lines) — Minimal

### No `Dim`, `Bold`, or other styles
The help screen uses raw `\033[9m` for strikethrough inline. If we add more styled output, centralize ANSI codes here.

---

## Cross-Cutting Issues

### No `context.Context` threading
Only `copyAll` uses context. The analyzer, display, and detector have no cancellation support. For 0.1.7+, threading context through from `main` would enable clean shutdown of everything.

### Error handling philosophy unclear
Some errors are fatal (`os.Exit`), some are warnings (`fmt.Fprintf(os.Stderr, ...)`), some are friendly (`friendlyErr`), some are raw (`%v`). Establishing a clear pattern would help:
- Fatal: only in `main()` before event loop starts
- User-facing: always `friendlyErr`
- Log: always raw error

### Test coverage gaps
| Package | Coverage | Notes |
|---------|----------|-------|
| main | 0% | Needs refactor to be testable (extract app logic) |
| detect | 11.5% | Platform code; `detectBrand` + `FormatBytes` testable |
| pick | 0% | macOS-only, osascript — hard to test |
| speedtest | 0% | Side-effectful, needs real card or mock FS |
| ui | 0% | Trivial, but easy to add |

### Binary size
4.9 MB for a CLI tool is chunky. Mostly from `imagemeta` dependency. `go build -ldflags="-s -w"` would drop ~1 MB. UPX could go further if distribution size matters.

---

## Action Items (Priority Order)

1. **Split `main.go`** into 3–4 files — improves readability and makes testing possible
2. **Add `context.Context` to `displayCard`/analyzer** — enables clean cancellation
3. **Add `df.Sync()` before close** in copy — correctness on Linux
4. **Extract `printCardHeader`** — DRY up card info display
5. **Remove startup sleeps** or gate behind flag — blocks 0.1.7 startup goal
6. **Move `FormatBytes` to unguarded file** — unblocks cross-platform
7. **Add `detectBrand` + `FormatBytes` unit tests** — easy coverage wins
8. **Standardize error handling** — `friendlyErr` everywhere user-facing
9. **Consider `internal/format` package** — single `FormatBytes` source of truth
10. **`go build -ldflags="-s -w"`** — free 1 MB binary size reduction
