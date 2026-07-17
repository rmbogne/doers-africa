#!/usr/bin/env bash
set -euo pipefail

matches="$(
    grep -RIn \
        --exclude-dir=.git \
        --exclude-dir=node_modules \
        --exclude='security_check.sh' \
        --include='*.html' \
        --include='*.js' \
        --include='*.ts' \
        --include='*.tsx' \
        --include='*.go' \
        -E '\.innerHTML[[:space:]]*=' \
        . || true
)"

if [[ -n "${matches}" ]]; then
    echo "Unsafe innerHTML assignment detected:"
    echo "${matches}"
    echo
    echo "Use textContent, createElement, append, or replaceChildren instead."
    exit 1
fi

echo "No unsafe innerHTML assignments found."
