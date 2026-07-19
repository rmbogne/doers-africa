#!/usr/bin/env bash
set -euo pipefail

patterns=(
  'password=password'
  'mongodb://[^[:space:]"]+:[^[:space:]"]+@'
  'postgres(ql)?://[^[:space:]"]+:[^[:space:]"]+@'
  'CSRF_SECRET[[:space:]]*=[[:space:]]*"[^"]+"'
  'const[[:space:]]+postgresConnectionString'
)

failed=0

for pattern in "${patterns[@]}"; do
  matches="$(
    grep -RInE \
      --exclude-dir=.git \
      --exclude-dir=node_modules \
      --exclude-dir=vendor \
      --exclude='.env' \
      --exclude='.env.*' \
      --exclude='.env.example' \
      --exclude='check_no_hardcoded_secrets.sh' \
      "${pattern}" \
      . || true
  )"

  if [[ -n "${matches}" ]]; then
    echo "Potential hardcoded secret detected:"
    echo "${matches}"
    echo
    failed=1
  fi
done

if [[ "${failed}" -ne 0 ]]; then
  exit 1
fi

echo "No obvious hardcoded credentials or secrets found."
