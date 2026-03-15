#!/usr/bin/env bash
set -euo pipefail

# macOS: ~/Library/Application Support/cardbot/config.json
# Linux: ~/.config/cardbot/config.json
if [[ "$(uname)" == "Darwin" ]]; then
  CFG="$HOME/Library/Application Support/cardbot/config.json"
else
  CFG="${XDG_CONFIG_HOME:-$HOME/.config}/cardbot/config.json"
fi

if [[ ! -f "$CFG" ]]; then
  echo "Config not found: $CFG"
  exit 1
fi

python3 - "$CFG" <<'PY'
import json, pathlib, sys
p = pathlib.Path(sys.argv[1])
try:
    d = json.loads(p.read_text())
except Exception as e:
    print(f"Failed to parse {p}: {e}")
    sys.exit(1)

d.setdefault("update", {})["last_check"] = ""
p.write_text(json.dumps(d, indent=2) + "\n")
print(f"Cleared update cache: {p}")
print("update.last_check = ''")
PY
