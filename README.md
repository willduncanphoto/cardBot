# CardBot

A CLI tool for camera memory cards.

## What CardBot Does

CardBot generates a concise overview of your memory card and provides modern copy tools. It will also rename your files one day.

**Current capabilities:**
- Detect CFexpress, XQD, and SD cards on macOS and Linux
- Quickly analyze a cards content and technical information
- Show starred image count for future quick copy opteration
- Copy all files to dated folders with basic verification
- Track copy history via `.cardbot` dotfile written to the card
- Queue multiple cards
- Eject cards safely
- Doesn't delete your hard work

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
[2026-03-11 21:15:32] Starting CardBot 0.1.5...
[2026-03-11 21:15:32] Copy location is set to ~/Pictures/CardBot
[2026-03-11 21:15:32] File renaming is set to Original
[2026-03-11 21:15:32] Scanning for memory cards...card found.
[2026-03-11 21:15:33] Scanning /Volumes/NIKON Z 9  ... 3051 files ✓
[2026-03-11 21:15:33] Scan completed in 0 seconds

  Status:   New
  Path:     /Volumes/NIKON Z 9  
  Storage:  96.4 GB / 476.9 GB (20%)
  Camera:   Nikon Z 9
  Starred:  1
  Content:  2026-02-27      12.9 GB    418   NEF
            2026-02-26      28.4 MB      1   NEF

  Total:    3048 photos, 0 videos, 96.0 GB
────────────────────────────────────────
[a] Copy All  [e] Eject  [c] Cancel  >
```

### Commands

| Key | Action |
|-----|--------|
| `a` + Enter | Copy all files to destination |
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

## Copy

Press `a` to copy all files. CardBot groups files into dated folders based on EXIF date:

```
~/Pictures/CardBot/
├── 2026-02-26/
│   └── 100NIKON/
│       └── DSC_0001.NEF
├── 2026-02-27/
│   └── 100NIKON/
│       ├── DSC_0002.NEF
│       └── DSC_0003.JPG
└── 2026-03-08/
    └── 101NIKON/
        └── DSC_0200.MOV
```

After a successful copy, CardBot writes a `.cardbot` file to the card. On re-insert, the card shows `Status: Copied on 2026-03-11 21:31` instead of `Status: New`.

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

### ✅ 0.1.0 – 0.1.5 — Detection, Analysis, Config, Copy & Hardening
- Native macOS card detection (DiskArbitration)
- DCIM walking, date grouping, file type breakdown
- EXIF camera model, XMP star ratings, parallel EXIF workers
- Hardware info (macOS via IOKit/system_profiler, Linux via sysfs/CID)
- Config file with first-run setup and native folder picker
- Brand name cleanup and ANSI colors
- Copy all files with dated folders, size verification, dotfile tracking
- File collision skip (same size = skip, safe re-copy)
- Bug fixes: race conditions, input drain, path escaping, log formatting
- Test suite: 81 tests across 6 packages

### 🔧 0.1.6 — Copy Stuff (Next)
- Handle card removal during copy, disk full, cancel with cleanup
- Read-only card warnings
- Output mutex for concurrent progress/scan output
- Better error messages

### 📋 0.1.7 — Cleanup
- Single-key input (no Enter required)
- Startup under 100ms, ETA during copy

**Later:** Windows support, Linux testing, file renaming, starred-only copy mode, resume interrupted copies, video metadata, auto-update, copyright/personal data injection on copy

## Project Structure

```
cardbot/
├── main.go                          # CLI, event loop, display, input, copy orchestration
├── internal/
│   ├── analyze/
│   │   ├── analyze.go               # DCIM walking, parallel EXIF/XMP, date grouping
│   │   └── analyze_test.go          # Unit tests
│   ├── config/
│   │   └── config.go                # Config load/save, schema versioning, path expansion
│   ├── copy/
│   │   └── copy.go                  # File copy engine — walk, copy, verify
│   ├── detect/
│   │   ├── card.go                  # Card struct
│   │   ├── shared.go                # Brand detection, FormatBytes
│   │   ├── detect_darwin.go         # macOS native (CGO + DiskArbitration)
│   │   ├── detect_darwin_nocgo.go   # macOS polling fallback
│   │   ├── detect_linux.go          # Linux polling
│   │   ├── detect_other.go          # Unsupported platforms stub
│   │   ├── hardware_darwin.go       # macOS hardware info (IOKit, system_profiler)
│   │   └── hardware_linux.go        # Linux hardware info (sysfs, CID)
│   ├── dotfile/
│   │   └── dotfile.go               # .cardbot read/write for copy tracking
│   ├── log/
│   │   └── log.go                   # File logging with rotation
│   ├── pick/
│   │   ├── pick_darwin.go           # Native macOS folder picker (osascript)
│   │   └── pick_other.go            # Fallback stub
│   ├── speedtest/
│   │   ├── speedtest_darwin.go      # 256MB sequential read/write benchmark
│   │   └── speedtest_other.go       # Stub for unsupported platforms
│   └── ui/
│       └── color.go                 # ANSI brand colors
├── docs/                            # Project documentation
└── go.mod
```

## Dependencies

- **Runtime:** Zero external runtime dependencies
- **Build:** `github.com/evanoberholster/imagemeta` for EXIF/XMP parsing
- **Optional:** Xcode CLI Tools for macOS native card detection

## Size

- Binary: ~4.9 MB
- Source: ~3,500 lines of Go across 20 files
- Tests: ~1,350 lines, 81 tests across 6 packages

## License

TBD — will be added before public release

## DISCLAIMER: Built with AI Coding Tools

CardBot was built with the help of human guided AI coding models and many many many open source projects. There is no way in hell I could do this alone. A special thanks goes out to **[Pi](https://shittycodingagent.ai)** — a terminal-based coding agent.

- Website: [shittycodingagent.ai](https://shittycodingagent.ai)
- GitHub: [github.com/badlogic/pi-mono](https://github.com/badlogic/pi-mono)

## Notes

- **CID on Linux:** The SD Card Identification register (manufacturer ID, serial, manufacturing date) is only accessible with direct SD card slots. USB readers hide it.
- **Hardware size vs filesystem size:** macOS reports the card's raw physical capacity alongside the formatted filesystem size — this is why a "512GB" card shows ~477GB usable.
- **Speed test:** CardBot includes a hidden `[t]` command that runs a 256MB sequential read/write benchmark on the card. Results are synthetic — read speeds in particular may be inflated by the OS page cache.
