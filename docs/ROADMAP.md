# CardBot Roadmap

## Current: 0.3.4 (Stable)

**Status:** Feature-complete for renaming. Ready for real-world validation.

**What's Implemented:**
- Two naming modes: Camera original (DSC_xxxx.NEF) vs Timestamp + sequence (YYMMDDTHHMMSS_xxxx.NEF)
- Fixed 4-digit sequence (0001-9999, loops back to 0001)
- Chronological ordering by EXIF capture time
- Dry-run with rename preview
- Setup prompts explain the problem/solution

**Known Limitations (Documented):**
- 10000+ files in one day: sequence loops (0001→9999→0001). Very rare edge case.
- Multi-camera same second: collision risk. See "Stuff to Think About" below.
- Re-copy detection: uses file size, not rename mapping. May re-copy in timestamp mode.

---

## 0.4.0 — TBD

**Status:** Not yet defined. Candidates:
- Video workflow separation (photos → pictures, videos → movies)
- Config schema v3 with migration harness
- Linux platform support

---

## 0.5.0 — TBD

Future major feature. See "Stuff to Think About" for candidates.

---

## Stuff to Think About (Unscheduled)

### Multi-Camera Collision Prevention

**Problem:** Two cameras (Z9 + Z8) shooting same wedding:
```
Z9: 260314T143052_0001.NEF
Z8: 260314T143052_0001.NEF  ← collision!
```

**Also:** Two Z9 bodies (primary + backup) report identical Make/Model.

**Potential Solutions:**

| Approach | Example | Notes |
|----------|---------|-------|
| Camera prefix | `Z9_260314T143052_0001.NEF` | Human-readable |
| Camera suffix | `260314T143052_0001_Z9.NEF` | Timestamp first |
| Body serial | `Z9_6746_260314T143052_0001.NEF` | Handles two Z9s |
| Pre-ingest audit | Scan all cards, assign global sequence | Heavy UX |
| Per-camera subfolders | `2026-03-14/Z9/...` | Loses chronology |

**Rejected:** Subsecond timestamp (Z9 only has 1/100s precision, insufficient for 20fps burst). See [Z9-EXIF-ANALYSIS.md](Z9-EXIF-ANALYSIS.md).

**Blockers:**
- Need to think through collision detection workflow
- How to handle same-camera-wrong-time (two Z9s, one clock wrong)
- Smartphone/drone files without EXIF Model

**Verdict:** Parking lot. Not 0.4.0 priority.

---

### Video Workflow Separation

**Goal:** Photos and videos to different destinations.

**Use Case:** Photos → `~/Pictures/CardBot/` (Lightroom), Videos → `~/Movies/CardBot/` (Premiere)

**Features:**
- Separate `destination.video_path` config
- `[v]` Copy Videos to video destination
- `[a]` Copy All splits content
- Independent dotfile tracking

---

### Dynamic Per-Date Digits

If single calendar day has >999 files, use 4-digit for that day only.

**Deferred:** Rare edge case, adds complexity.

---

### Linux/Windows Support

**Blockers:** Platform-specific detection, hardware info, speed test, eject.

---

### Single-Key Input

Raw terminal mode (no Enter required). Power user polish.

---

## Completed Milestones

### 0.3.x — File Renaming
- Timestamp-based renaming with EXIF capture time
- Fixed 4-digit sequence (0001-9999)
- Chronological ordering across DCIM folders
- Dry-run with preview

### 0.2.x — Selective Copy
- `[s]` Starred/picked files only
- `[p]` Photos only, `[v]` Videos only
- Dotfile v2 with multi-mode tracking

### 0.1.x — Foundation
- Core copy engine with progress
- Card detection (macOS)
- EXIF analysis with parallel workers
- Self-updater
- Selective copy framework

---

## Version History

| Version | Date | Key Feature |
|---------|------|-------------|
| 0.3.4 | 2026-03-15 | Simplified prompt, dead code removal, stale comment fixes |
| 0.3.3 | 2026-03-15 | Cleanup pass: docs sync, dead code removal, prompt consistency |
| 0.3.2 | 2026-03-14 | Simplified UX, fixed 4-digit sequence, Z9 EXIF analysis |
| 0.3.1 | — | (Skipped, merged into 0.3.2) |
| 0.3.0 | — | Initial renaming implementation |
| 0.2.9 | 2026-03-12 | Self-update fixes |
| 0.2.0 | 2026-03-10 | Selective copy complete |
| 0.1.8 | 2026-03-09 | Code health refactor |
| 0.1.0 | 2026-03-08 | Initial release |
