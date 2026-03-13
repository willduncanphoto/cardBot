# Roadmap

Work items grouped by milestone.

---

## Completed

### 0.1.0 — Detection
- [x] macOS native detection (CGO + DiskArbitration)
- [x] macOS fallback detection (polling, no Xcode)
- [x] DCIM folder detection
- [x] Brand guess from folder names
- [x] Basic display (path, storage, brand)
- [x] `[e] Eject` and `[x] Exit` actions
- [x] Card queue for multiple cards
- [x] Graceful shutdown

### 0.1.1 — Card Analysis
- [x] Walk DCIM tree recursively
- [x] Skip hidden files: `.*`, `._*`, `.DS_Store`, `.Trashes`
- [x] Group files by date
- [x] Count files per date, sum sizes
- [x] List file extensions per date
- [x] Fixed-width column formatting
- [x] Handle empty cards gracefully
- [x] Summary line with totals

### 0.1.2 — EXIF, Gear & Hardware
- [x] Add `evanoberholster/imagemeta` dependency
- [x] Extract camera model from EXIF
- [x] Display camera model
- [x] Use `DateTimeOriginal` for date grouping
- [x] Extract star ratings from XMP
- [x] Display starred count
- [x] Photo/video split in totals
- [x] Hardware info query via IOKit (macOS)
- [x] Raw device size vs filesystem size
- [x] `[i]` key for hardware info (hidden command)

### 0.1.3 — Config & Destination
- [x] Config file: `~/.config/cardbot/config.json`
- [x] First-run setup — prompt for destination
- [x] Store: default destination path
- [x] CLI flags: `--dest`, `--version`, `--dry-run`, `--setup`, `--reset`
- [x] Logging: `~/.cardbot/cardbot.log` (5MB rotation)

### 0.1.4 — UI Polish
- [x] Merge brand + camera into single `Camera:` line
- [x] Clean brand names ("NIKON Z 9" → "Nikon Z 9")
- [x] Brand colors (ANSI): Nikon yellow, Canon red, Sony white, etc.
- [x] Card status line (New / Copied via `.cardbot` dotfile)
- [x] Parallel EXIF workers (4 default, 3.7x faster)
- [x] Progress callback throttled to every 100 files
- [x] Config save skipped when destination unchanged
- [x] Native macOS folder picker via `osascript`
- [x] `~` shorthand for paths (display and storage)
- [x] `[t]` hidden speed test command

### 0.1.5 — Copy & Hardening
- [x] `[a] Copy All` mode
- [x] Dated folders: `YYYY-MM-DD/<original-structure>`
- [x] Buffered copy with configurable buffer size
- [x] Progress updates every 2 seconds
- [x] Size verification after each file
- [x] `.cardbot` dotfile written on success
- [x] "Copied on" status on re-insert
- [x] Destination write probe on first creation
- [x] Dry-run aware (no side effects)
- [x] Post-copy prompt: `[e] Eject  [x] Done`
- [x] File collision skip (same size = skip)
- [x] Color output respects `config.output.color`
- [x] Input drain after blocking operations (copy, speed test)
- [x] Pre-compiled regexes for hardware detection
- [x] Fixed double-timestamp in log output
- [x] AppleScript path escaping in folder picker
- [x] `ExpandPath` rejects `~user/` syntax
- [x] `ContractPath` handles trailing slashes
- [x] `displayCard` race fix — re-checks current card after analysis
- [x] Eject error re-shows prompt
- [x] Stale `lastResult` cleared on card removal
- [x] Test suite: 81 tests across 6 packages

### 0.1.6 — Copy Robustness & UX
- [x] Cancel during copy (`[\] Cancel Copy` key)
- [x] Card removal mid-copy — detected and handled gracefully
- [x] Ctrl+C during copy — clean shutdown
- [x] Disk space preflight check
- [x] Read-only card warnings
- [x] Output mutex for concurrent progress/scan output
- [x] Path traversal guard on copy destinations
- [x] File handle leak fix (named return for close errors)
- [x] Goroutine leak fix (input reader shutdown)
- [x] Invalid card handling (no DCIM → friendly message, eject/exit only)
- [x] Help command (`[?]`) with full key reference
- [x] Unknown input feedback
- [x] Friendly error messages (disk full, permission denied, I/O errors)
- [x] Key remapping: `x` = Exit, `\` = Cancel Copy, stubs for `s`/`p`/`v`
- [x] Copy Selects, Copy Photos, Copy Videos shown in help (strikethrough — coming soon)
- [x] Test suite: 97 tests across 6 packages, all passing with `-race`

### 0.1.7 — Bug Fixes & Data Integrity
- [x] Fix stale `cardInvalid` on card removal (queued card inherited wrong state)
- [x] Fix event loop blocking during removal delay (runs in goroutine now)
- [x] Remove dead `isCurrentCard` check after analyze error
- [x] Add `df.Sync()` in `copyFile` for data integrity on removable media
- [x] Remove duplicate `fmtBytes` from copy package → uses `detect.FormatBytes`
- [x] Startup animation: dots appear one at a time over ~1.8s
- [x] Scanning spinner: classic `| / - \` with clean goroutine lifecycle
- [x] Timestamps: ISO format with T separator everywhere
- [x] Improved `readExif` Make/Model dedup (handles "NIKON CORPORATION" + "NIKON Z 9")
- [x] Updated 7 dependencies to latest
- [x] go.mod bumped to go 1.25.0
- [x] Test suite: 100 tests across 8 packages, all passing with `-race`

### 0.1.8 — Code Health
- [x] Split `main.go` into `main.go`, `app.go`, `display.go`, `copy_cmd.go`
- [x] Extract `printCardHeader` helper (DRY)
- [x] Add `context.Context` to `displayCard` and analyzer — clean cancellation on card removal
- [x] Log walk errors instead of silently swallowing (warnings in Result)
- [x] Standardize `friendlyErr` for all user-facing errors
- [x] Validate destination path at copy start
- [x] Move `FormatBytes` to platform-agnostic file (`detect/format.go`)
- [x] Remove 500ms `displayCard` delay — analysis starts immediately
- [x] Context cancellation test added
- [x] Test suite: 101 tests across 8 packages, all passing with `-race`

### 0.1.9 — Selective Copy
- [x] `[s]` Copy Selects — copy starred/picked files only (XMP rating > 0)
- [x] `[p]` Copy Photos — copy photo files only (RAW + JPEG, no video)
- [x] `[v]` Copy Videos — copy video files only (MOV, MP4, MXF, etc.)
- [x] Dotfile v2 with multi-mode array schema
- [x] Status line tracks individual and combined modes (Photos + Videos copied on...)
- [x] Session guard logic to prevent double copies of selected subsets
- [x] Disk space preflight naturally scoped through filters

---

## Upcoming

### 0.2.0 — Testing
- [x] All 0.1.x items complete
- [x] Selective copy fully implemented with correct status tracking
- [x] Partial copy state in dotfile — multi-mode copy history
- [x] No known crashes or data loss scenarios
- [x] README reflects actual current behavior

Real-world testing with the Z9. Identify workflow friction and fine-tune based on actual use.

### 0.3.0 — Linux Support
- [ ] Linux detection (polling-based, /run/media, /media, /mnt)
- [ ] Linux hardware info (sysfs, CID parsing for direct SD slots)
- [ ] Linux speed test support
- [ ] Linux eject (udisksctl / umount)
- [ ] Real-world testing (Ubuntu, Fedora, Debian)
- [ ] Stable build and CI

---

## Wishlist

- Single-key input (raw terminal mode, no Enter required)
- Estimated time remaining during copy
- Show current filename during copy (deferred to renaming milestone)
- Per-file copy logging (forensic/recovery audit trail)
- Auto-update: check GitHub Releases for new version at startup
- Network destination support
- Windows support
- JSON output mode for scripting
- Star rating filters: `[2]` Copy 2★+, `[3]` Copy 3★+, `[4]` Copy 4★+, `[5]` Copy 5★ only
- File renaming on copy (date-based, camera+date, sequence numbering)
- Resume interrupted copies
- Video metadata (duration, resolution)
