# Installation and Build Guide

## Recommended: one-line installer

Latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/willduncanphoto/CardBot/main/scripts/install.sh | sh
```

Specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/willduncanphoto/CardBot/main/scripts/install.sh | sh -s -- --version <version>
# example: --version v0.5.2
```

Install to custom path without sudo:

```bash
curl -fsSL https://raw.githubusercontent.com/willduncanphoto/CardBot/main/scripts/install.sh | sh -s -- --install-dir "$HOME/.local/bin" --no-sudo
```

Installer options:

```bash
sh scripts/install.sh --help
```

Uninstall:

```bash
curl -fsSL https://raw.githubusercontent.com/willduncanphoto/CardBot/main/scripts/uninstall.sh | sh
```

Uninstall and purge config/log files:

```bash
curl -fsSL https://raw.githubusercontent.com/willduncanphoto/CardBot/main/scripts/uninstall.sh | sh -s -- --purge
```

Uninstaller options:

```bash
sh scripts/uninstall.sh --help
```

---

## Manual install from release assets

### macOS Apple Silicon (arm64)

```bash
curl -fL -o cardbot https://github.com/willduncanphoto/CardBot/releases/latest/download/cardbot-darwin-arm64
install -m 755 cardbot /usr/local/bin/cardbot
```

### macOS Intel (amd64)

```bash
curl -fL -o cardbot https://github.com/willduncanphoto/CardBot/releases/latest/download/cardbot-darwin-amd64
install -m 755 cardbot /usr/local/bin/cardbot
```

### Linux amd64

```bash
curl -fL -o cardbot https://github.com/willduncanphoto/CardBot/releases/latest/download/cardbot-linux-amd64
install -m 755 cardbot /usr/local/bin/cardbot
```

### Linux arm64

```bash
curl -fL -o cardbot https://github.com/willduncanphoto/CardBot/releases/latest/download/cardbot-linux-arm64
install -m 755 cardbot /usr/local/bin/cardbot
```

### User-only install (no sudo)

```bash
mkdir -p "$HOME/.local/bin"
curl -fL -o "$HOME/.local/bin/cardbot" https://github.com/willduncanphoto/CardBot/releases/latest/download/cardbot-<os>-<arch>
chmod +x "$HOME/.local/bin/cardbot"
```

Use one of: `darwin-arm64`, `darwin-amd64`, `linux-amd64`, `linux-arm64`.

---

## Build from source

Requirements:
- Go 1.25+
- Git

```bash
git clone https://github.com/willduncanphoto/CardBot.git
cd CardBot
go build -o cardbot .
./cardbot --version
```

### macOS with Xcode CLI tools (native detection path)

```bash
xcode-select --install
go build -o cardbot .
```

### macOS without Xcode (CGO disabled)

```bash
CGO_ENABLED=0 go build -o cardbot .
```

---

## Verify / test

```bash
go test ./... -count=1
make test
```

## Self-update

```bash
cardbot self-update
```

Self-update downloads the latest matching release asset and verifies SHA256 checksums.


Run daemon mode:

```bash
cardbot --daemon
```

Manage login auto-start (macOS):

```bash
cardbot install-daemon
cardbot uninstall-daemon
```

Check daemon status:

```bash
cardbot daemon-status
cardbot daemon-status --json
cardbot daemon-status --recent-launches 5
```

Toggle daemon debug logging:

```bash
cardbot daemon-debug status
cardbot daemon-debug on
cardbot daemon-debug off
```

After changing debug mode, restart a running daemon process so the new setting is applied.

### Expected debug output

When `daemon.debug` is `true`:

- Interactive startup (`cardbot`) prints: `Daemon debug: enabled`
- Daemon startup (`cardbot --daemon`) prints: `Daemon debug logging: enabled`
- Daemon launch flow logs verbose lines including: config summary, mount path, single-instance guard block reason, launcher branch selection, exact executed command arguments

### Typical debug workflow

1. `cardbot daemon-debug on`
2. Restart daemon (`cardbot --daemon`)
3. Insert card and reproduce the issue
4. Inspect terminal output and log file for `Debug:` lines
5. `cardbot daemon-debug off`


# Uninstalling CardBot

## Quick Teardown (instant)

```bash
# Kill background daemon
pkill -f "cardbot --daemon"

# Remove LaunchAgent
launchctl bootout gui/$(id -u)/com.illwill.cardbot
rm -f ~/Library/LaunchAgents/com.illwill.cardbot.plist

# Remove binary
rm ~/bin/cardbot

# Optional: purge config + logs
rm -rf ~/Library/Application\ Support/cardbot ~/.cardbot
```

---

## With Uninstall Script

```bash
# Full uninstall (daemon + binary)
sh scripts/uninstall.sh --install-dir ~/bin

# Full uninstall + purge config + logs
sh scripts/uninstall.sh --install-dir ~/bin --purge
```

### Uninstall Script Options

| Flag | Description |
|------|-------------|
| `--install-dir <path>` | Additional directory to remove `<path>/cardbot` |
| `--no-sudo` | Skip sudo attempts for protected files |
| `--purge` | Also remove config and log files |
| `--dry-run` | Print actions without deleting anything |
| `-h, --help` | Show help |

---

## What Gets Removed

| Item | Path |
|------|------|
| LaunchAgent plist | `~/Library/LaunchAgents/com.illwill.cardbot.plist` |
| Binary | `~/bin/cardbot` (or other install path) |
| Config | `~/Library/Application Support/cardbot/` |
| Logs | `~/.cardbot/` |

---

## After Uninstall

- CardBot will no longer start at login.
- No background daemon will be running.
- Config and logs are preserved unless `--purge` was used.
- To reinstall, see the installation section at the top of this file.

## Daemon behavior

- Launches **Terminal.app** via AppleScript on card insert.
- Terminal selection has been simplified in setup (no app-choice prompt).
- Single-instance guard prevents duplicate foreground launches.
- Duplicate-event cooldown suppresses rapid repeat mount events.

If launch fails:
- Apple Events/automation errors → grant **Automation** permission.
- Permission denied / operation not permitted → grant **Full Disk Access**.

## Configuration

Config file path is platform specific:

- macOS: `~/Library/Application Support/cardbot/config.json`
- Linux: `~/.config/cardbot/config.json`

Important daemon fields:

```json
"daemon": {
  "enabled": false,
  "start_at_login": false,
  "terminal_app": "Terminal",
  "launch_args": [],
  "debug": false
}
```

`terminal_app` is retained for backward compatibility but daemon launches currently use Terminal.app via AppleScript.
Set `daemon.debug` to `true` to enable verbose daemon/launcher debug logging.

### 0.7.0 scope (Homebrew support)

- Add Homebrew tap + `cardbot` formula
- Use release binary assets in formula install flow
- Automate formula version/SHA updates on tagged releases
- Document `brew` install/upgrade/uninstall workflow
- Improve `self-update` permission guidance for brew-managed installs


## CLI flags

| Flag | Description |
|------|-------------|
| `--dest <path>` | Override destination path for this run |
| `--dry-run` | Analyze only; do not copy |
| `--daemon` | Run headless background watcher |
| `--setup` | Re-run setup prompts |
| `--reset` | Clear saved config |
| `--version` | Print version |