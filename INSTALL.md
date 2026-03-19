# Installation and Build Guide

## Recommended: one-line installer

Latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/willduncanphoto/CardBot/main/install.sh | sh
```

Specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/willduncanphoto/CardBot/main/install.sh | sh -s -- --version <version>
# example: --version v0.5.2
```

Install to custom path without sudo:

```bash
curl -fsSL https://raw.githubusercontent.com/willduncanphoto/CardBot/main/install.sh | sh -s -- --install-dir "$HOME/.local/bin" --no-sudo
```

Installer options:

```bash
sh install.sh --help
```

Uninstall:

```bash
curl -fsSL https://raw.githubusercontent.com/willduncanphoto/CardBot/main/uninstall.sh | sh
```

Uninstall and purge config/log files:

```bash
curl -fsSL https://raw.githubusercontent.com/willduncanphoto/CardBot/main/uninstall.sh | sh -s -- --purge
```

Uninstaller options:

```bash
sh uninstall.sh --help
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
