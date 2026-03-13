# CardBot — TODO

## Current Version: 0.1.9

Detection, analysis, EXIF, config, UI polish, copy with robustness, UX improvements,
bug fixes, and code health refactor complete. 105 tests across 8 packages, all passing
with `-race`.

**Target: 0.2.0 — Daily Driver.** The version you hand to another photographer and say "try this."

---

## Quick Fixes (land in any release)

- [x] Add "OM System" to `BrandColor` in `ui/color.go` — `cleanGear` already maps
      `"OM DIGITAL"` → `"OM System"` but the color map only has `"Olympus"` → cyan
- [x] `go build -ldflags="-s -w"` — strips debug info, saves ~1.8MB on binary (Makefile added)

---

## 0.1.8 — Code Health

Cleanup pass with verified, real issues from four model reviews cross-checked
against the actual codebase. Refactor to make the codebase testable and clean
before adding selective copy features.

### Split main.go (~995 lines)

- [x] **`main.go`** — flag parsing, config, logger setup, signal handling, `main()`
- [x] **`app.go`** — `app` struct, event loop, card/queue management, `handleInput`
- [x] **`display.go`** — `printCardInfo`, `printInvalidCardInfo`, `printPrompt`, `showHelp`,
  `showHardwareInfo`, `friendlyErr`
- [x] **`copy_cmd.go`** — `copyAll` method, `runSpeedTest`

### Other Refactors

- [x] Extract `printCardHeader` helper — DRY between `printCardInfo` and `printInvalidCardInfo`
- [x] Add `context.Context` to `displayCard` and analyzer — enables clean cancellation
      when card is removed mid-scan
- [x] Log walk errors instead of swallowing them — permission/IO errors now collected
      as warnings in `Result.Warnings` (both analyze and copy packages)
- [x] Standardize `friendlyErr` for all user-facing errors — dotfile write warning,
      config load errors, speed test errors all routed through `friendlyErr`
- [x] Validate destination path — empty destination check at copy start with clear message
- [x] Move `FormatBytes` to platform-agnostic file — `detect/format.go` with no build
      constraints, compiles on all platforms
- [x] Remove 500ms `displayCard` delay — analysis starts immediately on card detection

### Test Additions

- [x] `TestAnalyze_ContextCancelled` — verifies analyzer respects context cancellation
- [x] `TestFormatBytes` moved to `format_test.go` — runs on all platforms (was darwin/linux only)

| Package | Coverage | Notes |
|---------|----------|-------|
| analyze | covered | Context cancellation test added |
| config | covered | Existing tests sufficient |
| copy | covered | Walk warnings added to Result |
| detect | covered | FormatBytes tests now platform-agnostic |
| dotfile | covered | Existing tests sufficient |
| log | covered | Existing tests sufficient |
| ui | covered | Existing tests sufficient |
| main | 0% | Blocked on integration testing — requires real card hardware |
| pick | 0% | macOS-only osascript — skip |
| speedtest | 0% | Needs real filesystem — skip |

---

## 0.1.9 — Selective Copy

Core feature: let users copy subsets of a card instead of everything.
See [docs/SELECTIVE-COPY.md](SELECTIVE-COPY.md) for the full design spec.

- [x] `[s]` Copy Selects — copy starred/picked files only (XMP rating > 0)
- [x] `[p]` Copy Photos — copy photo files only (RAW + JPEG, no video)
- [x] `[v]` Copy Videos — copy video files only (MOV, MP4, MXF, etc.)
- [x] Dotfile v2: `copies` array — one entry per mode, upsert on re-run
- [x] Dotfile v1 → v2 migration on read
- [x] Status line reflects partial copy (`Selects copied on ...`)
- [x] Session guard per mode — `copiedModes map[string]bool` replaces `copied bool`
- [x] "All" supersedes selective modes in guard and display
- [x] Empty filter guard (0 starred, 0 photos, 0 videos → message)
- [x] Disk space preflight scoped to selected file subset
- [x] Help removes strikethrough from `[s]`, `[p]`, `[v]`
- [x] Analyzer: `FileRatings map[string]int` for per-file star data
- [x] Analyzer: export `IsPhoto(ext)` and `IsVideo(ext)` helpers
- [x] Copy engine: `Filter func(relPath, ext string) bool` in Options
- [x] Extract `copyFiltered(card, mode, filter)` from `copyAll`

### Dotfile Design — Resolved

All design decisions are documented in [SELECTIVE-COPY.md](SELECTIVE-COPY.md):

- [x] Schema: `copies` array with one entry per mode (upsert on re-run)
- [x] Status logic: "all" supersedes → "Copied on ..."; otherwise list modes
- [x] `[a]` only records `"all"` entry — does not modify selective entries
- [x] Re-copy after `[a]`: engine's size-check skip handles it automatically
- [x] `photos + videos ≠ all` — tracked independently

---

## 0.2.0 — Daily Driver

Everything from 0.1.x is solid, tested, and feels intentional.

- [ ] All 0.1.8, 0.1.9 items complete
- [ ] Single-key input (raw terminal mode, no Enter required)
- [ ] Selective copy fully implemented with correct status tracking
- [ ] Partial copy state in dotfile — multi-mode copy history
- [ ] No known crashes or data loss scenarios
- [ ] Tested on personal gear across multiple shooting days
- [ ] Feedback from at least one other photographer
- [ ] README reflects actual current behavior
- [ ] First public-facing release candidate

---

## 0.3.0 — Linux Support

- [ ] Linux detection (polling-based, /run/media, /media, /mnt)
- [ ] Linux hardware info (sysfs, CID parsing)
- [ ] Linux speed test
- [ ] Linux eject (udisksctl / umount)
- [ ] Real-world testing (Ubuntu, Fedora, Debian)
- [ ] Stable build and CI

---

## Wishlist

Not on the immediate roadmap. Nice-to-have for someday.

- Estimated time remaining during copy
- Show current filename during copy (deferred to renaming milestone)
- Per-file copy logging (forensic/recovery audit trail)
- Single-key input (raw terminal mode, no Enter required) → promoted to 0.2.0
- Auto-update: check GitHub Releases for new version at startup
- Network destination support
- Windows support
- JSON output mode for scripting
- Star rating filters: `[2]` Copy 2★+, `[3]` Copy 3★+, `[4]` Copy 4★+, `[5]` Copy 5★ only
- File renaming on copy (date-based, camera+date, sequence numbering)
- Resume interrupted copies
- Video metadata (duration, resolution)

---

## Won't Fix

Items raised in code reviews that were investigated and rejected.

| Item | Why |
|------|-----|
| `lastUpdate` race in copy progress | Not a race — only the copy goroutine reads/writes it |
| `cardInvalid` naming ("negative name") | Reads fine: `if a.cardInvalid`. `hasDCIM` would be worse |
| Queue can grow unbounded | Photographers don't have 10 card readers. Never happens |
| Input channel size 10 vs 1 | Works fine with `drainInput()`. Not worth changing |
| FAT32 dotfile atomicity | Rename is atomic for metadata. Non-issue |
| XMP buffer too small/large | 256KB is correct for camera RAW headers |
| God object / extract pure functions | Premature abstraction. `app` struct is manageable |
| Version constant should be typed | Idiomatic Go. `const version = "0.1.8"` is correct |
| Log file needs fsync | CLI tool log doesn't need fsync on every write |
| `printf` vs `fmt.Printf` inconsistent | Actually consistent: `a.printf` = print+log, `fmt.Printf` = transient output |
| `FormatBytes` duplication in copy | Already fixed in 0.1.7 — copy imports `detect.FormatBytes` |
| Detector channels unbuffered | Wrong — they're buffered at size 10 |

---

## Review History

This file consolidates findings from four independent code reviews (Claude, Kimi,
MiniMax, GLM) conducted on 2026-03-12, cross-checked against the actual codebase
on 2026-03-13. Stale items (already fixed in 0.1.7) were removed. Disagreements
were resolved by reading the code. The individual review files have been retired.

Code health refactor (0.1.8) completed on 2026-03-13: main.go split into 4 files,
context threading, walk error logging, friendlyErr standardization, FormatBytes
moved to platform-agnostic file, 500ms delay removed, destination validation added.
