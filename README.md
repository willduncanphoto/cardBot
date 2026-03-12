# CardBot

A CLI tool for camera memory card ingestion.

## What It Does

CardBot watches for memory card insertion, analyzes the contents, and helps you copy your work safely. It's designed for photographers and videographers who want a fast, no-nonsense ingest workflow from the terminal.

**Current capabilities:**
- Detect CFexpress, XQD, and SD cards on macOS and Linux
- Analyze card contents — files grouped by date with sizes and types
- Extract camera model and star ratings from EXIF/XMP
- Queue multiple cards
- Eject cards cleanly
- First-run setup with native macOS folder picker

## Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| macOS (with Xcode) | ✅ Working | Native DiskArbitration, instant detection |
| macOS (no Xcode) | ✅ Working | Polling fallback, 1s interval |
| Linux | 🧪 Experimental | Polling-based, needs real-world testing |
| Windows | ❌ Not supported | Future goal |

## Installation

Requires Go 1.21 or later.

### macOS (Recommended)

```bash
# Install Xcode CLI tools if you haven't already
xcode-select --install

git clone https://github.com/willduncanphoto/CardBot.git
cd CardBot
go build -o cardbot .
```

### macOS without Xcode

```bash
CGO_ENABLED=0 go build -o cardbot .
```

### Linux

```bash
go build -o cardbot .
```

## Usage

Run CardBot and insert a memory card:

```bash
./cardbot
```

**First run** — CardBot will open a folder picker (macOS) or prompt for a destination path. The choice is saved to `~/.config/cardbot/config.json`.

**Output example:**

```
[2026-03-11 19:30:14] Starting CardBot 0.1.4...
[2026-03-11 19:30:14] Scanning for memory cards...card found.
[2026-03-11 19:30:15] Scanning /Volumes/NIKON Z 9  ... 3048 files ✓
[2026-03-11 19:30:15] Scan completed in 1 second

  Path:     /Volumes/NIKON Z 9  
  Storage:  96.4 GB / 476.9 GB (20%)
  Brand:    Nikon
  Camera:   NIKON Z 9
  Starred:  1
  Content:  2026-02-27      12.9 GB    418   NEF
            2026-02-26      28.4 MB      1   NEF

  Total:    3048 photos, 0 videos, 96.0 GB
────────────────────────────────────────
[e] Eject  [c] Cancel  > 
```

### Commands

| Key | Action |
|-----|--------|
| `e` + Enter | Eject the card |
| `c` + Enter | Cancel / dismiss |

### CLI Flags

| Flag | Description |
|------|-------------|
| `--dest <path>` | Override destination path for this session |
| `--dry-run` | Scan cards but do not copy files |
| `--setup` | Re-run destination setup |
| `--reset` | Clear saved config |
| `--version` | Print version and exit |

## Supported Cameras

Tested:
- Nikon Z9 (CFexpress Type B)

Expected to work (based on DCIM folder patterns):
- Nikon Z8, Z7 II, Z6 III, D850, D780
- Canon EOS R5, R6, R3, 5D IV
- Sony A1, A7 IV, A7R V, FX3, FX6
- Fujifilm X-T5, X-H2S, GFX 100S
- Panasonic GH6, S5 II
- OM System OM-1

See [docs/CARDS.md](docs/CARDS.md) for the full testing checklist.

## Supported File Types

**Photos:** NEF, NRW, CR2, CR3, CRW, ARW, SRF, SR2, RAF, ORF, RW2, DNG, PEF, 3FR, IIQ, JPG, JPEG, TIF, TIFF, HEIC, HEIF, PNG

**Videos:** MOV, MP4, AVI, MXF, MTS, M2TS, R3D, BRAW

**Metadata:** EXIF (dates, camera model), XMP (star ratings)

## Configuration

Config is stored at `~/.config/cardbot/config.json`:

```json
{
  "$schema": "cardbot-config-v1",
  "destination": {
    "path": "~/Pictures/CardBot"
  },
  "output": {
    "color": true,
    "quiet": false
  },
  "advanced": {
    "buffer_size_kb": 256,
    "exif_workers": 4,
    "log_file": "~/.cardbot/cardbot.log"
  }
}
```

Run `cardbot --setup` to change the destination. Run `cardbot --reset` to clear all saved config.

## Roadmap

### ✅ 0.1.0 – 0.1.3 — Detection, Analysis, Config
- Native macOS card detection (DiskArbitration)
- DCIM walking, date grouping, file type breakdown
- EXIF camera model, XMP star ratings
- Hardware info (macOS via IOKit/system_profiler, Linux via sysfs/CID)
- Config file with first-run setup and native folder picker
- Logging with 5MB rotation
- Card queue for multiple cards

### 🔧 0.1.4 — UI Polish (In Progress)
- Merge brand + camera lines
- Clean up repetitive brand names
- Brand colors (ANSI)
- Handle edge cases: no DCIM, read-only cards, file collisions

### 📋 0.1.5 — Copy
- `[a] All` copy mode with dated folders (`CardBot_YYMMDD_001/`)
- Progress updates, cancel with cleanup
- Size verification
- `.cardbot` dotfile to track processed cards

### 📋 0.1.6 — Linux
- Verified testing on Ubuntu, Fedora, Debian

### 📋 0.1.7 — Polish
- Single-key input (no Enter required)
- Startup under 100ms

### 📋 0.1.8 — Distribution
- GitHub releases (macOS Intel/ARM, Linux AMD64/ARM64)
- Homebrew formula

**Later:** Windows support, starred-only copy mode, resume interrupted copies, video metadata

## Project Structure

```
cardbot/
├── main.go                          # CLI, event loop, display, input
├── internal/
│   ├── analyze/
│   │   ├── analyze.go               # DCIM walking, EXIF/XMP parsing, date grouping
│   │   └── analyze_test.go          # Unit tests
│   ├── config/
│   │   └── config.go                # Config load/save, schema versioning, path expansion
│   ├── detect/
│   │   ├── card.go                  # Card struct
│   │   ├── shared.go                # Brand detection, FormatBytes
│   │   ├── detect_darwin.go         # macOS native (CGO + DiskArbitration)
│   │   ├── detect_darwin_nocgo.go   # macOS polling fallback
│   │   ├── detect_linux.go          # Linux polling
│   │   ├── detect_other.go          # Unsupported platforms stub
│   │   ├── hardware_darwin.go       # macOS hardware info (IOKit, system_profiler)
│   │   └── hardware_linux.go        # Linux hardware info (sysfs, CID)
│   ├── log/
│   │   └── log.go                   # File logging with rotation
│   ├── pick/
│   │   ├── pick_darwin.go           # Native macOS folder picker (osascript)
│   │   └── pick_other.go            # Fallback stub
│   └── speedtest/
│       ├── speedtest_darwin.go      # 256MB sequential read/write benchmark
│       └── speedtest_other.go       # Stub for unsupported platforms
├── docs/                            # Project documentation
└── go.mod
```

## Dependencies

- **Runtime:** Zero external runtime dependencies
- **Build:** `github.com/evanoberholster/imagemeta` for EXIF/XMP parsing
- **Optional:** Xcode CLI Tools for macOS native card detection

## Size

- Binary: ~5.1 MB
- Source: ~3,100 lines of Go across 17 files

## License

TBD — will be added before public release (0.1.8)

## Built With AI

CardBot was built with the help of human-guided AI coding models. Special thanks to **[Pi](https://shittycodingagent.ai)** — a terminal-based coding agent.

- Website: [shittycodingagent.ai](https://shittycodingagent.ai)
- GitHub: [github.com/badlogic/pi-mono](https://github.com/badlogic/pi-mono)

## Notes

- **CID on Linux:** The SD Card Identification register (manufacturer ID, serial, manufacturing date) is only accessible with direct SD card slots. USB readers hide it.
- **Hardware size vs filesystem size:** macOS reports the card's raw physical capacity alongside the formatted filesystem size — this is why a "512GB" card shows ~477GB usable.
- **Speed test:** CardBot includes a hidden `[t]` command that runs a 256MB sequential read/write benchmark on the card. Results are synthetic — read speeds in particular may be inflated by the OS page cache.
