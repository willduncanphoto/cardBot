#!/usr/bin/env sh
set -eu

LABEL="com.illwill.cardbot"
NO_SUDO=0
DRY_RUN=0
PURGE=0
EXTRA_INSTALL_DIR=""

usage() {
  cat <<'EOF'
CardBot uninstaller

Usage:
  sh uninstall.sh [options]

Options:
  --install-dir <path>  Additional install dir to remove <path>/cardbot
  --no-sudo             Do not attempt sudo for protected files
  --purge               Remove config and logs in addition to binary/launch agent
  --dry-run             Print actions without deleting anything
  -h, --help            Show help

Examples:
  sh uninstall.sh
  sh uninstall.sh --purge
  sh uninstall.sh --install-dir "$HOME/.local/bin" --no-sudo
EOF
}

say() {
  printf '%s\n' "$*"
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'Error: missing required command: %s\n' "$1" >&2
    exit 1
  }
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --install-dir)
      [ "$#" -ge 2 ] || { printf 'Error: --install-dir requires a value\n' >&2; exit 1; }
      EXTRA_INSTALL_DIR="$2"
      shift 2
      ;;
    --no-sudo)
      NO_SUDO=1
      shift
      ;;
    --purge)
      PURGE=1
      shift
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      printf 'Error: unknown option: %s (use --help)\n' "$1" >&2
      exit 1
      ;;
  esac
done

need_cmd uname
need_cmd id

run_maybe() {
  if [ "$DRY_RUN" -eq 1 ]; then
    say "[dry-run] $*"
    return 0
  fi
  "$@"
}

run_ignore() {
  if [ "$DRY_RUN" -eq 1 ]; then
    say "[dry-run] $*"
    return 0
  fi
  "$@" >/dev/null 2>&1 || true
}

remove_file() {
  p="$1"
  [ -n "$p" ] || return 0
  if [ ! -e "$p" ] && [ ! -L "$p" ]; then
    return 0
  fi

  if [ "$DRY_RUN" -eq 1 ]; then
    say "[dry-run] remove $p"
    return 0
  fi

  if rm -f "$p" >/dev/null 2>&1; then
    say "Removed: $p"
    return 0
  fi

  if [ "$NO_SUDO" -eq 0 ] && command -v sudo >/dev/null 2>&1; then
    if sudo rm -f "$p" >/dev/null 2>&1; then
      say "Removed (sudo): $p"
      return 0
    fi
  fi

  say "Warning: could not remove $p"
}

remove_dir_if_empty() {
  d="$1"
  [ -n "$d" ] || return 0
  if [ ! -d "$d" ]; then
    return 0
  fi
  if [ "$DRY_RUN" -eq 1 ]; then
    say "[dry-run] rmdir $d (if empty)"
    return 0
  fi
  rmdir "$d" >/dev/null 2>&1 || true
}

say "==> CardBot uninstaller"
OS_RAW="$(uname -s)"
CB_BIN="$(command -v cardbot 2>/dev/null || true)"

if [ -n "$CB_BIN" ]; then
  say "Detected binary in PATH: $CB_BIN"
else
  say "Binary not found in PATH; continuing with manual cleanup"
fi

# Try built-in daemon uninstall first (updates config start_at_login=false).
if [ -n "$CB_BIN" ]; then
  run_ignore "$CB_BIN" uninstall-daemon
fi

# Always stop background daemon process if running.
run_ignore pkill -f "cardbot --daemon"

if [ "$OS_RAW" = "Darwin" ]; then
  UID_NUM="$(id -u)"
  PLIST="$HOME/Library/LaunchAgents/${LABEL}.plist"

  # Best-effort launchctl cleanup even if binary is gone.
  run_ignore launchctl bootout "gui/${UID_NUM}/${LABEL}"
  run_ignore launchctl bootout "gui/${UID_NUM}" "$PLIST"

  if [ "$DRY_RUN" -eq 1 ]; then
    say "[dry-run] remove $PLIST"
  else
    rm -f "$PLIST" >/dev/null 2>&1 || true
  fi

  remove_dir_if_empty "$HOME/Library/LaunchAgents"
fi

# Remove detected + likely binary locations.
remove_file "$CB_BIN"
[ "$CB_BIN" = "/usr/local/bin/cardbot" ] || remove_file "/usr/local/bin/cardbot"
[ "$CB_BIN" = "$HOME/.local/bin/cardbot" ] || remove_file "$HOME/.local/bin/cardbot"
[ "$CB_BIN" = "/opt/homebrew/bin/cardbot" ] || remove_file "/opt/homebrew/bin/cardbot"

if [ -n "$EXTRA_INSTALL_DIR" ]; then
  remove_file "$EXTRA_INSTALL_DIR/cardbot"
fi

if [ "$PURGE" -eq 1 ]; then
  say "==> Purging CardBot config/log files"

  # Config (macOS / Linux)
  remove_file "$HOME/Library/Application Support/cardbot/config.json"
  remove_file "$HOME/.config/cardbot/config.json"

  # Logs
  remove_file "$HOME/.cardbot/cardbot.log"
  remove_file "$HOME/.cardbot/cardbot.log.old"

  # Attempt to remove empty app-specific dirs left behind.
  remove_dir_if_empty "$HOME/Library/Application Support/cardbot"
  remove_dir_if_empty "$HOME/.config/cardbot"
  remove_dir_if_empty "$HOME/.cardbot"
fi

say "==> Done"
if [ "$PURGE" -eq 1 ]; then
  say "Config/log files were purged."
else
  say "Tip: re-run with --purge to remove config/log files too."
fi
