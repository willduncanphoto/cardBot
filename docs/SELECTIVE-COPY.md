# Selective Copy — Design

Spec for 0.1.9 selective copy modes: `[s]` Selects, `[p]` Photos, `[v]` Videos.

---

## Key Insight: The Dotfile Is Informational

The copy engine already skips files that exist at the destination with the correct
size. This means **the dotfile doesn't control what gets copied** — the filesystem
does. The dotfile's only job is telling the user what has been done to this card.

This simplifies everything. We don't need precise per-file tracking in the dotfile.
We just need to record which modes completed and when.

---

## Real-World Workflows

These are the scenarios a working photographer would hit:

### 1. Full Backup (today's workflow)
Insert card → `[a]` Copy All → eject. Simple. No change needed.

### 2. Field Triage
On location with a laptop. Insert card → `[s]` Copy Selects → eject.
Quick export of star-rated images for social/client preview.
Card goes back in camera. Full backup happens at the studio later.

### 3. Staggered Copy
Insert card → `[p]` Copy Photos → start editing immediately.
Later: `[v]` Copy Videos → when drive space is available.

### 4. Selects Then All
Insert card → `[s]` Copy Selects → quick review.
Then `[a]` Copy All → full backup. Selects already exist at destination,
copy engine skips them, copies remaining files.

### 5. Re-Insert After Copy
Card was fully copied last week. Re-insert today.
Status shows "Copied on 2026-03-08T15:04:05". User can still press `[a]`
to re-copy (catches any files added since last copy — engine skips existing).

### 6. Multi-Shoot Same Card
Photographer shoots, copies all, shoots again on same card.
Re-insert: status shows "Copied on ..." but new files exist.
Press `[a]` — new files are copied, old files are skipped. Dotfile updated.

### 7. Read-Only Card
Write-protected card. Copy succeeds but dotfile can't be written.
Next insert: card shows "New" even though files exist at destination.
User presses `[a]` — engine skips all files (already exist). Harmless.

### 8. Different Destinations
User copies photos to ~/Pictures/CardBot, changes config, copies videos
to ~/Videos/CardBot. Both copies succeed. Dotfile records both destinations.
Edge case but not broken — each entry tracks where its files went.

---

## Dotfile Schema v2

Array of copy records, one per mode. When a mode is re-run, its entry is
**replaced** (updated timestamp and counts). New modes are appended.

```json
{
  "$schema": "cardbot-dotfile-v2",
  "copies": [
    {
      "mode": "selects",
      "timestamp": "2026-03-12T12:31:05-07:00",
      "destination": "/Users/user/Pictures/CardBot",
      "files_copied": 42,
      "bytes_copied": 1234567890,
      "verified": true,
      "cardbot_version": "0.1.9"
    },
    {
      "mode": "all",
      "timestamp": "2026-03-12T14:22:10-07:00",
      "destination": "/Users/user/Pictures/CardBot",
      "files_copied": 3051,
      "bytes_copied": 96424837120,
      "verified": true,
      "cardbot_version": "0.1.9"
    }
  ]
}
```

### Schema Migration

- CardBot 0.1.9+ reads v1 and v2
- v1 file: treat as a single `copies` entry with `mode` from the existing field
- v2 file read by old CardBot (<0.1.9): unknown schema → treated as "New"
  (safe — user just sees "New" status, no data loss)
- Writing always produces v2

### Why an Array, Not a Map

- Preserves insertion order (visible in JSON)
- Simple upsert: find entry with matching mode, replace; or append
- JSON maps don't guarantee order

---

## Status Display

The status line reads the dotfile and shows what's been done.

### Rules (evaluated in order)

1. No dotfile or parse error → `New`
2. Any entry with `mode: "all"` → `Copied on <timestamp>`
3. Single selective mode → `<Mode> copied on <timestamp>`
4. Multiple selective modes → `<Mode> + <Mode> copied on <latest timestamp>`

### Examples

| Dotfile State | Status Line |
|---------------|-------------|
| No `.cardbot` file | `New` |
| `[{mode: "all", ...}]` | `Copied on 2026-03-12T14:22:10` |
| `[{mode: "selects", ...}]` | `Selects copied on 2026-03-12T12:31:05` |
| `[{mode: "photos", ...}]` | `Photos copied on 2026-03-12T12:31:05` |
| `[{mode: "videos", ...}]` | `Videos copied on 2026-03-12T14:22:10` |
| `[{mode: "photos"}, {mode: "videos"}]` | `Photos + Videos copied on 2026-03-12T14:22:10` |
| `[{mode: "selects"}, {mode: "all"}]` | `Copied on 2026-03-12T14:22:10` |

The timestamp shown is always the **latest** across all entries (or the "all" entry
if present).

### Mode Display Names

| Mode Key | Display Name |
|----------|-------------|
| `all` | (none — just "Copied") |
| `selects` | `Selects` |
| `photos` | `Photos` |
| `videos` | `Videos` |

---

## Session Guard (Repeat Copy Prevention)

After a successful copy, prevent the same (or superseded) mode from running again
**in the same session** without ejecting and re-inserting.

### Per-Session State

Replace the current `copied bool` with:

```go
copiedModes map[string]bool // modes completed this session
```

### Guard Logic

| Command | Blocked If | Message |
|---------|-----------|---------|
| `[a]` | `"all"` in copiedModes | `Already copied.` |
| `[s]` | `"selects"` or `"all"` in copiedModes | `Selects already copied.` |
| `[p]` | `"photos"` or `"all"` in copiedModes | `Photos already copied.` |
| `[v]` | `"videos"` or `"all"` in copiedModes | `Videos already copied.` |

### What "All" Supersedes

`[a]` Copy All copies every file on the card. After it completes:
- `[s]`, `[p]`, `[v]` are all blocked (those files are already at the destination)
- This avoids pointless re-copies in the same session

### What "All" Does NOT Do

- `[a]` does not clear or modify previous selective entries in the dotfile
- It adds/replaces the `"all"` entry
- Previous selective entries remain as history

### Re-Insert Behavior

On card removal, `copiedModes` is cleared. Re-inserting the same card starts fresh.
The dotfile shows historical status, but the user can re-run any mode.

This is correct because:
- New files may have been added between sessions
- The copy engine handles existing files via size-check skip
- No harm in re-running a mode

---

## Prompt Changes

The prompt should reflect what actions are available.

| State | Prompt |
|-------|--------|
| No copies yet | `[a] Copy All  [s] Selects  [p] Photos  [v] Videos  [e] Eject  [x] Exit  [?]  >` |
| After any selective copy | Same as above (other modes still available) |
| After `[a]` Copy All | `[e] Eject  [x] Done  [?]  >` |
| Invalid card (no DCIM) | `[e] Eject  [x] Exit  [?]  >` |

**Note:** After "all" is copied, all copy options disappear since everything is done.
After selective copies, remaining modes still show. The guard logic handles repeats
if the user presses an already-completed mode.

**Simplification for 0.1.9:** Keep the current prompt format. Adding `[s] [p] [v]`
to the default prompt makes it very wide. The help screen `[?]` shows all modes.
The prompt stays as `[a] Copy All  [e] Eject  [x] Exit  [?]  >` and the user
discovers selective modes via help. Revisit prompt layout if user feedback says
the modes aren't discoverable enough.

---

## Copy Engine Changes

### File Filter

Add a filter function to `copy.Options`:

```go
type Options struct {
    CardPath      string
    DestBase      string
    BufferKB      int
    DryRun        bool
    AnalyzeResult *analyze.Result
    Filter        func(relPath string, ext string) bool // nil = copy all
}
```

During the walk phase, skip files where `Filter` returns false:

```go
if opts.Filter != nil && !opts.Filter(rel, ext) {
    return nil // skip
}
```

The filter is constructed by the caller based on mode:

| Mode | Filter Logic |
|------|-------------|
| `all` | `nil` (no filter — copy everything) |
| `photos` | `analyze.IsPhoto(ext)` |
| `videos` | `analyze.IsVideo(ext)` |
| `selects` | `analyzeResult.FileRatings[relPath] > 0` |

### Disk Space Preflight

The preflight check currently uses total card size. With filters, it should
scope to the selected subset. The walk phase already computes `totalBytes`
after filtering — the existing preflight logic works without changes.

---

## Analyzer Changes

### Per-File Star Ratings

Currently the analyzer counts starred files (`Starred int`) but doesn't track
**which** files are starred. The selects filter needs to know.

Add to `analyze.Result`:

```go
FileRatings map[string]int // relPath → star rating (1-5), only non-zero entries
```

This is populated during EXIF extraction alongside the existing `starred` counter.
Only files with rating > 0 are stored (memory efficient — most files have no rating).

### Export Photo/Video Classification

The analyzer already has `photoExts` and `videoExts` maps but they're unexported.
Export helper functions for the copy filter:

```go
func IsPhoto(ext string) bool { return photoExts[ext] }
func IsVideo(ext string) bool { return videoExts[ext] }
```

`ext` is uppercase without dot (e.g. `"NEF"`, `"MOV"`), matching the existing convention.

---

## Interaction Matrix

Every combination of "previous state" × "user action" and what happens.

### Copy Actions

| Previous State | User Presses | What Happens |
|----------------|-------------|-------------|
| New card | `[a]` | Copy all files. Dotfile: `[{mode: "all", ...}]` |
| New card | `[s]` | Copy starred files. Dotfile: `[{mode: "selects", ...}]` |
| New card | `[p]` | Copy photos. Dotfile: `[{mode: "photos", ...}]` |
| New card | `[v]` | Copy videos. Dotfile: `[{mode: "videos", ...}]` |
| Selects copied (this session) | `[a]` | Copy all. Engine skips selects (exist). Dotfile adds `"all"` entry. |
| Selects copied (this session) | `[s]` | Blocked: "Selects already copied." |
| Selects copied (this session) | `[p]` | Copy photos. Engine skips starred photos (exist). Dotfile adds `"photos"`. |
| Photos copied (this session) | `[v]` | Copy videos. Dotfile adds `"videos"`. |
| Photos copied (this session) | `[p]` | Blocked: "Photos already copied." |
| All copied (this session) | `[a]` | Blocked: "Already copied." |
| All copied (this session) | `[s]` | Blocked: "Selects already copied." |
| All copied (this session) | `[p]` | Blocked: "Photos already copied." |
| All copied (this session) | `[v]` | Blocked: "Videos already copied." |

### Re-Insert Scenarios

| Dotfile State | User Presses | What Happens |
|---------------|-------------|-------------|
| All copied (previous session) | `[a]` | Runs. Engine skips existing files. Harmless. |
| All copied (previous session) | `[p]` | Runs. Engine skips existing photos. Harmless. |
| Photos copied (previous session) | `[a]` | Runs. Copies videos + unstarred non-photo files. |
| Photos copied (previous session) | `[p]` | Runs. Engine skips existing. Catches new photos. |
| Selects copied (previous session) | `[a]` | Runs. Copies everything. Skips existing selects. |

### Edge Cases

| Scenario | Behavior |
|----------|----------|
| Card has 0 starred files, user presses `[s]` | "No starred files found on this card." |
| Card has 0 videos, user presses `[v]` | "No video files found on this card." |
| Card has 0 photos, user presses `[p]` | "No photo files found on this card." |
| Card removal during selective copy | Same as today: cancel, show partial count. |
| `[\]` cancel during selective copy | Same as today: cancel, show partial count. |
| Read-only card + selective copy | Copy succeeds, dotfile write fails, warning shown. |
| `--dry-run` + selective mode | "Dry-run: would copy N photos to <dest>" |

---

## Implementation Plan

### Step 1: Analyzer — FileRatings + Exported Helpers
- Add `FileRatings map[string]int` to `Result`
- Populate during EXIF phase (where `starred++` currently happens)
- Add `IsPhoto(ext)` and `IsVideo(ext)` exported functions
- Tests: verify FileRatings populated, verify IsPhoto/IsVideo

### Step 2: Copy Engine — Filter Support
- Add `Filter func(relPath, ext string) bool` to `Options`
- Apply filter in walk phase
- No other changes needed (preflight already scoped to walked files)
- Tests: copy with photo-only filter, copy with custom filter

### Step 3: Dotfile v2
- New schema with `copies` array
- Read: parse v1 (migrate to single-entry array) and v2
- Write: always produce v2
- Upsert logic: replace existing mode entry or append
- `ReadStatus` returns structured data (modes, timestamps)
- `FormatStatus` renders the status display line
- Tests: v1 migration, v2 round-trip, upsert, multi-mode status formatting

### Step 4: Wire Up Commands
- `copy_cmd.go`: extract `copyFiltered(card, mode, filter)` from `copyAll`
- `app.go`: replace `copied bool` with `copiedModes map[string]bool`
- `app.go`: handleInput for `s`, `p`, `v` calls `copyFiltered` with appropriate filter
- Guard logic: check copiedModes before starting copy
- Pre-copy validation: check if card has files matching the filter (starred count, photo count, video count)
- Dotfile write: pass mode to dotfile.Write, upsert into copies array
- `display.go`: update `printPrompt` for post-all vs post-selective state
- `display.go`: remove strikethrough from help text for implemented modes

### Step 5: Docs & Tests
- Update DOTFILE.md with v2 schema
- Update OUTPUT.md with selective copy examples
- Update README.md command table
- Update TODO.md / ROADMAP.md

---

## Open Questions — Resolved

### Should `[a]` mark selective modes as complete?
**No.** `[a]` only records an `"all"` entry. Previous selective entries remain as history.
The status display logic handles the "all supersedes" display.

### If photos were copied and user runs `[a]`, skip photo files?
**Yes, automatically.** The copy engine's size-check skip handles this. Photo files already
exist at the destination with correct size → skipped. No special dotfile logic needed.

### `completed_modes` field vs `copies` array?
**`copies` array.** It stores richer data (timestamp, counts, destination per mode)
and enables history display. A flat `completed_modes` list loses the per-mode stats.

### Should photos + videos = all?
**No.** `"all"` is its own mode. `photos + videos` doesn't include potential future file
types or files that aren't classified as either. They're tracked independently.
