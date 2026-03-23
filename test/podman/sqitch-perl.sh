#!/usr/bin/env bash
set -euo pipefail

IMAGE=${IMAGE:-docker.io/sqitch/sqitch:latest}
WORKDIR=${WORKDIR:-/work}

usage() {
  echo "usage: $0 run <sqitch-args...>"
}

case "${1:-}" in
  run)
    shift
    podman run --rm -it \
      -v "${PWD}:${WORKDIR}" \
      -w "${WORKDIR}" \
      "$IMAGE" sqitch "$@"
    ;;
  *)
    usage
    exit 1
    ;;
esac
