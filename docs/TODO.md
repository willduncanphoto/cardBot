# CardBot — TODO

## Current Version: 0.1.2 ✅

Detection, card analysis, EXIF, and star ratings complete.

---

## 0.1.3 — Config & Destination (Next Up)

- [ ] Config file: `~/.config/cardbot/config.json`
- [ ] First-run setup — prompt for destination path
- [ ] Store: default destination path
- [ ] CLI flags: `--dest`, `--version`
- [ ] `--dry-run` mode (show what would be copied)
- [ ] Logging: `~/.cardbot/cardbot.log` (5MB rotation)

---

## 0.1.4 — UI Polish

- [x] Merge brand + camera lines
- [x] Clean up repetitive brand names ("Nikon NIKON Z 9" → "Nikon Z 9")
- [x] Brand colors (ANSI): Nikon yellow, Canon red, Sony white, etc.
- [x] Performance: parallel EXIF workers (4 default, 3.7x faster)
- [x] Card status line (New / Copied via .cardbot dotfile)
- [ ] Handle "no DCIM" case with warning
- [ ] Handle read-only cards
- [ ] File collision logic (skip, rename, overwrite)
- [ ] Better error messages

---

## 0.2.0 — Copy Release

- [ ] `[a] All` copy mode
- [ ] Use destination from config (set in 0.1.3)
- [ ] Create dated folder: `CardBot_YYMMDD_001/`
- [ ] Copy files with `io.CopyBuffer` (256KB)
- [ ] Progress updates every 5 seconds
- [ ] Cancel copy in progress (with cleanup)
- [ ] Size verification after copy
- [ ] Write `.cardbot` dotfile to card
- [ ] Display "Status: Processed" on re-insert
- [ ] **Output mutex** — add `outputMu sync.Mutex` to `app` before copy progress lands;
      copy ticker + scan goroutine will interleave without it
- [ ] **Cancel in-flight scan on removal** — `displayCard` goroutine currently finishes
      and prints results even if the card was removed mid-scan; needs a context or
      cancellation channel threaded through `Analyze()`

---

## Platform Support

- [x] macOS with Xcode (CGO + DiskArbitration)
- [x] macOS without Xcode (polling fallback)
- [x] Linux (polling-based, zero CGO)
- [ ] Linux verification — 0.3.x series
- [ ] Windows — long-term goal, no version assigned

---

## Speed Test — Future Improvements

Current implementation is a synthetic 256MB sequential read/write benchmark.
Down the line, beef this up with real-world tests:

- [ ] Write test files sized like actual RAW photos (e.g. 50MB for Z9 NEF)
- [ ] Write test files sized like actual video clips (e.g. 500MB–2GB for N-RAW/ProRes)
- [ ] Multi-file burst test (simulate ingesting a full card worth of files)
- [ ] Report burst speed vs sustained speed separately
- [ ] Compare results against card's rated spec (pull from known card database or let user enter rated speed)
- [ ] Warn if measured speed is significantly below rated speed (possible card degradation)
- [ ] Bypass OS page cache for accurate read speeds (`F_NOCACHE` / `fcntl`) — current read
      test hits macOS page cache and reports inflated numbers

---

## Testing Notes

- [ ] **Destination path display** — verify `~` shorthand is shown correctly at startup
      across all setup flows: first run, `--setup`, folder picker (full path returned by
      osascript should be contracted to `~/...`), and manual text entry

- [ ] Split `main.go` — extract display/prompt/UI logic into separate package
- [ ] Drop `app.printf()` method — use explicit `fmt.Printf` + `a.logf` instead
- [ ] Add output mutex before copy progress lands
