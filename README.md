# CardBot

A CLI tool for camera memory cards

![CardBot screenshot](screenshot.png) 

## What CardBot does

- Detect camera memory cards on macOS
- Generate overview of content (file count, type, dates, equiptment data) 
- Creates folder structures in copy destination
- Copy modes: all, selects (starred), photos only, videos only, etc
- Tracks card copy status

## Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| macOS | Supported | Tested |
| Linux | I'm told it works | Untested |
| Windows | Ugh | Someday, Maybe Not |

## Installation

The easy install:

```bash
curl -fsSL https://raw.githubusercontent.com/willduncanphoto/CardBot/main/scripts/install.sh | sh
```

## Usage

Start CardBot:

```bash
cardbot
```

Quit CardBot:

`Ctrl+C`

CardBot will automatically run the setup if no config file is present.

Update to the latest version:

```bash
cardbot self-update
```

To run the setup again:

```bash
cardbot --setup
```

## Commands

| Key | Action |
|-----|--------|
| `a` | Copy all |
| `s` | Copy selects (starred/picked) |
| `p` | Copy photos only |
| `v` | Copy videos only |
| `e` | Eject card |
| `x` | Exit current card |
| `i` | Show card hardware info |
| `t` | Run speed test |
| `\` | Cancel active copy |
| `?` | Help |

## Roadmap

| Version | Focus | Status |
|---------|-------|--------|
| **0.8.0** | Card copy operations | Planned |
| **0.9.0** | Stuff | Planned |
| **0.10.0** | Copyright check and injection | Planned |
| **0.11.0** | Startup and Card Detection | Planned |
| **0.12.0** | Startup and Card Detection | Planned |



## Uninstalling

```bash
# Full uninstall (daemon + binary)
sh scripts/uninstall.sh --install-dir ~/bin

# Full uninstall + purge config + logs
sh scripts/uninstall.sh --install-dir ~/bin --purge
```

## License

MIT — see [LICENSE](LICENSE).
