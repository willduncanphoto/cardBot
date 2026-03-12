# CardBot — Supported File Types

File type classification as implemented in `internal/analyze/analyze.go`.
Only files matching these extensions are counted and displayed. All others are skipped.

## Photo Extensions (`photoExts`)

### RAW Files

| Extension | Brand | Format |
|-----------|-------|--------|
| `.NEF` | Nikon | Nikon Electronic Format |
| `.NRW` | Nikon | Nikon RAW (Coolpix) |
| `.CR2` | Canon | Canon RAW 2 |
| `.CR3` | Canon | Canon RAW 3 (ISO BMFF) |
| `.CRW` | Canon | Canon RAW (older) |
| `.ARW` | Sony | Sony Alpha RAW |
| `.SRF` | Sony | Sony RAW (older) |
| `.SR2` | Sony | Sony RAW 2 |
| `.RAF` | Fujifilm | Fuji RAW |
| `.ORF` | Olympus | Olympus RAW Format |
| `.RW2` | Panasonic | Panasonic RAW 2 |
| `.DNG` | Various | Digital Negative (Adobe/Leica/Pentax) |
| `.PEF` | Pentax | Pentax Electronic Format |
| `.3FR` | Hasselblad | Hasselblad RAW |
| `.IIQ` | Phase One | Intelligent Image Quality |

### Compressed Images

| Extension | Format |
|-----------|--------|
| `.JPG` / `.JPEG` | JPEG |
| `.TIF` / `.TIFF` | TIFF |
| `.HEIC` / `.HEIF` | High Efficiency Image |
| `.PNG` | PNG |

## Video Extensions (`videoExts`)

| Extension | Format |
|-----------|--------|
| `.MOV` | QuickTime |
| `.MP4` | MPEG-4 |
| `.AVI` | AVI |
| `.MXF` | Material Exchange Format |
| `.MTS` | AVCHD |
| `.M2TS` | Blu-ray BDAV |
| `.R3D` | RED RAW |
| `.BRAW` | Blackmagic RAW |

## EXIF-Supported Extensions (`supportedExif`)

These files are opened for EXIF extraction (date, camera model, star rating).
Files not in this list use file modification time for date grouping.

| Extension | EXIF | XMP Rating |
|-----------|------|------------|
| `.JPG` / `.JPEG` | ✅ | ✅ |
| `.NEF` | ✅ | ✅ |
| `.NRW` | ✅ | ✅ |
| `.CR2` | ✅ | ✅ |
| `.CR3` | ✅ | ✅ |
| `.CRW` | ✅ | ✅ |
| `.ARW` | ✅ | ✅ |
| `.SRF` | ✅ | ✅ |
| `.SR2` | ✅ | ✅ |
| `.RAF` | ✅ | ✅ |
| `.ORF` | ✅ | ✅ |
| `.RW2` | ✅ | ✅ |
| `.DNG` | ✅ | ✅ |
| `.PEF` | ✅ | ✅ |
| `.TIF` / `.TIFF` | ✅ | ✅ |
| `.HEIC` / `.HEIF` | ✅ | ✅ |

**Not EXIF-parsed:** PNG (no EXIF), 3FR, IIQ (not supported by imagemeta), all video formats.

Video files use file modification time for date grouping.

## Skipped Files

These are never counted or displayed:

- Hidden files (`.` prefix): `.DS_Store`, `._*`, `.Trashes/`
- Non-media files: `.DAT`, `.DSC`, `.THM`, `.LRV`, `.XMP`, etc.
- Files with no extension
- Files with unrecognized extensions

## Extension Handling

- Extensions are normalized to uppercase for comparison (`normalizeExt`)
- Display uses uppercase: `NEF`, `MOV`, `JPG`
- Matching is case-insensitive (`.nef`, `.Nef`, `.NEF` all match)

## Map Relationships

```
photoExts (22 entries)    — "count as photo"
videoExts (8 entries)     — "count as video"
supportedExif (19 entries) — "try EXIF parsing"

photoExts ∩ supportedExif = all of supportedExif
photoExts - supportedExif = PNG, 3FR, IIQ (no EXIF support)
videoExts ∩ supportedExif = ∅ (videos never EXIF-parsed)
```
