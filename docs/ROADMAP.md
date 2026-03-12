# Roadmap

Work items grouped by milestone.

---

## Completed

### 0.1.0 — Detection
- [x] macOS native detection (CGO + DiskArbitration)
- [x] macOS fallback detection (polling, no Xcode)
- [x] Linux detection (polling-based)
- [x] DCIM folder detection
- [x] Brand guess from folder names
- [x] Basic display (path, storage, brand)
- [x] `[e] Eject` and `[c] Cancel` actions
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
- [x] Hardware info query via IOKit (macOS) / sysfs (Linux)
- [x] Raw device size vs filesystem size
- [x] CID parsing on Linux with direct SD slot
- [x] `[i]` key for hardware info (hidden command)

---

## Upcoming

### 0.1.3 — Config & Destination (Current)
- [x] Config file: `~/.config/cardbot/config.json`
- [x] First-run setup — prompt for destination
- [x] Store: default destination path
- [x] CLI flags: `--dest`, `--version`
- [x] `--dry-run` mode
- [x] Logging: `~/.cardbot/cardbot.log` (5MB rotation)

### 0.1.4 — UI Polish
- [ ] Merge brand + camera lines
- [ ] Clean up repetitive brand names ("Nikon NIKON Z 9" → "Nikon Z 9")
- [ ] Brand colors (ANSI)
- [ ] Handle "no DCIM" case
- [ ] Handle read-only cards
- [ ] File collision logic (skip, rename, overwrite)
- [ ] Better error messages
- [ ] Performance: handle 50k+ files

### 0.1.5 — Copy
- [ ] `[a] All` copy mode
- [ ] Dated folders: `CardBot_YYMMDD_001/`
- [ ] Progress updates every 5 seconds
- [ ] Cancel copy with cleanup
- [ ] Handle card removed during copy
- [ ] Handle destination disk full
- [ ] Handle corrupt files
- [ ] Size verification
- [ ] `.cardbot` dotfile
- [ ] "Processed" status on re-insert
- [ ] `[⌥T]` Toggle: flat vs preserve DCIM
- [ ] `[v]` Videos only, `[p]` Photos only
- [ ] Estimated time remaining

### 0.1.6 — Linux
- [ ] Test on Ubuntu, Fedora, Debian
- [ ] Document mount point behavior
- [ ] Linux build marked stable

### 0.1.7 — Polish
- [ ] Startup <100ms
- [ ] Single-key input (raw terminal mode)
- [ ] Config validation
- [ ] Performance benchmarks

### 0.1.8 — Distribution
- [ ] README with install/usage
- [ ] GitHub releases (macOS Intel/ARM, Linux AMD64/ARM64)
- [ ] Homebrew formula
- [ ] `--help` with examples
- [ ] LICENSE file

---

## Later

- Windows support
- `[s] Selects` copy mode (starred only)
- Incremental copy
- Resume interrupted copies
- Video metadata (duration, resolution)
- Network destinations
- TOML/YAML config
- JSON output mode
