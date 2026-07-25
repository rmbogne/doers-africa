#!/usr/bin/env bash
set -euo pipefail

required_tokens=(
  "--bg-main:"
  "--bg-card:"
  "--text-main:"
  "--text-muted:"
  "--accent:"
  "--border-soft:"
)

stylesheet="static/css/style.css"

if [[ ! -f "${stylesheet}" ]]; then
  echo "Missing ${stylesheet}"
  exit 1
fi

for token in "${required_tokens[@]}"; do
  if ! grep -Fq -- "${token}" "${stylesheet}"; then
    echo "Shared theme token not found: ${token}"
    exit 1
  fi
done

legacy_pattern='rgba\(255,[[:space:]]*255,[[:space:]]*255,[[:space:]]*0\.(04|05|08|1|10|12|18)\)'

legacy_matches="$(
  grep -RInE \
    --include='*.html' \
    "${legacy_pattern}" \
    templates 2>/dev/null || true
)"

if [[ -n "${legacy_matches}" ]]; then
  echo "Legacy dark-theme literals remain:"
  echo "${legacy_matches}"
  exit 1
fi

echo "Shared light theme checks passed."
