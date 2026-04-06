#!/usr/bin/env bash
set -euo pipefail

NAME=${NAME:-gsqitch-mariadb}
IMAGE=${IMAGE:-docker.io/library/mariadb:11}
PORT=${PORT:-3307}
ROOT_PASSWORD=${ROOT_PASSWORD:-root}
DB=${DB:-test}
USER=${USER:-sqitch}
PASSWORD=${PASSWORD:-sqitch}
INIT_SQL=${INIT_SQL:-$(dirname "$0")/mariadb-init.sql}

usage() {
  echo "usage: $0 {start|stop|rm|logs}"
}

case "${1:-}" in
  start)
    podman run -d \
      --name "$NAME" \
      -e MARIADB_ROOT_PASSWORD="$ROOT_PASSWORD" \
      -e MARIADB_DATABASE="$DB" \
      -e MARIADB_USER="$USER" \
      -e MARIADB_PASSWORD="$PASSWORD" \
      -v "$INIT_SQL:/docker-entrypoint-initdb.d/00-sqitch-init.sql:ro" \
      -p "${PORT}:3306" \
      "$IMAGE"
    ;;
  stop)
    podman stop "$NAME"
    ;;
  rm)
    podman rm -f "$NAME"
    ;;
  logs)
    podman logs "$NAME"
    ;;
  *)
    usage
    exit 1
    ;;
esac
