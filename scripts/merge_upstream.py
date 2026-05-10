#!/usr/bin/env python3
"""Merge upstream disposable lists into data/domains.txt.

Reads existing entries, merges them with upstream sources, normalises
(lowercase + strip + IDN-to-ASCII via `idna`), removes anything in
exceptions.txt, sorts, dedupes, and writes back to the output path.

Used by .github/workflows/daily-update.yml.
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

try:
    import idna
except ImportError:
    print(
        "missing dependency: python -m pip install -r scripts/requirements.txt",
        file=sys.stderr,
    )
    sys.exit(2)


def read_lines(path: Path) -> set[str]:
    if not path.exists():
        return set()
    out: set[str] = set()
    with path.open() as fh:
        for raw in fh:
            entry = raw.strip()
            if not entry or entry.startswith("#"):
                continue
            out.add(normalize(entry))
    out.discard("")
    return out


def normalize(domain: str) -> str:
    domain = domain.strip().lower()
    if not domain:
        return ""
    try:
        ascii_domain = idna.encode(domain).decode("ascii")
    except idna.IDNAError:
        return ""
    if not is_valid_ascii_hostname(ascii_domain):
        return ""
    return ascii_domain


def is_valid_ascii_hostname(domain: str) -> bool:
    if not domain or len(domain) > 253:
        return False
    for label in domain.split("."):
        if not label or len(label) > 63:
            return False
        if label.startswith("-") or label.endswith("-"):
            return False
        if any(not (ch.isascii() and (ch.isalnum() or ch == "-")) for ch in label):
            return False
    return True


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", nargs="+", required=True, type=Path)
    parser.add_argument("--existing", required=True, type=Path)
    parser.add_argument("--exceptions", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()

    merged = read_lines(args.existing)
    for src in args.input:
        merged |= read_lines(src)
    merged -= read_lines(args.exceptions)
    merged.discard("")

    args.output.write_text("\n".join(sorted(merged)) + "\n")
    print(f"wrote {len(merged)} domains to {args.output}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
