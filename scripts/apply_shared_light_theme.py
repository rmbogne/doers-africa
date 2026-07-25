#!/usr/bin/env python3
"""Replace dark-theme-only literals in templates with shared light variables.

The script is deliberately narrow. It changes only exact legacy color tokens
used by the existing dashboard, marketplace, notification, and detail-page
styles. It does not alter HTML structure, Go template expressions, form
actions, CSRF fields, or JavaScript behavior.
"""

from pathlib import Path
import sys

TEMPLATE_DIRECTORY = Path("templates")

REPLACEMENTS = {
    "rgba(255,255,255,0.04)": "var(--surface-subtle)",
    "rgba(255,255,255,0.05)": "var(--surface-soft)",
    "rgba(255,255,255,0.08)": "var(--surface-accent)",
    "rgba(255,255,255,0.1)": "var(--border-soft)",
    "rgba(255,255,255,0.10)": "var(--border-soft)",
    "rgba(255,255,255,0.12)": "var(--border-soft)",
    "rgba(255,255,255,0.18)": "var(--border-strong)",
    "rgba(255, 255, 255, 0.04)": "var(--surface-subtle)",
    "rgba(255, 255, 255, 0.05)": "var(--surface-soft)",
    "rgba(255, 255, 255, 0.08)": "var(--surface-accent)",
    "rgba(255, 255, 255, 0.1)": "var(--border-soft)",
    "rgba(255, 255, 255, 0.10)": "var(--border-soft)",
    "rgba(255, 255, 255, 0.12)": "var(--border-soft)",
    "rgba(255, 255, 255, 0.18)": "var(--border-strong)",
}

if not TEMPLATE_DIRECTORY.is_dir():
    raise SystemExit(
        "Run this script from the repository root; templates/ was not found."
    )

changed_files = []
replacement_count = 0

for path in sorted(TEMPLATE_DIRECTORY.glob("*.html")):
    original = path.read_text(encoding="utf-8")
    updated = original
    file_replacements = 0

    for old_value, new_value in REPLACEMENTS.items():
        occurrences = updated.count(old_value)
        if occurrences:
            updated = updated.replace(old_value, new_value)
            file_replacements += occurrences

    if updated != original:
        path.write_text(updated, encoding="utf-8")
        changed_files.append(path)
        replacement_count += file_replacements
        print(f"Updated {path}: {file_replacements} replacement(s)")

if not changed_files:
    print("No legacy dark-theme literals required normalization.")
else:
    print(
        f"Completed: {replacement_count} replacement(s) "
        f"across {len(changed_files)} template(s)."
    )
