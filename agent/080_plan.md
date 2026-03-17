# 080 — v0.8.0 Copyright Info Injection

**Status:** PLANNING  
**Target:** v0.8.0  
**Focus:** EXIF/XMP copyright and creator metadata injection  

---

## Theme

Professional photographers need their copyright embedded in every file at ingest time. CardBot should optionally inject copyright, creator name, and usage rights into each copied file's metadata — silently, reliably, and without slowing down the copy operation.

---

## Goals

### 1. Config-Based Copyright Presets

**Problem:** Copyright needs to be in every file, but typing it manually is error-prone.

**Solution:** Configurable copyright presets applied during copy.

```json
{
  "$schema": "cardbot-config-v1",
  "copyright": {
    "enabled": true,
    "preset": "default",
    "presets": {
      "default": {
        "artist": "William Duncan",
        "copyright": "© 2026 William Duncan. All rights reserved.",
        "usage_terms": "Contact for licensing",
        "creator_city": "Los Angeles",
        "creator_state": "CA",
        "creator_country": "USA"
      },
      "work_for_hire": {
        "artist": "William Duncan",
        "copyright": "© 2026 Client Name. Work for hire.",
        "usage_terms": "Exclusive rights assigned to client"
      }
    }
  }
}
```

**UI:**
```
[a] Copy All  [e] Eject  [x] Exit  [?] Help  > a

Copying all files to ~/Pictures/CardBot...
[2026-03-17T14:23:12] Injecting copyright metadata... ✓
[2026-03-17T14:23:12] Copying... 1,247/3,051 files
```

---

### 2. EXIF and XMP Injection

**Technical approach:**

| Field | EXIF Tag | XMP Property | Notes |
|-------|----------|--------------|-------|
| Artist/Creator | 0x013B (Artist) | dc:creator | Required |
| Copyright | 0x8298 (Copyright) | dc:rights | Required |
| DateTimeOriginal | 0x9003 | photoshop:DateCreated | Preserve existing |
| Usage Terms | — | xmpRights:UsageTerms | XMP only |
| Creator Address | — | Iptc4xmpCore:CreatorCity/State/Country | XMP only |

**Implementation strategy:**

Option A: In-place modification (risky)
- Modify EXIF/XMP in source file before copy
- Risk: corrupts source card if interrupted
- **Rejected**

Option B: Sidecar injection (safer, preferred)
- Copy file to destination
- Inject metadata into destination copy only
- Source card untouched
- **Accepted**

**Libraries:**
- `evanoberholster/imagemeta` — already in use, but read-only
- Need write capability: evaluate `github.com/dsoprea/go-exif/v3` or `github.com/rwcarlsen/goexif` for EXIF writing
- XMP: may need custom writer (XMP is just XML in a specific packet format)

**File support priority:**
1. JPEG — easiest, TIFF-based EXIF
2. TIFF/RAW — NEF, CR2, ARW (TIFF-based)
3. CR3 — ISO BMFF, harder
4. HEIC — ISO BMFF, harder
5. Video — MOV/MP4 (XMP can be embedded, but different format)

---

### 3. Smart Date Handling

**Problem:** Copyright year should match capture year, not current year.

**Solution:** Extract `DateTimeOriginal` first, use that year in copyright string.

```json
{
  "copyright": {
    "copyright": "© {{year}} William Duncan. All rights reserved."
  }
}
```

Becomes: `© 2026 William Duncan. All rights reserved.` for files shot in 2026.

If `DateTimeOriginal` is missing, fall back to:
1. File modification time
2. Current year (last resort)

---

### 4. Selective Injection

**Problem:** Not every copy needs copyright (e.g., personal vs. client work).

**Solution:** Per-copy-mode injection rules.

```json
{
  "copyright": {
    "enabled": true,
    "apply_to": ["all", "selects"],
    "skip_for": ["photos", "videos"]
  }
}
```

**Runtime override:**
```
[2026-03-17T14:23:12] Card detected
  ...
  Copyright:  © 2026 William Duncan (enabled)

[a] Copy All  [c] Toggle Copyright  [e] Eject  > c
  Copyright:  Disabled for this copy

[a] Copy All  [c] Toggle Copyright  [e] Eject  > a
Copying without copyright injection...
```

---

### 5. Verification

**Problem:** User needs to know injection worked.

**Solution:** Sample verification + logging.

**Per-file:** Too slow. Don't verify every file.

**Sample verification:**
- Verify first file copied
- Verify one random file per 100
- Log all injections to `cardbot.log`

```
[2026-03-17T14:23:12] Copyright injected: 260227T143052_0001.NEF
[2026-03-17T14:23:15] Copyright injected: 260227T143052_0100.NEF (sample)
[2026-03-17T14:23:18] Copyright injected: 260227T143052_0200.NEF (sample)
```

**Post-copy summary:**
```
[2026-03-17T14:25:26] Copy complete ✓
Copied:      1,247 files, 4.2 GB
Copyright:   Injected into 1,247 files
Verified:    13 files (sampled)
```

---

## Non-Goals for 0.8.0

| Item | Why Deferred |
|------|--------------|
| Watermarking/visible copyright | Image manipulation too complex, out of scope |
| Full IPTC workflow | XMP is modern standard, IPTC legacy |
| RAW developer settings | Different problem, needs different tool |
| Batch copyright for existing files | CardBot is ingest-time tool, not archive manager |

---

## Open Questions

1. **EXIF write library choice:**
   - `dsoprea/go-exif/v3` — has write support, but large, stalled
   - `rwcarlsen/goexif` — popular, but no write support
   - Fork imagemeta and add write? (maintainer is responsive)
   - Write minimal EXIF writer ourselves? (TIFF structure is documented)

2. **XMP packet format:**
   - XMP must be wrapped in `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>` ... `<?xpacket end="w"?>`
   - Need to preserve existing XMP or create new packet
   - XMP can be embedded in JPEG APP1, TIFF IFD, or sidecar .xmp file

3. **CR3/HEIC support:**
   - These are ISO BMFF (MP4-style box format)
   - EXIF is in a `moov/uuid` box or similar
   - Much harder than TIFF-based formats
   - Defer to 0.8.1+ or accept JPEG/TIFF/RAW only for v0.8.0

4. **Performance impact:**
   - EXIF write requires: parse IFD, modify fields, rewrite file
   - Could add 50-100ms per file = 5 minutes for 3,000 files
   - Solution: async injection pipeline separate from copy? Or accept the cost?

5. **Legal template presets:**
   - Should we ship with common templates (CC-BY, All Rights Reserved, etc.)?
   - Or just let users define their own?

---

## Dependencies

- Evaluate/add EXIF write library (TBD)
- No other new dependencies expected

---

## Risks

| Risk | Mitigation |
|------|------------|
| EXIF write corrupts files | Only write to destination copy, never source; verify samples |
| Library adds significant binary size | Evaluate before committing; prefer minimal implementation |
| Performance unacceptable | Benchmark early; consider async pipeline if needed |
| CR3/HEIC support missing | Document limitation; JPEG/TIFF/RAW covers most pro workflows |

---

## Success Metrics

- [ ] Copyright appears in EXIF of copied JPEGs
- [ ] Copyright appears in XMP of copied RAW files
- [ ] Year in copyright matches capture date year
- [ ] Source card files are never modified
- [ ] Copy with injection is <20% slower than without
- [ ] Works with NEF, CR2, ARW, JPEG (priority formats)

---

## Research Needed

1. EXIF write library evaluation (spike: 1 day)
2. XMP packet format specification review
3. Performance benchmark: injection vs. plain copy
4. Legal review: are our default templates correct? (not legal advice, just accurate)

---

## Related Work

- imagemeta (current read-only library): https://github.com/evanoberholster/imagemeta
- go-exif (write support): https://github.com/dsoprea/go-exif
- Adobe XMP Specification: https://developer.adobe.com/xmp/docs/XMPSpecifications/
- EXIF Specification: JEITA CP-3451C
