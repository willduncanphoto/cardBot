# CardBot — Active Work

See [ROADMAP.md](ROADMAP.md) for full future planning.

## Current: 0.3.4

**Status:** Ready for real-world validation.

### Before Tagging 0.3.4
- [ ] Real-world test: Z9 card, timestamp mode
- [x] Verify 4-digit sequence behavior (0001-9999)
- [x] Verify dry-run preview output
- [ ] Verify re-copy behavior (expected: may re-copy due to no mapping log)

### Known Limitations (Acceptable for 0.3.4)
- 10000+ files/day: sequence loops (0001→9999→0001). Very rare, documented.
- Multi-camera same second: collision risk. See ROADMAP "Stuff to Think About".
- Re-copy: uses size check, may re-copy renamed files.

---

## 0.4.0 — UI/UX Review

**Goal:** Cleaner startup and card info output.

### Changes (Completed)
- [x] **Move startup config under card info**
  - Removed: `[timestamp] Copy path...` and `[timestamp] Naming...` at startup
  - Now showing after card detected, without timestamps:
    ```
    Copy to:  ~/Pictures/CardBot
    Naming:   Camera original (DSC_xxxx.NEF)
    ```
- [x] **Reduce visual noise** - removed spinner, fewer separators, cleaner layout
- [x] **Consistent naming display** - 4-digit sequence format everywhere

### Remaining for 0.4.0
- [ ] Evaluate: What info is needed at each stage vs what can be hidden
- [ ] Technical EXIF Display Mode (raw EXIF values)

---

## Future Versions (TBD)

See ROADMAP.md for candidates:
- Video workflow separation
- Config schema v3 migration harness
- Linux platform support

Multi-camera collision prevention is **parked**.

---

## UI/UX Updates (Future)

- [ ] **Technical EXIF Display Mode**
  - Show raw EXIF values in card info:
    ```
    Make    : NIKON CORPORATION
    Model   : NIKON Z 9
    ```
  - Instead of cleaned "Nikon Z 9"
  - Toggle or config option for technical vs friendly display
  
- [ ] **Card Info Layout Refresh**
  - More technical/professional appearance
  - Raw EXIF values where meaningful
  - Cleaner alignment

## Quick Fixes (Any Release)

- [ ] Add "OM System" brand color (when confirmed)
- [ ] Config path display command (`cardbot --config`)

---

## Done

See ROADMAP.md for completed milestones.
