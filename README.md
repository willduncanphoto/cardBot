# CardBot

A CLI tool for camera memory card ingestion.

## What It Does

CardBot gives you a quick overview and copy tools for ingesting your photography and video files in a simple and safe manner.

**Current capabilities:**
- Detect CFexpress, XQD and SD cards on macOS and Linux
- Display card overview
- Queue multiple cards

- Eject cards cleanly

## Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| macOS (with Xcode) | ✅ Working | Native DiskArbitration, instant detection |
| macOS (no Xcode) | ✅ Working | Polling fallback, 1s interval |
| Linux | 🧪 Experimental | Polling-based, needs testing |
| Windows | ❌ Not supported | Future goal |

## Installation

### From Source

Requires Go 1.21 or later.

```bash
git clone <repo>
cd cardbot
go build -o cardbot .
```

### macOS with Xcode (Recommended)

```bash
# One-time setup
xcode-select --install

# Build
go build -o cardbot .
```

### macOS without Xcode

```bash
CGO_ENABLED=0 go build -o cardbot .
```

## Usage

Run CardBot and insert a memory card:

```bash
./cardbot
```

Output:
```
[2026-03-11 13:57:36] Starting CardBot 0.1.3...
[2026-03-11 13:57:36] Scanning for memory cards...card found.
[2026-03-11 13:57:36] Scanning /Volumes/NIKON Z 9  ... 3048 files ✓
[2026-03-11 13:57:36] Scan completed in 1 second

  Path:     /Volumes/NIKON Z 9  
  Storage:  96.4 GB / 476.9 GB (20%)
  Brand:    Nikon
  Camera:   NIKON Z 9
  Starred:  1
  Content:  2026-02-27      12.9 GB    418   NEF
            2026-02-26      28.4 MB      1   NEF
            ...

  Total:    3048 photos, 0 videos, 96.0 GB
────────────────────────────────────────
[e] Eject  [c] Cancel  > 
```

### Commands

| Key | Action |
|-----|--------|
| `e` + Enter | Eject the card |
| `c` + Enter | Cancel and dismiss |

## Supported Cameras

Tested and verified:
- Nikon Z9 (SD and CFexpress Type B)

Expected to work (based on DCIM folder patterns):
- Nikon Z8, Z7, Z6, D850, D780
- Canon EOS R5, R6, R7, 5D IV
- Sony A1, A7 IV, A7R V, FX3
- Fujifilm X-T5, X-H2S, GFX 100S
- Panasonic GH6, S5 II
- OM System OM-1

See [docs/CARDS.md](docs/CARDS.md) for full testing checklist.

## Supported File Types

**Photos:** NEF, NRW, CR2, CR3, CRW, ARW, SRF, SR2, RAF, ORF, RW2, DNG, PEF, 3FR, IIQ, JPG, JPEG, TIF, TIFF, HEIC, HEIF, PNG

**Videos:** MOV, MP4, AVI, MXF, MTS, M2TS, R3D, BRAW

**Metadata:** EXIF (dates, camera model), XMP (star ratings)

## Roadmap

### 0.1.3 — Config & Destination (Current)
- [ ] Config file: `~/.config/cardbot/config.json`
- [ ] First-run destination prompt
- [ ] CLI flags: `--dest`, `--version`, `--dry-run`
- [ ] Logging

### 0.1.4 — UI Polish
- [ ] Merge brand + camera lines
- [ ] Clean up repetitive brand names ("Nikon NIKON Z 9" → "Nikon Z 9")
- [ ] Brand colors (Nikon yellow, Canon red, etc.)
- [ ] Handle "no DCIM" warning
- [ ] Handle read-only cards
- [ ] File collision logic (skip, rename, overwrite)
- [ ] 50k+ file performance

### 0.1.5 — Copy
- [ ] `[a] All` copy mode
- [ ] Dated folders: `CardBot_YYMMDD_001/`
- [ ] Progress updates every 5 seconds
- [ ] Cancel with cleanup
- [ ] Size verification
- [ ] `.cardbot` dotfile for tracking

### 0.1.6 — Linux
- [ ] Test on Ubuntu, Fedora, Debian
- [ ] Document mount point behavior
- [ ] Stable Linux build

### 0.1.7 — Polish
- [ ] Startup <100ms
- [ ] Single-key input (no Enter required)
- [ ] Performance benchmarks

### 0.1.8 — Distribution
- [ ] GitHub releases
- [ ] Homebrew formula
- [ ] Complete documentation

### Later
- Windows support
- `[s] Selects` mode (copy only starred images)
- Incremental/delta copy
- Resume interrupted copies
- Video metadata extraction

## Project Structure

```
cardbot/
├── main.go                     # CLI and event handling
├── internal/
│   ├── analyze/
│   │   ├── analyze.go          # DCIM walking, EXIF/XMP parsing
│   │   └── analyze_test.go     # Tests
│   └── detect/
│       ├── card.go             # Card struct
│       ├── shared.go           # Cross-platform helpers
│       ├── detect_darwin.go    # macOS native (CGO)
│       ├── detect_darwin_nocgo.go  # macOS fallback
│       ├── detect_linux.go     # Linux polling
│       ├── detect_other.go     # Unsupported platforms
│       ├── hardware_darwin.go  # macOS hardware info
│       └── hardware_linux.go   # Linux hardware info + CID
├── go.mod                      # Dependencies (just imagemeta)
├── .gitignore                  # What's excluded
└── README.md                   # This file
```

## Dependencies

- **Runtime:** Zero external dependencies (Go stdlib only)
- **Build:** `github.com/evanoberholster/imagemeta` v0.3.1 for EXIF/XMP
- **Optional:** Xcode CLI Tools for macOS native detection

## Size

- Binary: ~4.2 MB
- Source: ~2,600 lines of Go

## License

TBD — will be added before distribution (0.1.8)

## Notes

- **CID Access:** The SD Card Identification register (manufacturer ID, serial number, manufacturing date) is only available on Linux with direct SD card slots. USB readers hide this information.
- **Hardware Info:** macOS shows raw device size (what the card physically is) vs filesystem size (what you can actually use). This explains why a "512GB" card shows ~477GB available.
