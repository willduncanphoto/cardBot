# CardBot

A CLI tool for camera memory cards.

![alt text](screenshot.png)

## DISCLAIMER: Built with AI Coding Tools

CardBot was built with the help of AI coding tools and many open source projects. This is an experiment in LLM slop. There is no warranty.

A special thanks goes out to **[Pi](https://shittycodingagent.ai)** — a terminal-based coding agent.

- Website: [pi.dev](https://pi.dev)
- GitHub: [github.com/badlogic/pi-mono](https://github.com/badlogic/pi-mono)

## What CardBot does

CardBot generates a concise overview camera memory cards and provides modern copy to offload media to a local destination.

**Current capabilities:**
- Detect camera memory cards on macOS
- Quickly analyze card content
- Disk space preflight check before copy
- Selective copy: copy only selects (Starred), photos, videos, or all
- Copy to dated folders
- Rename files to a ISO 8601 format
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

**First run** — CardBot will open a folder picker (macOS), ask for naming mode, ask whether to auto-launch on card insert, ask whether daemon mode should start automatically at login, and ask which terminal app daemon mode should open. Choices are saved to `~/.config/cardbot/config.json`.

To change these later, run `cardbot --setup` again (it reworks all setup choices each time).

**Output example:**

```
[2026-03-17T10:30:44] CardBot 0.5.0
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
| `--daemon` | Run headless daemon mode (watch for cards in background) |
| `--setup` | Re-run setup prompts (destination, naming, daemon, start-at-login, terminal app) |
| `--reset` | Clear saved config |
| `--version` | Print version and exit |

### Background Daemon

Run CardBot in headless background mode:

```bash
cardbot --daemon
```

Install/uninstall login auto-start on macOS:

```bash
cardbot install-daemon
cardbot uninstall-daemon
```

Check daemon + LaunchAgent status:

```bash
cardbot daemon-status
cardbot daemon-status --json
```

JSON mode includes `version`, `pid`, `daemon`, `single_instance_guard`, and `launch_agent` fields.

This mode watches for card insertions without showing the interactive prompt.
It launches your preferred terminal app from config (`daemon.terminal_app`).
Advanced command templates can be set with `daemon.launch_args`
using `{{cardbot_binary}}` and `{{mount_path}}` placeholders.

Daemon reliability notes:
- Single-instance guard: if another `cardbot` process is already running, auto-launch is skipped.
- Duplicate cooldown: rapid duplicate mount events (common around sleep/wake) are suppressed briefly.

Daemon troubleshooting:
- If launch fails with Apple Events/automation errors, grant Automation permission in macOS Privacy & Security.
- If launch fails with permission denied/operation not permitted, grant Full Disk Access.

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
- Re-copying the same card skips files that already exist with the correct size (untested)

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
  "daemon": {
    "enabled": false,
    "start_at_login": false,
    "terminal_app": "Terminal",
    "launch_args": []
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

Run `cardbot --setup` to re-run setup and change saved preferences. Run `cardbot --reset` to clear all saved config.

## Release Prep (0.5.0)

- QA checklist: `docs/050_QA_CHECKLIST.md`
- Release notes draft: `docs/050_RELEASE_NOTES_DRAFT.md`

## Roadmap

| Version | Focus | Status |
|---------|-------|--------|
| **0.5.0** | Background auto-launch (daemon + LaunchAgent + status) | Current |
| **0.5.1** | Bugfix follow-up from 0.5.0 testing | Next |
| **0.6.0** | Copy operations improvements | Planned |
| **0.8.0** | Copyright metadata injection | Planned |

### Maybe Someday
- Windows support
- Checksum verification (xxhash)
- Network destinations (SFTP, S3)
- Camera datetime drift detection
- Audio note file transcription

## Size

- Binary: ~10 MB (stripped)
- Source: ~9,200 lines of Go across 55 files
- Tests: 137 tests across 10 packages

## License

MIT License — see [LICENSE](LICENSE) for details.

## Notes

- **CID on Linux:** The SD Card Identification register (manufacturer ID, serial, manufacturing date) is only accessible with direct SD card slots. USB readers hide it.
- **Hardware size vs filesystem size:** macOS reports the card's raw physical capacity alongside the formatted filesystem size — this is why a "512GB" card shows ~477GB usable.
- **Speed test:** CardBot includes a hidden `[t]` command that runs a 256MB sequential read/write benchmark on the card. Results are synthetic — read speeds in particular may be inflated by the OS page cache.