#!/usr/bin/env python3
"""Refresh data/domains.txt from upstream community lists.

Driven by scripts/sources.json. For each source we:

1. Download the upstream LICENSE file (path declared in sources.json) and
   verify it still contains a fingerprint matching the declared
   `expected_license`. The declared license must also be in the global
   `permissive_licenses` allowlist. Any drift fails the run.
2. Download the raw data file from GitHub (text lines or JSON array).
3. Normalize entries to lowercase ASCII (IDN-to-Punycode via `idna`) and
   drop anything that is not a valid hostname.
4. Union all sources with the existing on-disk list, subtract `exceptions.txt`,
   sort, and write back to `--output`.

Used by .github/workflows/update.yml; safe to run locally for a dry run.
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import urllib.error
import urllib.request
from pathlib import Path
from typing import Iterable

try:
    import idna
except ImportError:
    print(
        "missing dependency: python -m pip install -r scripts/requirements.txt",
        file=sys.stderr,
    )
    sys.exit(2)


RAW_BASE = "https://raw.githubusercontent.com"
USER_AGENT = "billionverify-disposable-updater"

# Substrings unique enough to identify each SPDX license in the wild. Matching
# is case-insensitive; at least one fingerprint per `expected_license` must
# appear in the upstream's declared license file. Override per-source with
# `license_fingerprints` when the upstream stores its license outside the
# standard LICENSE file (package.json, README, etc).
DEFAULT_FINGERPRINTS: dict[str, list[str]] = {
    "MIT": [
        "MIT License",
        "Permission is hereby granted, free of charge",
    ],
    "BSD-2-Clause": [
        "BSD 2-Clause License",
    ],
    "BSD-3-Clause": [
        "BSD 3-Clause License",
        "Neither the name of",
    ],
    "0BSD": [
        "BSD Zero Clause License",
    ],
    "ISC": [
        "ISC License",
    ],
    "CC0-1.0": [
        "CC0 1.0 Universal",
        "Public Domain Dedication",
    ],
    "Apache-2.0": [
        "Apache License, Version 2.0",
    ],
    "Unlicense": [
        "This is free and unencumbered software released into the public domain",
    ],
}


def http_get(url: str) -> bytes:
    req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    token = os.environ.get("GITHUB_TOKEN")
    if token and url.startswith(RAW_BASE):
        # GITHUB_TOKEN works for raw.githubusercontent.com via basic auth.
        req.add_header("Authorization", f"Bearer {token}")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return resp.read()
    except urllib.error.HTTPError as e:
        raise SystemExit(f"GET {url} -> HTTP {e.code}") from e


def check_license(source: dict, allowlist: set[str]) -> str:
    expected = source["expected_license"]
    if expected not in allowlist:
        raise SystemExit(
            f"[{source['id']}] expected_license {expected!r} is not in the permissive allowlist {sorted(allowlist)}"
        )
    license_path = source["license_path"]
    url = f"{RAW_BASE}/{source['owner']}/{source['repo']}/{source['ref']}/{license_path}"
    body = http_get(url).decode("utf-8", errors="replace").lower()
    fingerprints = source.get("license_fingerprints") or DEFAULT_FINGERPRINTS.get(expected, [])
    if not fingerprints:
        raise SystemExit(
            f"[{source['id']}] no fingerprints configured for {expected!r}; add license_fingerprints in sources.json"
        )
    if not any(fp.lower() in body for fp in fingerprints):
        raise SystemExit(
            f"[{source['id']}] {license_path} no longer contains a {expected} fingerprint. "
            "Either the upstream re-licensed (audit and update sources.json) or the file moved."
        )
    return expected


def fetch_source(source: dict) -> Iterable[str]:
    url = f"{RAW_BASE}/{source['owner']}/{source['repo']}/{source['ref']}/{source['data_path']}"
    body = http_get(url)
    fmt = source.get("format", "text")
    if fmt == "json":
        data = json.loads(body)
        if not isinstance(data, list):
            raise SystemExit(f"[{source['id']}] expected JSON array at {url}")
        return (str(x) for x in data)
    if fmt == "text":
        return body.decode("utf-8", errors="replace").splitlines()
    raise SystemExit(f"[{source['id']}] unknown format {fmt!r}")


def read_lines(path: Path) -> set[str]:
    if not path.exists():
        return set()
    out: set[str] = set()
    with path.open() as fh:
        for raw in fh:
            entry = normalize(raw)
            if entry:
                out.add(entry)
    return out


def normalize(raw: str) -> str:
    entry = raw.strip().lower()
    if not entry or entry.startswith("#"):
        return ""
    if "#" in entry:
        entry = entry.split("#", 1)[0].strip()
        if not entry:
            return ""
    try:
        ascii_domain = idna.encode(entry).decode("ascii")
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
    parser.add_argument("--sources", required=True, type=Path)
    parser.add_argument("--existing", required=True, type=Path)
    parser.add_argument("--exceptions", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()

    config = json.loads(args.sources.read_text())
    allowlist = set(config.get("permissive_licenses", []))
    if not allowlist:
        raise SystemExit("sources.json: permissive_licenses must be non-empty")

    merged = read_lines(args.existing)
    starting = len(merged)
    print(f"existing: {starting} entries", file=sys.stderr)

    for source in config["sources"]:
        spdx = check_license(source, allowlist)
        added = 0
        for raw in fetch_source(source):
            entry = normalize(raw)
            if entry and entry not in merged:
                merged.add(entry)
                added += 1
        print(
            f"  {source['id']:30s} license={spdx:13s} added={added:+6d}  (total {len(merged)})",
            file=sys.stderr,
        )

    merged -= read_lines(args.exceptions)
    merged.discard("")

    args.output.write_text("\n".join(sorted(merged)) + "\n")
    print(
        f"wrote {len(merged)} domains to {args.output} (net {len(merged) - starting:+d})",
        file=sys.stderr,
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
