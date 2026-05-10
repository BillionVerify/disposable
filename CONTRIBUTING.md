# Contributing

Thanks for considering a contribution. The goal of this repo is narrow: keep an
accurate list of disposable / one-time email domains, expose a tiny Go API around
it, and ship daily updates without breaking downstream consumers.

## Code style

- Standard Go formatting (`gofmt`); CI runs `go vet` and `go test ./...`.
- Public functions must have a doc comment — the package is small and every
  comment is read.
- No new runtime dependencies without discussion. The only runtime import today
  is `golang.org/x/net/idna`.
- The upstream merge script uses Python dependencies listed in
  `scripts/requirements.txt`.

## Adding a domain

1. Add the domain to `data/domains.txt` in lowercase ASCII (use the Punycode
   form for IDN — e.g. `xn--mller-kva.example`, not `möller.example`).
2. Re-sort: `sort -u -o data/domains.txt data/domains.txt`.
3. Run `go test ./...`.
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

## Daily auto-merge policy

`.github/workflows/daily-update.yml` runs every night and:

1. Fetches upstream community lists (disposable-email-domains, mailchecker).
2. Normalises (lowercase, IDN-to-ASCII, dedup) and drops malformed entries.
3. Subtracts entries already in `data/exceptions.txt`.
4. Diffs against `data/domains.txt`.
5. Opens a PR if there are net changes.

PRs from the daily job that **only add** domains may be auto-merged once CI is
green. PRs that **remove** domains require human review — removals are usually
a sign that an upstream list lost an entry, not that we should drop it.

## Releasing

1. Tag a patch version once changes accumulate: `git tag v0.YYYY.MMDD && git push --tags`.
2. The Go module proxy will pick it up within minutes.

There is no semver-style breaking-change tag — the API surface is intentionally
small and frozen. If you ever need to break it, fork.

See `RELEASE.md` for the full release checklist.
