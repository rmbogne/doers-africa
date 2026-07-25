#!/usr/bin/env python3
"""Add authenticated user names to the two existing dashboard headings."""

from pathlib import Path

REPLACEMENTS = {
    Path("templates/doer_dashboard.html"): (
        "Doer Dashboard",
        "Welcome, {{.UserName}}",
    ),
    Path("templates/customer_dashboard.html"): (
        "Customer Dashboard",
        "Welcome, {{.UserName}}",
    ),
}

for path, (old_text, new_text) in REPLACEMENTS.items():
    if not path.exists():
        raise SystemExit(f"Missing template: {path}")

    content = path.read_text(encoding="utf-8")

    if new_text in content:
        print(f"Already updated: {path}")
        continue

    occurrences = content.count(old_text)
    if occurrences != 1:
        raise SystemExit(
            f"Expected exactly one {old_text!r} in {path}; found {occurrences}"
        )

    path.write_text(
        content.replace(old_text, new_text, 1),
        encoding="utf-8",
    )
    print(f"Updated: {path}")
