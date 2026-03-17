# 050 — v0.5.0 Copy UI/UX Focus

**Status:** PLANNING  
**Target:** v0.5.0  
**Focus:** Copy UI/UX settings and operation  

---

## Theme

The copy operation is the core of CardBot — it's where users spend the most time waiting and watching. Current implementation works but is minimal. v0.5.0 makes the copy experience informative, predictable, and polished.

---

## Primary Goals

### 1. ETA During Copy

**Problem:** Users stare at progress bars with no sense of time remaining.

**Solution:** Add ETA calculation based on current transfer rate and remaining bytes.

```
[2026-03-17T14:23:12] Copying... 1,247/3,051 files  14.2 GB/32.1 GB (44%)  ETA 2m 14s
```

**Technical notes:**
- Calculate rolling average speed over last 10 seconds
- ETA = remaining bytes / average speed
- Update every 2 seconds (same as progress)
- Show "--:--" if speed is 0 or erratic

---

### 2. Resume Interrupted Copies

**Problem:** Card pulled mid-copy or app crashed = start over from 0.

**Solution:** Dotfile tracks partial copy state. On re-insert, detect partial copy and offer resume.

**Flow:**
```
Status:   Partial copy (1,247/3,051 files, 14.2 GB copied on 2026-03-17T14:20:05)

[r] Resume Copy  [x] Start Fresh  [e] Eject
```

**Technical notes:**
- Extend dotfile v2 schema with `partial` section
- Track: files completed, bytes completed, last successful file path
- Resume = skip files already copied (size match verification)
- "Start Fresh" = clear partial state and re-copy everything

**Dotfile schema addition:**
```json
{
  "$schema": "cardbot-dotfile-v2",
  "copies": [...],
  "partial": {
    "mode": "all",
    "started": "2026-03-17T14:20:05",
    "files_completed": 1247,
    "bytes_completed": 15241783296,
    "last_file": "100NIKON/DSC_8234.NEF"
  }
}
```

---

### 3. Pre-Copy Preview

**Problem:** Users don't know what "Copy Selects" or "Copy Photos" will actually copy until they run it.

**Solution:** Show a preview of what will be copied before starting.

**Flow:**
```
[a] Copy All  [s] Copy Selects  [p] Copy Photos  [v] Copy Videos  [?] Help  > s

Copy Selects: 247 files, 4.2 GB
  2026-02-27: 12 files (NEF)
  2026-02-26: 235 files (NEF, JPG)

[Enter] to proceed  [x] Cancel
```

**Technical notes:**
- Reuse analyze results (we already have file list + filters)
- Show per-date breakdown like the main card info
- Allow cancellation before copy starts
- This is NOT dry-run — it's a preview using already-cached data

---

### 4. Better Progress Display

**Problem:** Single-line progress with \r breaks on narrow terminals or if user resizes.

**Solution:** Multi-line progress that stays readable.

**Current:**
```[2026-03-17T14:23:12] Copying... 1,247/3,051 files  14.2 GB/32.1 GB (44%)    ```

**Proposed:**
```
[2026-03-17T14:23:12] Copying...
  1,247/3,051 files  (40.8%)
  14.2 GB / 32.1 GB
  ~45 MB/s  ETA 2m 14s

  Current: 260227T143052_1247.NEF
```

**Technical notes:**
- Multi-line = no \r clobbering issues
- Shows current filename (helps user see it's moving)
- Speed and ETA on same line for quick reading
- Press [\] to cancel (unchanged)

---

### 5. Post-Copy Summary

**Problem:** Copy finishes, user gets a single line. No sense of what happened.

**Solution:** Brief summary of what was copied, what was skipped, and where.

```
[2026-03-17T14:25:26] Copy complete ✓

Copied:   1,247 files, 4.2 GB
Skipped:  0 files (0 B already existed)
To:       ~/Pictures/CardBot/2026-02-26, 2026-02-27
Time:     2m 14s (31.4 MB/s avg)

[e] Eject  [x] Exit
```

**Technical notes:**
- Track skipped files (already existed with correct size)
- Show destination folders (could be multiple dates)
- Average speed over entire operation
- Clearer next-action prompt

---

## Secondary Goals (If Time Permits)

### 6. Video/Photo Destination Separation

**Problem:** Videos and photos go to same destination. Some workflows want separation.

**Solution:** Config option to split destinations.

```json
{
  "destination": {
    "photos": "~/Pictures/CardBot",
    "videos": "~/Movies/CardBot"
  }
}
```

**UI impact:**
- Copy All respects the split
- Copy Photos goes to photos destination
- Copy Videos goes to videos destination
- Copy Selects goes to photos destination (since ratings are on photos)

---

### 7. Copy Settings Quick Toggle

**Problem:** User wants to change naming mode or buffer size mid-session.

**Solution:** Add quick settings command.

```
[?] Help shows:
...
[+] Settings — change copy options
```

**Settings menu:**
```
Current settings:
  Naming:   Timestamp + sequence
  Buffer:   256 KB
  
[n] Change naming  [b] Change buffer  [x] Back
```

**Technical notes:**
- Settings are per-session (not saved to config)
- Allows quick experimentation without --setup
- Or should these persist? Need to decide.

---

## Non-Goals for 0.5.0

| Item | Why Deferred |
|------|--------------|
| Raw terminal mode (single-key, no Enter) | General UX improvement, not copy-specific. Save for 0.6.0 |
| TUI library (bubbletea, etc.) | Too heavy. Multi-line progress is sufficient. |
| Checksum verification (xxhash) | Still deferred. Size verification is working. |
| Batch operations (multiple cards simultaneously) | Architecture change. Keep sequential for now. |
| Performance profiling | Only if copy is slow. Let's add ETA first and see. |

---

## Success Metrics

- [ ] ETA is within 10% accuracy after first 30 seconds
- [ ] Resume works after controlled interruption (pull card mid-copy)
- [ ] Preview shows accurate file counts before copy starts
- [ ] Progress display is readable on 80-column terminal
- [ ] Post-copy summary fits on one screen

---

## Open Questions

1. **Resume state on different machine?** If user moves card to different computer with different destination path, should we still offer resume? (Probably not — destination mismatch check.)

2. **Partial resume granularity?** File-level (skip whole files) or block-level (resume partial file)? File-level is much simpler and sufficient for network/USB interruptions.

3. **Preview for large cards?** Card with 10K files — preview could be huge. Cap at top N dates? Or show summary only?

4. **Settings persistence?** If user changes buffer size in [+] menu, should it update config.json or just this session?

---

## Dependencies

- No new external dependencies
- All changes internal to `internal/copy`, `internal/dotfile`, `internal/app`

---

## Risks

| Risk | Mitigation |
|------|------------|
| Dotfile schema change breaks old versions | v2 schema already extensible; add `partial` as optional field |
| ETA wildly inaccurate on variable-speed cards | Use rolling average; show "calculating..." first 10s |
| Multi-line progress breaks piped output | Detect if stdout is TTY; fallback to single line if piped |

---

## Execution Order

1. ETA calculation (low risk, adds value immediately)
2. Better progress display (builds on ETA work)
3. Post-copy summary (uses same tracking data)
4. Resume infrastructure (dotfile changes)
5. Pre-copy preview (UI polish)
6. Video/photo split (if time)
7. Settings toggle (nice to have)
