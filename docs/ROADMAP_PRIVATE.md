# CardBot Roadmap (Private)

Moved from README.md — not public until closer to release.

## [OK] 0.1.0 – 0.1.5 — Detection, Analysis, Config, Copy & Hardening
- Native macOS card detection (DiskArbitration)
- DCIM walking, date grouping, file type breakdown
- EXIF camera model, XMP star ratings, parallel EXIF workers
- Hardware info (macOS via IOKit/system_profiler)
- Config file with first-run setup and native folder picker
- Brand name cleanup and ANSI colors
- Copy all files with dated folders, size verification, dotfile tracking
- File collision skip (same size = skip, safe re-copy)
- Bug fixes: race conditions, input drain, path escaping, log formatting
- Test suite: 81 tests across 6 packages

## [OK] 0.1.6 — Copy Robustness & UX
- Cancel during copy (`[\]` key), card removal mid-copy, Ctrl+C during copy
- Disk space preflight check
- Read-only card warnings
- Output mutex for concurrent progress/scan output
- Path traversal guard on copy destinations
- File handle leak fix, goroutine leak fix, named return for close errors
- Invalid card handling (no DCIM → friendly message, eject/exit only)
- Help command (`[?]`) with full key reference
- Unknown input feedback
- Friendly error messages (disk full, permission denied, I/O errors)
- Key remapping: `x` = exit, `\` = cancel copy, `s` = selects (stub)
- Test suite: 97 tests across 6 packages, all passing with `-race`

## [TODO] 0.1.7 — Polish
- Single-key input (no Enter required)
- Startup under 100ms, ETA during copy
- Copy Selects mode (starred files only)
- Show current filename during copy (deferred to renaming milestone)

## Later
- File renaming, resume interrupted copies, video metadata, auto-update, copyright/personal data injection on copy
