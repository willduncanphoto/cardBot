#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "[qa-050] running go test"
go test ./... -count=1

echo "[qa-050] building cardbot"
go build -o cardbot .

echo "[qa-050] checking daemon-status text output"
./cardbot daemon-status >/tmp/cardbot-daemon-status.txt
for key in "cardBot Daemon Status" "Single-instance guard" "LaunchAgent"; do
  if ! grep -q "$key" /tmp/cardbot-daemon-status.txt; then
    echo "missing expected daemon-status key: $key" >&2
    exit 1
  fi
done

echo "[qa-050] checking daemon-status --json output"
./cardbot daemon-status --json >/tmp/cardbot-daemon-status.json
for key in '"version"' '"daemon"' '"single_instance_guard"' '"launch_agent"'; do
  if ! grep -q "$key" /tmp/cardbot-daemon-status.json; then
    echo "missing expected daemon-status JSON key: $key" >&2
    exit 1
  fi
done

echo "[qa-050] done"
