# Output Format

## Full Startup Flow

Complete sequence from launch to first card display:

```
[2026-03-12T12:15:32] Starting CardBot 0.1.9.     ← dots animate over ~1s
[2026-03-12T12:15:33] Starting CardBot 0.1.9..
[2026-03-12T12:15:34] Starting CardBot 0.1.9...
[2026-03-12T12:15:34] Copy path /Pictures/CardBot
[2026-03-12T12:15:34] Keep original filenames
[2026-03-12T12:15:34] Scanning  |                  ← spinner animates: | / - \
[2026-03-12T12:15:35] Card detected               ← card inserted, spinner stops
[2026-03-12T12:15:35] Scanning /Volumes/NIKON Z 9  ... 3051 files ✓ (0s)

  Status:   New
  Path:     /Volumes/NIKON Z 9
  Storage:  96.4 GB / 476.9 GB (20%)
  Camera:   Nikon Z 9
  Starred:  1
  Content:  2026-02-27      12.9 GB    418   NEF
            2026-02-26      28.4 MB      1   NEF

  Total:    3048 photos, 0 videos, 96.0 GB
────────────────────────────────────────
[a] Copy All  [e] Eject  [x] Exit  [?] Help  >
```

## Startup

```
[2026-03-12T12:15:32] Starting CardBot 0.1.9...
[2026-03-12T12:15:32] Copy path /Pictures/CardBot
[2026-03-12T12:15:32] Keep original filenames
[2026-03-12T12:15:32] Scanning  |
```

## First Run (No Config)

```
Welcome to CardBot!

Where should CardBot copy your work?

[macOS: native folder picker opens]

Destination: /Users/user/Pictures/CardBot

[2026-03-12T12:15:32] Starting CardBot 0.1.9...
[2026-03-12T12:15:32] Copy path /Pictures/CardBot
[2026-03-12T12:15:32] Scanning...
```

## Card Detected (New)

```
[2026-03-12T12:15:32] Card detected
[2026-03-12T12:15:33] Scanning /Volumes/NIKON Z 9  ... 3051 files ✓ (0s)

  Status:   New
  Path:     /Volumes/NIKON Z 9
  Storage:  96.4 GB / 476.9 GB (20%)
  Camera:   Nikon Z 9
  Starred:  1
  Content:  2026-02-27      12.9 GB    418   NEF
            2026-02-26      28.4 MB      1   NEF
            ...

  Total:    3048 photos, 0 videos, 96.0 GB
────────────────────────────────────────
[a] Copy All  [e] Eject  [x] Exit  [?]  >
```

## Card Detected (Previously Copied)

```
  Status:   Copy completed on 2026-03-12T12:31:05
  Path:     /Volumes/NIKON Z 9
  Storage:  96.4 GB / 476.9 GB (20%)
  Camera:   Nikon Z 9
  ...
────────────────────────────────────────
[a] Copy All  [e] Eject  [x] Exit  [?]  >
```

## Card Invalid (No DCIM)

```
[2026-03-12T12:15:33] Card is invalid (no DCIM found)

  Status:   New
  Path:     /Volumes/UNTITLED
  Storage:  1.2 GB / 32.0 GB (3%)
  Camera:   Unknown
  Content:  (no DCIM — not a camera card)
────────────────────────────────────────
[e] Eject  [x] Exit  [?]  >
```

## Copy Progress

```
[a] Copy All  [e] Eject  [x] Exit  [?]  > a

[2026-03-12T12:15:35] Copying all files to ~/Pictures/CardBot
[2026-03-12T12:15:35] Press [\] to cancel
[2026-03-12T12:15:40] Copying... 1247/3051 files  48.2 GB/96.4 GB (50%)
...
[2026-03-12T12:22:18] Copy complete ✓
[2026-03-12T12:22:18] 3051 files, 96.0 GB copied in 8m32s (188.4 MB/s)

[e] Eject  [x] Done  [?]  >
```

## Copy Cancelled

```
[2026-03-12T12:18:05] Copy cancelled — 1247 files copied.
[a] Copy All  [e] Eject  [x] Exit  [?]  >
```

## Destination Structure

Files are grouped by date, preserving original folder structure:

```
~/Pictures/CardBot/
├── 2026-02-26/
│   └── 100NIKON/
│       └── DSC_0001.NEF
├── 2026-02-27/
│   └── 100NIKON/
│       ├── DSC_0002.NEF
│       ├── DSC_0003.NEF
│       └── DSC_0004.JPG
└── 2026-03-08/
    ├── 100NIKON/
    │   └── DSC_0100.NEF
    └── 101NIKON/
        └── DSC_0200.MOV
```

## Eject

```
[a] Copy All  [e] Eject  [x] Exit  [?]  > e
Ejecting NIKON Z 9  ...

[2026-03-12T12:20:15] Card ejected: /Volumes/NIKON Z 9

[2026-03-12T12:20:18] Scanning...
```

## Card Removal (Unexpected)

```
[2026-03-12T12:20:15] Card removed: /Volumes/NIKON Z 9

[2026-03-12T12:20:18] Scanning...
```

## Queue

When multiple cards are inserted:

```
[2026-03-12T12:15:33] Nikon detected (1 card in queue)
```

Queue is processed in insertion order. The queue count appears when additional cards are waiting.

## Commands

| Key | Action |
|-----|--------|
| `a` | Copy All — copy all files to destination |
| `s` | Copy Selects — copy starred/picked files only |
| `p` | Copy Photos — copy photos only |
| `v` | Copy Videos — copy videos only |
| `e` | Eject the card |
| `x` | Exit — skip this card, move to next |
| `\` | Cancel Copy — cancel the copy in progress |
| `?` | Help — show all commands |

### Hidden Commands

| Key | Action |
|-----|--------|
| `i` | Show card hardware info (device, model, serial, firmware) |
| `t` | Run 256MB speed test (sequential write + read) |

## Help Screen

Output of the `[?]` command:

```
  Commands:
  [a]  Copy All       copy all files to destination
  [s]  Copy Selects   copy starred/picked files only
  [p]  Copy Photos    copy photos only              
  [v]  Copy Videos    copy videos only              
  [e]  Eject          safely eject this card
  [x]  Exit           skip this card, move to next
  [i]  Card Info      show hardware details
  [t]  Speed Test     benchmark read/write speed
  [\]  Cancel Copy    cancel the copy in progress
  [?]  Help           show this help
```

## Content Layout

Fixed-width columns for consistent visual scanning:

```
  Content:  2026-03-08      12.9 GB    418   NEF
            2026-03-07      28.4 MB      1   NEF, JPG
```

| Column | Width | Alignment | Description |
|--------|-------|-----------|-------------|
| Date | 10 chars | Left | `YYYY-MM-DD` |
| Size | 10 chars | Right | `NNN.N GB` or `NNN.N MB` |
| Count | variable | Right | File count, right-aligned to widest |
| Extensions | variable | Left | Sorted alphabetically, uppercase |

## Dry-Run Mode

```bash
./cardbot --dry-run
```

Shows `(dry-run)` next to destination. Copy commands report what would happen without writing files.
