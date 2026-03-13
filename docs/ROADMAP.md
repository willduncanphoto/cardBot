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

---

## 0.1.7 — Bug Fixes & Data Integrity ✓
- [x] Fix stale `cardInvalid` on card removal (queued card inherited wrong state)
- [x] Fix event loop blocking during removal delay (2s unresponsive)
- [x] Remove dead `isCurrentCard` check after analyze error
- [x] Add `df.Sync()` in `copyFile` for data integrity on removable media

---

## Upcoming

---

## Wishlist

- Estimated time remaining during copy
- Show current filename during copy (deferred to renaming milestone)
- Per-file copy logging (forensic/recovery audit trail)
- Single-key input (raw terminal mode, no Enter required)
- Auto-update: check GitHub Releases for new version at startup, `--update` flag
- Network destination support
- Windows support
- JSON output mode for scripting
- Star rating filters: `[2]` Copy 2★+, `[3]` Copy 3★+, `[4]` Copy 4★+, `[5]` Copy 5★ only

---

### 0.1.8 — Selective Copy
- [ ] `[s]` Copy Selects — copy starred/picked files only (XMP rating > 0)
- [ ] `[p]` Copy Photos — copy photo files only (RAW + JPEG, no video)
- [ ] `[v]` Copy Videos — copy video files only (MOV, MP4, MXF, etc.)
- [ ] Dotfile tracks copy mode per operation (`"mode": "selects"`)
- [ ] Status line reflects partial copy (`Selects copied on ...`)
- [ ] Re-copy guard per mode (don't skip if previous copy was a different mode)

### 0.1.9 — Code Health
- [ ] Split `main.go` into `app.go`, `display.go`, `copy_cmd.go`
- [ ] Add `context.Context` to `displayCard` and analyzer
- [ ] Remove startup `time.Sleep` calls
- [ ] Standardize error handling — `friendlyErr` everywhere user-facing
- [ ] Move `FormatBytes` to platform-agnostic file
- [ ] Test coverage pass — target 80%+ across all packages
- [ ] `go build -ldflags="-s -w"` for binary size reduction

---

### 0.2.0 — Daily Driver
**The "I am willing to use this in my workflow" release.**

Everything from 0.1.x is solid, tested, and feels intentional. This is the version
you hand to another photographer and say "try this."

- [ ] All 0.1.7, 0.1.8, 0.1.9 items complete
- [ ] Single-key input working reliably on macOS
- [ ] Selective copy (`[s]`, `[p]`, `[v]`) fully implemented with correct status tracking
- [ ] Partial copy state in dotfile — multi-mode copy history
- [ ] No known crashes or data loss scenarios
- [ ] Tested on personal gear across multiple shooting days
- [ ] Feedback from at least one other photographer on real-world use
- [ ] README reflects actual current behaviour (no aspirational features listed as present)
- [ ] First public-facing release candidate

---

---

### 0.3.0 — Linux Support

**Full Linux platform support — detection, copy, hardware info, speed test.**

- [ ] Linux detection (polling-based, /run/media, /media, /mnt)
- [ ] Linux hardware info (sysfs, CID parsing for direct SD slots)
- [ ] Linux speed test support
- [ ] Linux eject (udisksctl / umount)
- [ ] Real-world testing (Ubuntu, Fedora, Debian)
- [ ] Mount point documentation
- [ ] Stable Linux build and CI

---

## Later

- File renaming on copy (date-based, camera+date, sequence numbering)
- Resume interrupted copies
- Video metadata (duration, resolution)
- Windows support
- TOML/YAML config
- Toggle flat vs preserve DCIM structure
