#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

docker compose --project-directory "$PROJECT_ROOT" -f "$PROJECT_ROOT/docker-compose.yml" down --remove-orphans
echo "Sub2API Fast Toggle was removed. Configuration files were preserved."
