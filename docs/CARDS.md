# Memory Card Testing Checklist

Physical cards to test with CardBot. Check off when verified working.

## Brands

### Nikon
- [ ] Nikon Z9 (CFexpress Type B + SD) — **PRIORITY** (primary dev card)
- [ ] Nikon Z8 (CFexpress Type B + SD)
- [ ] Nikon Z7 II (SD)
- [ ] Nikon Z6 II (SD)
- [ ] Nikon D850 (SD + XQD)
- [ ] Nikon D780 (SD)

### Canon
- [ ] Canon R5 (CFexpress + SD)
- [ ] Canon R6 (SD)
- [ ] Canon R7 (SD)
- [ ] Canon 5D IV (CF + SD)

### Sony
- [ ] Sony A1 (CFexpress Type A + SD)
- [ ] Sony A7 IV (SD)
- [ ] Sony A7R V (CFexpress Type A + SD)
- [ ] Sony FX3 (CFexpress Type A + SD)

### Fujifilm
- [ ] Fujifilm X-T5 (SD)
- [ ] Fujifilm X-H2S (CFexpress + SD)
- [ ] Fujifilm GFX 100S (SD)

### Panasonic
- [ ] Panasonic GH6 (CFexpress + SD)
- [ ] Panasonic S5 II (SD)

### Olympus/OM System
- [ ] OM-1 (SD)
- [ ] Olympus E-M1 III (SD)

### Other
- [ ] Leica Q3 (SD)
- [ ] Pentax K-3 III (SD)

## Card Types

### SD Cards
- [ ] SDHC (32GB)
- [ ] SDXC (128GB)
- [ ] SDXC (256GB)
- [ ] SDXC (512GB) — **HAVE** (NIKON Z 9)
- [ ] UHS-I
- [ ] UHS-II

### CFexpress
- [ ] CFexpress Type B
- [ ] CFexpress Type A

### XQD
- [ ] XQD (legacy Nikon)

### microSD
- [ ] microSD with adapter
- [ ] GoPro (microSD)
- [ ] DJI drone (microSD)

## Test Scenarios

### Basic
- [ ] Card detection on insert
- [ ] Brand identification correct
- [ ] Storage size accurate
- [ ] File count matches
- [ ] Eject works cleanly

### Content
- [ ] DCIM folder structure parsed
- [ ] Date groups accurate
- [ ] Photo count correct
- [ ] Video count correct
- [ ] Star ratings detected

### Edge Cases
- [ ] Empty card
- [ ] Nearly full card (>95%)
- [ ] Card with no DCIM folder
- [ ] Card with corrupted files
- [ ] Read-only card (lock switch)
- [ ] Multiple cards inserted

## Linux Testing

### Distros
- [ ] Ubuntu 22.04 LTS
- [ ] Ubuntu 24.04 LTS
- [ ] Fedora 40
- [ ] Debian 12

### Hardware
- [ ] Built-in SD slot (CID readable)
- [ ] USB 3.0 card reader
- [ ] USB-C card reader
- [ ] USB 2.0 card reader (slow)

## Notes

### Current Test Status
| Card | Camera | Size | Status | Notes |
|------|--------|------|--------|-------|
| SDXC | Nikon Z9 | 512GB | ✅ Working | Primary dev card, USB reader |

### Known Limitations
- CID info only available on Linux with direct SD slot
- USB readers hide hardware details
- macOS shows raw device size vs filesystem size

### Wishlist
- [ ] Test with actual CFexpress Type B card
- [ ] Test with dual-slot camera (mixed card types)
- [ ] Test with 1TB+ card
