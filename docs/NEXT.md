# CardBot — Next Steps

Written: 2026-03-12

## What was done tonight (0.1.7)

### Dependencies & housekeeping
- Updated 7 dependencies to latest (zerolog 1.34, msgp 1.6.3, go-colorable 0.1.14, fwd 1.2.0, go-isatty 0.0.20, golang.org/x/sys 0.42.0)
- go.mod bumped to go 1.25.0
- Fixed `log_test.go` Printf format string for stricter Go vet
- Removed duplicate `fmtBytes` from copy package → uses `detect.FormatBytes`
- Added `ui/color_test.go` (3 new tests), `TestTitleCase` (1 new test) → 97 → 100 tests

### Code fixes (from review notes)
- Improved `readExif` Make/Model dedup — handles "NIKON CORPORATION" + "NIKON Z 9" correctly
- `cleanGear` spacing fix — proper trim + join after brand prefix replacement
- Added `titleCase` helper for future use

### UX polish
- Startup animation: dots appear one at a time over ~1s
- Scanning spinner: classic `| / - \` with clean goroutine lifecycle
- Timestamps: ISO format with T separator everywhere (`2026-03-12T23:09:16`)
- Status: `Copy completed on 2026-03-12T12:31:05` (full timestamp with seconds)
- Trimmed "Scanning for memory cards" → "Scanning"
- Removed redundant "Scan completed" line → merged into file count line `3051 files ✓ (0s)`

### Platform
- Linux support pushed to 0.3.0 milestone
- README and ROADMAP updated — Linux is "Planned for 0.3.0"
- All Linux-specific items consolidated into new 0.3.0 milestone in ROADMAP

### Version strings synced
- 0.1.7 across README, OUTPUT, DOTFILE, CODE_REVIEW, main.go

---

## What to tackle next

### Quick fixes (anytime)
| Item | Source |
|------|--------|
| Add "OM System" to `BrandColor` in `ui/color.go` | GLM review |
| `go build -ldflags="-s -w"` for smaller binary | MiniMax review |

### 0.1.8 — Selective Copy
| Item | Notes |
|------|-------|
| `[s]` Copy Selects — starred/picked files only (XMP rating > 0) | Core feature |
| `[p]` Copy Photos — RAW + JPEG, no video | Core feature |
| `[v]` Copy Videos — MOV, MP4, MXF, etc. | Core feature |
| Dotfile tracks copy mode per operation (`"mode": "selects"`) | Design needed — see TODO.md |
| Status line reflects partial copy | e.g. `Selects copied on ...` |
| Re-copy guard per mode | Don't skip if previous copy was a different mode |

### 0.1.9 — Code Health
| Item | Source |
|------|--------|
| Log walk errors instead of silently swallowing | Kimi, Claude reviews |
| Destination path validation on config load | Kimi, GLM reviews |
| Split `main.go` (~950 lines) into `app.go`, `display.go`, `copy_cmd.go` | All 4 reviews |
| Extract `printCardHeader` helper (DRY) | All 4 reviews |
| `friendlyErr` for all user-facing errors | All 4 reviews |
| Add `context.Context` to `displayCard` and analyzer | Kimi, Claude reviews |
| Move `FormatBytes` to platform-agnostic file | MiniMax review |
| Test coverage pass — target 80%+ | Roadmap |

### 0.2.0 — Daily Driver
- All 0.1.x items complete
- Single-key input (raw terminal, no Enter)
- Tested on personal gear across multiple shooting days
- Feedback from at least one other photographer
- First public-facing release candidate

### 0.3.0 — Linux Support
- Detection, hardware info, speed test, eject
- Real-world testing (Ubuntu, Fedora, Debian)
- Stable build and CI

---

## Review notes location
- `docs/claude_todo.md` — Claude's review (bugs found, prioritized fixes)
- `docs/kimi_todo.md` — Kimi's review (architecture, concurrency, error handling)
- `docs/minimax_todo.md` — MiniMax's review (quick wins, potential issues)
- `docs/glm_todo.md` — GLM's review (cross-review comparison, unique findings)
- `docs/TODO.md` — Master TODO with wishlist and selective copy design notes
- `docs/ROADMAP.md` — Full milestone history and upcoming milestones
