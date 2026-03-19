# Installation and Build Guide

## Prebuilt binaries (recommended)

Download from GitHub Releases and run directly.

### Apple Silicon (arm64)

```bash
curl -fL -o cardbot https://github.com/willduncanphoto/CardBot/releases/latest/download/cardbot-darwin-arm64
chmod +x cardbot
sudo mv cardbot /usr/local/bin/cardbot
```

### Intel Mac (amd64)

```bash
curl -fL -o cardbot https://github.com/willduncanphoto/CardBot/releases/latest/download/cardbot-darwin-amd64
chmod +x cardbot
sudo mv cardbot /usr/local/bin/cardbot
```

### Install without sudo

```bash
mkdir -p ~/.local/bin
curl -fL -o ~/.local/bin/cardbot https://github.com/willduncanphoto/CardBot/releases/latest/download/cardbot-darwin-arm64
chmod +x ~/.local/bin/cardbot
# Ensure ~/.local/bin is in your PATH
```

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

## Test commands

```bash
go test ./... -count=1
make test
```

## Self-update

If installed from a release binary, you can update in place:

```bash
cardbot self-update
```

This downloads the latest release asset for your platform and verifies SHA256 checksums.
