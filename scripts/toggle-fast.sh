#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
MODE_FILE="$PROJECT_ROOT/runtime/mode"
COMMAND=${1:-toggle}

mkdir -p "$PROJECT_ROOT/runtime"
CURRENT=off
if [ -f "$MODE_FILE" ] && [ "$(tr -d '\r\n ' < "$MODE_FILE")" = "on" ]; then
  CURRENT=on
fi

case "$COMMAND" in
  on|off) TARGET=$COMMAND ;;
  status) TARGET=$CURRENT ;;
  toggle)
    if [ "$CURRENT" = "on" ]; then TARGET=off; else TARGET=on; fi
    ;;
  *) echo "Usage: $0 [on|off|status|toggle]" >&2; exit 2 ;;
esac

if [ "$COMMAND" != "status" ]; then
  TEMP_FILE="$PROJECT_ROOT/runtime/mode.$$.tmp"
  trap 'rm -f "$TEMP_FILE"' EXIT INT TERM
  printf '%s\n' "$TARGET" > "$TEMP_FILE"
  chmod 0644 "$TEMP_FILE"
  mv -f "$TEMP_FILE" "$MODE_FILE"
  trap - EXIT INT TERM
fi

echo "Sub2API Fast mode: $(printf '%s' "$TARGET" | tr '[:lower:]' '[:upper:]')"
