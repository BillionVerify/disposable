# Contributing

Thanks for considering a contribution. The goal of this repo is narrow: keep an
accurate list of disposable / one-time email domains, expose tiny Go and Rust
interfaces around it, and ship frequent updates without breaking downstream
consumers.

## Code style

- Standard Go formatting (`gofmt`); CI runs `go vet` and `go test ./...`.
- Standard Rust formatting (`cargo fmt`); CI runs Clippy, tests, and package
  verification.
- Public functions must have a doc comment — the package is small and every
  comment is read.
- No new runtime dependencies without discussion. IDN normalization uses
  `golang.org/x/net/idna` in Go and `idna` in Rust.
- The upstream merge script uses Python dependencies listed in
  `scripts/requirements.txt`.

## Adding a domain

1. Add the domain to `data/domains.txt` in lowercase ASCII (use the Punycode
   form for IDN — e.g. `xn--mller-kva.example`, not `möller.example`).
2. Re-sort: `sort -u -o data/domains.txt data/domains.txt`.
3. Run `go test ./...` and `cargo test --all-targets`.
4. Open a PR. Brief rationale in the description is enough — link to the
   provider's site if it's not obvious.

## Removing a domain (false positive)

If a domain was wrongly listed, add it to `data/exceptions.txt` rather than
deleting it from `data/domains.txt`. Two reasons:

1. Upstream syncs may re-add the entry; the exceptions file makes our override
   explicit and durable across re-imports.
2. Auditors can see the exact list of domains we've ever overridden.

## Adding a wildcard

Use `data/wildcards.txt` for entries like `*.tempmail.example`. The lookup
matches both the suffix itself and any subdomain. Wildcards are slower than
exact matches (linear scan), so prefer exact entries when you can enumerate
them.

## Hourly auto-merge policy

`.github/workflows/update.yml` runs every hour and:

1. For each entry in `scripts/sources.json`, downloads the upstream's
   `license_path` and verifies a fingerprint of the declared
   `expected_license` is still present. License drift fails the run.
2. Fetches each upstream's data file (text or JSON).
3. Normalises (lowercase, IDN-to-ASCII, dedup) and drops malformed entries.
4. Subtracts entries already in `data/exceptions.txt`.
5. If `data/domains.txt` changed, commits the new file directly to `main`
   with a message like `chore: hourly disposable list update (+42 → 197857)`.

There is no PR step. The merge is additive by design (false positives are
silenced via `exceptions.txt`, not by deleting upstream entries), and
running CI on every hour-of-the-day data churn is more noise than signal.
If you don't trust an upstream, remove it from `scripts/sources.json`.

## Adding a new upstream source

1. Append a block to `scripts/sources.json` with `owner`, `repo`, `ref`,
   `data_path`, `format` (`text` or `json`), `expected_license`, and the
   path to the upstream's license file (`license_path`). If the license is
   declared in a non-standard file (e.g. `package.json`), add an explicit
   `license_fingerprints` list — the script will fail unless one of those
   substrings appears in `license_path`.
2. Make sure `expected_license` is in the `permissive_licenses` allowlist
   at the top of `sources.json` (MIT, BSD-2/3-Clause, 0BSD, ISC, CC0-1.0,
   Apache-2.0, Unlicense).
3. Add the source to `THIRD_PARTY_NOTICES.md`.
4. Open a PR — the next hourly run will start pulling from it.

## Releasing

1. For Go, tag a patch version once changes accumulate: `git tag v0.YYYY.MMDD && git push --tags`.
2. The Go module proxy will pick it up within minutes.
3. Rust releases follow the separate Cargo checklist in `RELEASE.md`; updating
   the shared data does not implicitly publish a crate.

The Go interface remains intentionally small and frozen. The Rust crate follows
Cargo semantic versioning for any future interface changes.

See `RELEASE.md` for the full release checklist.
