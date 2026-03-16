# CardBot Roadmap

## 0.3.5 (Current)
- ✅ Spinner animation for idle scanning
- ✅ Clean output formatting
- ✅ Cross-platform builds (darwin arm64/amd64, linux amd64/arm64)
- ✅ Architecture refactor: move root logic to `internal/app/`
- ✅ Update check on every startup with "Up to date" confirmation

## 0.4.0 (Footer Version)
Focus: UI/UX polish for Destination and Naming configuration

### Display Improvements
- [ ] Cleaner card info display when copy completes
- [ ] Better visual hierarchy for config info (Destination, Naming, Mode)
- [ ] Reduce visual clutter in the main prompt area
- [ ] Consider grouping related config items

### Destination Display
- [ ] Show abbreviated/contracted path consistently (e.g., `~/Pictures/Archive` not full path)
- [ ] Improve readability of destination status in copy output
- [ ] Clear indication of where files are being copied to

### Naming Display
- [ ] Simplify "Timestamp + sequence (xxxx = 0001-9999)" explanation
- [ ] Better preview of how files will be renamed
- [ ] Clearer distinction between "original" vs "timestamp" modes in output

## 0.4.1 (Housecleaning)
- [ ] Code style consistency pass
- [ ] Remove any dead code or unused helpers
- [ ] Standardize error message formatting
- [ ] Improve test coverage for edge cases
- [ ] Documentation cleanup

## 0.4.2 (Further Refinement)
- [ ] Performance profiling of large card scans
- [ ] Optimize progress update frequency during copy
- [ ] Consider caching strategies for repeated cards
- [ ] Terminal resize handling improvements
- [ ] Enhanced logging output for debugging

## Future (0.5+)
- [ ] Batch operations for multiple cards
- [ ] Configuration presets/profiles
- [ ] Integration with external tools (EXIF, color management)
- [ ] Faster incremental scanning for known cards
