#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

TS="$(date +%Y%m%d-%H%M%S)"
OUT_DIR="${1:-/tmp/cardbot-qa-051-permissions-$TS}"
mkdir -p "$OUT_DIR"
LOG_FILE="$OUT_DIR/daemon.log"

echo "[qa-051-perms] output dir: $OUT_DIR"
echo "[qa-051-perms] building cardbot"
go build -o cardbot .

echo "[qa-051-perms] starting daemon"
./cardbot --daemon >"$LOG_FILE" 2>&1 &
DAEMON_PID=$!

TAIL_PID=""
cleanup() {
  set +e
  if [[ -n "$TAIL_PID" ]]; then
    kill "$TAIL_PID" >/dev/null 2>&1 || true
  fi
  if kill -0 "$DAEMON_PID" >/dev/null 2>&1; then
    kill -INT "$DAEMON_PID" >/dev/null 2>&1 || true
    wait "$DAEMON_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT INT TERM

sleep 1

echo "[qa-051-perms] daemon pid: $DAEMON_PID"
echo "[qa-051-perms] log file: $LOG_FILE"
echo
echo "Manual permission validation steps:"
echo "  1) Revoke Automation for terminal app in System Settings"
echo "  2) Insert card and confirm daemon prints Hint: ... Automation ..."
echo "  3) Restore Automation, then reproduce permission-denied/FDA path"
echo "  4) Confirm daemon prints Hint: ... Full Disk Access ..."
echo
echo "Live daemon log (Ctrl+C to stop):"

(tail -f "$LOG_FILE") &
TAIL_PID=$!
wait "$TAIL_PID" || true

echo
echo "[qa-051-perms] summary"
AUTO_HINTS=$(grep -ci "hint: .*automation" "$LOG_FILE" || true)
FDA_HINTS=$(grep -ci "hint: .*full disk access" "$LOG_FILE" || true)
LAUNCH_FAILS=$(grep -ci "launch failed:" "$LOG_FILE" || true)

echo "  launch failures: $LAUNCH_FAILS"
echo "  automation hints: $AUTO_HINTS"
echo "  full disk access hints: $FDA_HINTS"
echo
echo "[qa-051-perms] done"
