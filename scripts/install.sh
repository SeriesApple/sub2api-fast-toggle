#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

command -v docker >/dev/null 2>&1 || { echo "docker was not found" >&2; exit 1; }
[ -f "$PROJECT_ROOT/.env" ] || cp "$PROJECT_ROOT/.env.example" "$PROJECT_ROOT/.env"
mkdir -p "$PROJECT_ROOT/runtime"
if [ ! -f "$PROJECT_ROOT/runtime/mode" ]; then
  printf 'on\n' > "$PROJECT_ROOT/runtime/mode"
fi
chmod 0755 "$PROJECT_ROOT/runtime"
chmod 0644 "$PROJECT_ROOT/runtime/mode"

docker compose --project-directory "$PROJECT_ROOT" -f "$PROJECT_ROOT/docker-compose.yml" config --quiet
docker compose --project-directory "$PROJECT_ROOT" -f "$PROJECT_ROOT/docker-compose.yml" up -d --build --wait
PROXY_ADDRESS=$(docker compose --project-directory "$PROJECT_ROOT" -f "$PROJECT_ROOT/docker-compose.yml" port fast-proxy 8080)

echo "Installation complete."
echo "Client API base URL: http://$PROXY_ADDRESS/v1"
echo "Sub2API management remains on its original port, normally http://localhost:8081"
