#!/usr/bin/env bash

set -euo pipefail

readonly MIGRATIONS_PATH="db/migrations"

if [[ -z "${POSTGRES_DSN:-}" ]]; then
    echo "POSTGRES_DSN is required." >&2
    exit 1
fi

if ! command -v migrate >/dev/null 2>&1; then
    echo "The migrate CLI is not installed." >&2
    exit 1
fi

command="${1:-}"

case "$command" in
    up)
        migrate \
            -path "$MIGRATIONS_PATH" \
            -database "$POSTGRES_DSN" \
            up
        ;;

    down)
        migrate \
            -path "$MIGRATIONS_PATH" \
            -database "$POSTGRES_DSN" \
            down 1
        ;;

    version)
        migrate \
            -path "$MIGRATIONS_PATH" \
            -database "$POSTGRES_DSN" \
            version
        ;;

    *)
        echo "Usage: $0 {up|down|version}" >&2
        exit 1
        ;;
esac