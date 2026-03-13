# Auto-Update

CardBot update flow is designed for non-technical users while staying safe for a photo-ingest tool.

---

## Current Behavior

### Startup check (implemented)

- CardBot checks GitHub Releases for a newer version on startup.
- Check is cached to once per 24 hours (`config.update.last_check`).
- Network/API failures are silent (no noisy error output).
- Timeout is short (~2s max).
- If newer version exists, CardBot prints:
  - `Update available: X.Y.Z (you have A.B.C)`
  - `Run: cardbot self-update`

### Manual update command (implemented)

```bash
cardbot self-update
```

What it does:

1. Detects platform (`darwin-arm64`, `darwin-amd64`, etc.)
2. Reads latest release metadata from GitHub
3. Downloads matching binary + `checksums.txt`
4. Verifies SHA256 checksum
5. Atomically replaces current binary
6. Preserves executable permissions

If install path is not writable, CardBot prints a `sudo` command to retry.

---

## Safety Properties

- Explicit user action for install (`self-update`)
- Checksum verification before replace
- Atomic file replacement (no partial binary state)
- Graceful failure when offline or unauthorized

---

## CI/CD Dependency

Release workflow must publish:

- `cardbot-darwin-arm64`
- `cardbot-darwin-amd64`
- `cardbot-linux-amd64`
- `cardbot-linux-arm64`
- `checksums.txt` (SHA256)

---

## Open Questions / Future Investigation

### User-local install paths (avoid sudo)

Current install docs suggest `/usr/local/bin/cardbot`, which requires `sudo` for self-update.

**Idea:** Support / document installing to user-writable paths:
- `~/bin/cardbot` (add `~/bin` to PATH)
- `~/.local/bin/cardbot` (XDG standard, already in PATH on many Linux distros)
- `~/.cardbot/bin/cardbot` (self-contained)

Benefits:
- No sudo needed for self-update
- Works on shared machines without admin rights
- Simpler install story for non-technical users

Trade-offs:
- Need to ensure `~/bin` or `~/.local/bin` is in user's PATH
- Different path conventions across platforms
- May need shell profile modification (`.zshrc`, `.bashrc`)

**Decision needed:** Should we change default install path recommendation? Provide multiple options?

---

## Out of Scope (for now)

- Silent auto-apply updates on startup
- GUI updater
- Homebrew formula auto-bump
- Delta/binary patch updates
- Windows installer/update support
