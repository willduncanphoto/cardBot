# CardBot

A CLI tool for reading and copying camera memory cards into my workflow.

## What CardBot Does

CardBot generates a concise overview of camera memory cards and provides modern copy tools and logging to save time in professional work environments.

**Current capabilities:**
- Detect camera memory card volumes on macOS
- Quickly analyze content and technical information
- Disk space preflight check before copy to Destination folder
- Selective copy: Copy only Selects (Starred), Photos, Videos, or All
- Create dated folder structure based on card content during ingestion
- Copy files safely with size verification
- Rename files
- Remember card copy status
- Queue multiple cards
- Cancel in-progress transfers safely
- Eject cards safely (WHO DOES THAT?!)

## Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| macOS (with Xcode) | [OK] Working | Native DiskArbitration, instant detection |
| macOS (no Xcode) | [OK] Working | Polling fallback, 1s interval |
| Linux | [--] It might work | Planned |
| Windows | [--] Not supported | Not Planned |

## Installation

**No Go required.** The pre-built binaries run directly on macOS with zero dependencies.

### Option 1: Pre-built Binary (macOS)

Download the latest release for your Mac (Apple Silicon or Intel):

> Note: release binaries run fine without Xcode, but are currently built with `CGO_ENABLED=0`, so card detection uses polling fallback (~1s) instead of native instant DiskArbitration callbacks.
> Build from source with Xcode tools if you want native instant detection.

```bash
# Apple Silicon (M1/M2/M3)
curl -fL -o cardbot https://github.com/willduncanphoto/CardBot/releases/latest/download/cardbot-darwin-arm64
chmod +x cardbot
sudo mv cardbot /usr/local/bin/cardbot

# Intel Mac
curl -fL -o cardbot https://github.com/willduncanphoto/CardBot/releases/latest/download/cardbot-darwin-amd64
chmod +x cardbot
sudo mv cardbot /usr/local/bin/cardbot
```

**Avoid sudo:** Install to a user directory (e.g., `~/.local/bin`) and ensure it's in your PATH:
```bash
mkdir -p ~/.local/bin
curl -fL -o ~/.local/bin/cardbot https://github.com/willduncanphoto/CardBot/releases/latest/download/cardbot-darwin-arm64
chmod +x ~/.local/bin/cardbot
# Add to PATH if needed: export PATH="$HOME/.local/bin:$PATH"
```

Or grab the binary from the repo directly (Apple Silicon only):

```bash
git clone https://github.com/willduncanphoto/CardBot.git
cd CardBot
./cardbot --version
```

### Option 2: Build from Source

Requires Go 1.25 or later.

**macOS (Recommended — with Xcode):**
```bash
# Install Xcode CLI tools if you haven't already
xcode-select --install

git clone https://github.com/willduncanphoto/CardBot.git
cd CardBot
go build -o cardbot .
```

**macOS (without Xcode):**
```bash
CGO_ENABLED=0 go build -o cardbot .
```

**Linux (not yet supported):**
```bash
go build -o cardbot .
```

## Usage

**Quick start** — download and run (no install needed):

```bash
# Download for Apple Silicon Mac
curl -LO https://github.com/willduncanphoto/CardBot/releases/latest/download/cardbot-darwin-arm64
chmod +x cardbot-darwin-arm64
./cardbot-darwin-arm64
```

Or if you moved it to your PATH:

```bash
cardbot
```

Then insert a memory card.

**First run** — CardBot will open a folder picker (macOS) or prompt for a copy destination path. The choice is saved to `~/.config/cardbot/config.json`.

**Output example:**

```
[2026-03-17T10:30:44] CardBot 0.4.7
[2026-03-17T10:30:44] CardBot is up to date

[2026-03-17T10:30:51] Scanning started
[2026-03-17T10:30:51] "/Volumes/NIKON Z 9" (disk4s1) detected
[2026-03-17T10:30:51] Scanning 3051 files ✓
[2026-03-17T10:30:53] Scan completed in 2.3s

  Status:   New
  Path:     /Volumes/NIKON Z 9
  Storage:  96.4 GB / 476.9 GB (20%)
  Camera:   Nikon Z 9
  Starred:  1
  Content:  2026-02-27      12.9 GB   418   NEF
            2026-02-26      28.4 MB     1   NEF

  Total:    3048 photos, 0 videos, 96.0 GB

  Copy to:  ~/Pictures/CardBot
  Naming:   Timestamp + sequence

[a] Copy All  [e] Eject  [x] Exit  [?] Help  >
```

### Commands

| Key | Action |
|-----|--------|
| `a` | Copy All — copy all files to destination |
| `s` | Copy Selects — starred/picked files only |
| `p` | Copy Photos — photos only |
| `v` | Copy Videos — videos only |
| `e` | Eject the card |
| `x` | Exit — skip this card, move to next |
| `\` | Cancel Copy — cancel the copy in progress |
| `?` | Help — show all commands |

Press `?` to see all available commands.

### CLI Flags

| Flag | Description |
|------|-------------|
| `--dest <path>` | Override destination path for this session |
| `--dry-run` | Scan cards but do not copy files |
| `--setup` | Re-run destination setup |
| `--reset` | Clear saved config |
| `--version` | Print version and exit |

### Update Command

```bash
cardbot self-update
```

- Checks the latest GitHub release
- Verifies the downloaded binary with `checksums.txt` (SHA256)
- Replaces the current binary atomically
- Prints a `sudo` command if your install path is not writable

CardBot also checks for an update on startup.

## Copy

Press `a` to copy all files, or use selective copy modes to copy only specific file types. CardBot groups files into dated folders based on EXIF date:

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

**During copy:**
- Press `\` to cancel the copy in progress (files already copied are kept)
- If the card is removed mid-copy, CardBot detects it and stops gracefully
- Ctrl+C shuts down cleanly

**After copy:**
- CardBot writes a `.cardbot` file to the card
- On re-insert, the card shows `Status: Copy completed on 2026-03-12T12:31:05` instead of `Status: New`
- Re-copying the same card skips files that already exist with the correct size

**Invalid cards:**
- Cards without a DCIM folder show "Card is invalid (no DCIM found)" with basic info and eject/exit options

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
  "naming": {
    "mode": "original"
  },
  "output": {
    "color": true
  },
  "advanced": {
    "buffer_size_kb": 256,
    "exif_workers": 4,
    "log_file": "~/.cardbot/cardbot.log"
  }
}
```

Run `cardbot --setup` to change rerun the config setup. Run `cardbot --reset` to clear all saved config.

## Planned Stuff

- Resume interrupted copies
- ETA during copy
- Linux support
- Video and photo destinations

## Maybe Stuff

- Windows
- Custom filename workflow

## Size

- Binary: ~3.2 MB (stripped)
- Source: ~5,400 lines of Go across 55 files
- Tests: ~3,800 lines, 135 tests across 9 packages

## License

MIT License — see [LICENSE](LICENSE) for details.

## Notes

- **CID on Linux:** The SD Card Identification register (manufacturer ID, serial, manufacturing date) is only accessible with direct SD card slots. USB readers hide it.
- **Hardware size vs filesystem size:** macOS reports the card's raw physical capacity alongside the formatted filesystem size — this is why a "512GB" card shows ~477GB usable.
- **Speed test:** CardBot includes a hidden `[t]` command that runs a 256MB sequential read/write benchmark on the card. Results are synthetic — read speeds in particular may be inflated by the OS page cache.

## DISCLAIMER: Built with AI Coding Tools

CardBot was built with the help of AI coding models and many open source projects. There is no way that I could build this alone. A special thanks goes out to **[Pi](https://shittycodingagent.ai)** — a terminal-based coding agent.

- Website: [pi.dev](https://pi.dev)
- GitHub: [github.com/badlogic/pi-mono](https://github.com/badlogic/pi-mono)