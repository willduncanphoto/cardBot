# Output Format

## Card Detected — 0.1.0 (Minimal)

```
[2025-03-08 21:15:32] Card detected
  Path:     /Volumes/NIKON Z9
  Storage:  494.2 GB / 512.0 GB (97%)
  Brand:    Nikon
  Camera:   NIKON Z 9
────────────────────────────────────────
[e] Eject  [c] Cancel  >
```

Camera is extracted from EXIF; brand is guessed from DCIM folder names (e.g., `100NIKON` → Nikon).

## Card Detected — 0.2.0+ (Full Analysis)

```
[2025-03-08 21:15:32] Card detected
  Path:     /Volumes/NIKON Z9
  Storage:  494.2 GB / 512.0 GB (97%)
  Brand:    Nikon
  Camera:   NIKON Z 9
  Starred:  12
  Content:
    2025-03-08   142.3 GB   NEF, JPG, MOV
    2025-03-07    89.5 GB   NEF, JPG
  Status:   New
────────────────────────────────────────
[a] All  [s] Selects  [v] Videos  [p] Photos  [c] Cancel  [e] Eject  >
[⌥I] Card Info  [⌥T] Toggle Structure  [⌥D] Destination
(1 card in queue)
 > _
```

Note: Press key + Enter. `[s] Selects` is unavailable until starred image detection is implemented. Queue line only appears when additional cards are waiting.

## Card Detected (Processed)

```
[2025-03-09 10:22:15] Card detected
  Path:     /Volumes/NIKON Z9
  Storage:  398.1 GB / 512.0 GB (78%)
  Brand:    Nikon
  Camera:   NIKON Z 9
  Content:
    2025-03-09    48.2 GB   NEF, MOV
  Status:   Processed (copied 2 days ago)
────────────────────────────────────────
[Space] Copy again  [e] Eject  >
```

Pressing `Space` shows the main options prompt: `[a] All [s] Selects [v] Videos [p] Photos [c] Cancel [e] Eject`

## Copy Progress

```
[a] All  [s] Selects  [v] Videos  [p] Photos  [c] Cancel  [e] Eject  >  a
Destination: ~/Pictures/Ingest/CardBot_250308_001
Structure:  Flat
Proceed? [y/⌥T/n] > y

[2025-03-08 21:15:35] Copy started: NIKON Z9 -> ~/Pictures/Ingest/CardBot_250308_001
[2025-03-08 21:15:40] Copied 5.2 GB / 255.6 GB (2%) at 78.5 MB/s
...
[2025-03-08 21:22:18] Copy complete: 255.6 GB in 6m 42s (85.2 MB/s), 0 errors
[2025-03-08 21:22:18] Marked as processed: NIKON Z9
```

Options at proceed prompt:
- `y` = Yes, copy with current settings
- `⌥T` = Toggle structure (flat ↔ preserve DCIM)
- `n` = No, cancel

After toggle:

```
Destination: ~/Pictures/Ingest/CardBot_250308_001
Structure:  Preserve (keep DCIM folders)
Proceed? [y/t/n] > y

[2025-03-08 21:15:35] Copy started: NIKON Z9 -> ~/Pictures/Ingest/CardBot_250308_001
[2025-03-08 21:15:40] Copied 5.2 GB / 255.6 GB (2%) at 78.5 MB/s
...
[2025-03-08 21:22:18] Copy complete: 255.6 GB in 6m 42s (85.2 MB/s), 0 errors
[2025-03-08 21:22:18] Marked as processed: NIKON Z9
```

## Eject

When user presses `[e]`:

```
[a] All  [s] Selects  [v] Videos  [p] Photos  [c] Cancel  [e] Eject  >  e
[2025-03-08 21:20:15] NIKON Z9 has been ejected.
[2025-03-08 21:20:18] Monitoring for cards...
>
```

Eject message displays for 2-3 seconds, then returns to monitoring prompt.

## Card Removal (Unexpected)

```
[2025-03-08 21:20:15] Card removed: /Volumes/NIKON Z9
>
```

## Content Layout Specification

Fixed-width columns for consistent visual scanning:

```
    YYYY-MM-DD    NNN.N GB   EXT, EXT, EXT
    ───────────────────────────────────────
    2025-03-08    142.3 GB   NEF, JPG, MOV
    2025-03-07     89.5 GB   NEF, JPG
    2025-02-15      2.1 GB   MP4
```

| Column | Width | Alignment | Format |
|--------|-------|-----------|--------|
| Indent | 4 spaces | - | Leading padding |
| Date | 10 chars | Left | `YYYY-MM-DD` |
| Gap | 3 spaces | - | Separator |
| Size | 10 chars | Right | `NNN.N GB` or `NNN.N MB` |
| Gap | 3 spaces | - | Separator |
| Extensions | variable | Left | Sorted alphabetically, uppercase |

Size formatting rules:
- GB if ≥ 1 GB: `142.3 GB`, `89.5 GB`, `2.1 GB`
- MB if < 1 GB: `512.0 MB`, `89.5 MB`
- Always one decimal place
- Unit always 2 chars + space before

Extension formatting:
- Uppercase always
- Comma + space separator: `NEF, JPG, MOV`
- Alphabetical order

## Special Cases

| Case | Display |
|------|---------|
| Unknown Card | `Brand: -- \| Camera: -- \| Content: --` |
| Multiple Cameras | `Brand: Nikon \| Camera: NIKON Z 9, D850` (different models) or `Brand: Nikon \| Camera: NIKON Z 9 (2)` (same model, different serials) |
| No starred images | Omit `Starred:` line entirely |

## Hidden Commands

### Card Hardware Info

Press `⌥I` (Option+I) at any prompt to display card hardware metadata:

```
[a] All  [s] Selects  [v] Videos  [p] Photos  [c] Cancel  [e] Eject  >  ⌥I
  Hardware: SanDisk Extreme Pro
  Serial:   0xA1B2C3D4
  Type:     SDXC UHS-II
────────────────────────────────────────
[a] All  [s] Selects  [v] Videos  [p] Photos  [c] Cancel  [e] Eject  >
```

If hardware info is unavailable (USB reader):

```
> ⌥I
  Hardware: Not available (USB reader)
────────────────────────────────────────
```

### Edit Destination

Press `⌥D` (Option+D) at any prompt to change the default destination:

```
[a] All  [s] Selects  [v] Videos  [p] Photos  [c] Cancel  [e] Eject  >  ⌥D
Current destination: ~/Pictures/Ingest
New destination: /Volumes/RAID
Save as default? [Y/n] > y
You're all set!
────────────────────────────────────────
[a] All  [s] Selects  [v] Videos  [p] Photos  [c] Cancel  [e] Eject  >
```

Changes are saved to `~/.config/cardbot/config.json`.

## Application Startup

When CardBot starts, a brief intro sequence with spinner displays:

```
$ cardbot
Starting CardBot 0.1.0...
⠋
```

After 2-3 seconds, transitions to:

```
CardBot 0.1.0
Destination: ~/Pictures/Ingest
Scanning for memory cards...

>
```

The `>` prompt indicates CardBot is ready and listening. No user input is accepted at this prompt-it's for display only. Commands are triggered by card insertion.

## First-Run Setup

On first launch (no config file exists), CardBot prompts for destination:

```
$ cardbot
Starting CardBot 0.1.0...
⠋

When a card is detected, where would you like the copy destination to be?
Destination [~/Pictures/Ingest] > /Volumes/Backup
Save as default? [Y/n] > y
You're all set!

CardBot 0.1.0
Scanning for memory cards...
>
```

Press Enter to accept default `~/Pictures/Ingest`, or type a custom path. This sets the base folder where `CardBot_YYMMDD_001/` folders will be created.

### Corrupted Config

If config file is invalid or unreadable:

```
CardBot 0.1.0
Warning: Config file corrupted (backed up to ~/.cardbot/config.json.bak)

When a card is detected, where would you like the copy destination to be?
Destination [~/Pictures/Ingest] >
```

After first-run setup, the destination is displayed on startup:

```
CardBot 0.1.0
Destination: /Volumes/Backup
Scanning for memory cards...

>
```

## Error States

### Card Removed During Analysis

```
[2025-03-08 21:15:33] Card removed during analysis: /Volumes/NIKON Z9
```

### Card Removed During Copy

```
[2025-03-08 21:18:45] Copy interrupted: card removed
[2025-03-08 21:18:45] Cleaning up partial files...
[2025-03-08 21:18:46] Cleanup complete: 12.3 GB removed
```

### Read-Only Card (Physical Lock or Permissions)

```
[2025-03-08 21:15:32] Card detected
  Path:     /Volumes/NIKON Z9
  Storage:  494.2 GB / 512.0 GB (97%)
  Brand:    Nikon
  Camera:   NIKON Z 9
  Content:
    2025-03-08   142.3 GB   NEF, JPG, MOV
  Status:   New
  Warning:  Card is read-only (copy available, cannot mark as processed)
────────────────────────────────────────
[a] All  [s] Selects  [v] Videos  [p] Photos  [c] Cancel  [e] Eject  >
```

### No DCIM Directory

```
[2025-03-08 21:15:32] Volume detected: /Volumes/UNTITLED
  Path:     /Volumes/UNTITLED
  Storage:  32.0 GB / 64.0 GB (50%)
  Warning:  No DCIM directory found (not a camera card?)
────────────────────────────────────────
[a] Copy all files  [c] Cancel  [e] Eject  >
```

### Destination Disk Full

```
[2025-03-08 21:15:35] Copy started: NIKON Z9 -> ~/Pictures/Ingest/CardBot_250308_001
[2025-03-08 21:18:20] Error: destination disk full
[2025-03-08 21:18:20] Copied 89.2 GB / 142.3 GB (63%) at 81.2 MB/s
[2025-03-08 21:18:20] Cleanup: removing partial files...
[2025-03-08 21:18:21] Cleanup complete
[2025-03-08 21:18:21] Required: 53.1 GB additional space
```

### Network Destination

```
Destination: /Volumes/RAID/share
Error: Network destinations not supported in this version
Hint: Copy to a local drive, then transfer to network storage
────────────────────────────────────────
[c] Cancel  [⌥D] Change destination  >
```

### Corrupt/Unreadable Files

```
[2025-03-08 21:15:35] Copy started: NIKON Z9 -> ~/Pictures/Ingest/CardBot_250308_001
[2025-03-08 21:16:02] Warning: skipping corrupt file: DCIM/100NIKON/DSC_0001.NEF (read error)
[2025-03-08 21:16:02] Continuing with remaining files...
[2025-03-08 21:22:18] Copy complete: 141.9 GB copied, 1 file skipped, 0 errors
```

### Permission Denied

```
[2025-03-08 21:15:32] Card detected
  Path:     /Volumes/NIKON Z9
  Error:    Permission denied reading card contents
  Hint:     Card may require permission check
────────────────────────────────────────
[e] Eject  >
```

### EXIF Read Failures (Partial Analysis)

```
[2025-03-08 21:15:32] Card detected
  Path:     /Volumes/NIKON Z9
  Storage:  494.2 GB / 512.0 GB (97%)
  Brand:    Nikon
  Camera:   Unknown (could not read camera model)
  Content:
    2025-03-08   142.3 GB   NEF, JPG, MOV
  Status:   New
  Note:     3 files could not be analyzed (partial EXIF)
────────────────────────────────────────
[a] All  [s] Selects  [v] Videos  [p] Photos  [c] Cancel  [e] Eject  >
```

### Timeout

If no user input for 30 seconds:

```
[a] All  [s] Selects  [v] Videos  [p] Photos  [c] Cancel  [e] Eject  >
[2025-03-08 21:16:02] CardBot has timed out.
[2025-03-08 21:16:02] Monitoring for cards...
>
```

Timeout message appears inline, then returns to monitoring prompt. Card remains mounted.

### Summary Line Formats

| Outcome | Format |
|---------|--------|
| Success | `Copy complete: 255.6 GB in 6m 42s (85.2 MB/s), 0 errors` |
| With warnings | `Copy complete: 255.6 GB in 6m 42s (85.2 MB/s), 3 files skipped, 0 errors` |
| With errors | `Copy failed: 89.2 GB in 4m 12s (82.1 MB/s), 2 errors (see log)` |
| Cancelled | `Copy cancelled by user: 45.2 GB copied` |
| Timeout | `CardBot has timed out.` |

### Error Handling Philosophy

- **Fatal errors**: Stop immediately, cleanup partial files, report clearly
- **Recoverable errors**: Log warning, continue with remaining files
- **User can retry**: Leave partial state intact (user can retry with `--overwrite`)
- **Always report**: Every error/warning includes timestamp and specific file/action
