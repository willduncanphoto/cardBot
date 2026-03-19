# CardBot

CardBot is a CLI tool for ingesting camera memory cards on macOS.

![CardBot screenshot](screenshot.png)

## What it does

- Detects inserted camera cards
- Analyzes card contents (counts, size, dates, camera metadata)
- Copies files into date-grouped folders
- Supports copy modes: all, selects (starred), photos, videos
- Supports background daemon mode (`--daemon`)
- Supports macOS login auto-start (`install-daemon` / `uninstall-daemon`)
- Supports direct path targeting (`cardbot /Volumes/<CARD>`)

## Platform status

| Platform | Status | Notes |
|----------|--------|-------|
| macOS | ✅ Supported | Primary platform |
| Linux | ⚠️ Experimental | Works in limited testing |
| Windows | ❌ Not supported | Not planned for now |

## Installation

For complete install/build instructions, see **[INSTALL.md](INSTALL.md)**.

Quick install (Apple Silicon):

```bash
curl -fL -o cardbot https://github.com/willduncanphoto/CardBot/releases/latest/download/cardbot-darwin-arm64
chmod +x cardbot
sudo mv cardbot /usr/local/bin/cardbot
```

Quick install (Intel Mac):

```bash
curl -fL -o cardbot https://github.com/willduncanphoto/CardBot/releases/latest/download/cardbot-darwin-amd64
chmod +x cardbot
sudo mv cardbot /usr/local/bin/cardbot
```

## Usage

Start interactive mode:

```bash
cardbot
```

Run setup (destination, naming mode, daemon prefs):

```bash
cardbot --setup
```

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
```

## CLI flags

| Flag | Description |
|------|-------------|
| `--dest <path>` | Override destination path for this run |
| `--dry-run` | Analyze only; do not copy |
| `--daemon` | Run headless background watcher |
| `--setup` | Re-run setup prompts |
| `--reset` | Clear saved config |
| `--version` | Print version |

## Interactive commands

| Key | Action |
|-----|--------|
| `a` | Copy all |
| `s` | Copy selects (starred/picked) |
| `p` | Copy photos |
| `v` | Copy videos |
| `e` | Eject card |
| `x` | Exit current card |
| `\` | Cancel active copy |
| `?` | Help |

## Daemon behavior

- Launches your configured terminal app on card insert.
- Terminal app choices in setup: **Default (macOS default terminal app)**, Terminal, Ghostty, or custom.
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
  "launch_args": []
}
```

`terminal_app` can be `Default`, `Terminal`, `Ghostty`, or a custom app name.

## Roadmap

| Version | Focus | Status |
|---------|-------|--------|
| **0.5.1** | QA fixes + daemon polish | Current |
| **0.6.0** | Copy UX improvements | Next |
| **0.8.0** | Copyright metadata injection | Planned |

## License

MIT — see [LICENSE](LICENSE).
