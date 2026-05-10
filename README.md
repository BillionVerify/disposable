# disposable

Tiny Go library + CLI that answers one question:
**"Is this email's domain a disposable / one-time mailbox?"**

- ~160k known disposable domains, embedded into the binary (no network or disk I/O at runtime).
- O(1) exact-match lookup through an in-memory map; wildcard suffixes use a
  short linear scan. Safe for concurrent use.
- Daily updates merged from upstream community lists (see `.github/workflows/daily-update.yml`).
- MIT licensed.

Maintained by [BillionVerify](https://billionverify.com), where this same code powers the production `/v1/verify/disposable` endpoint.

---

## Install

```bash
go get github.com/billionverify/disposable
```

Or grab the CLI:

```bash
go install github.com/billionverify/disposable/cmd/disposable-check@latest
```

## Library usage

```go
import "github.com/billionverify/disposable"

if disposable.IsDomain("mailinator.com") {
    // refuse, throttle, or tag as low-trust
}

if disposable.IsEmail("user@mailinator.com") {
    // ...
}

fmt.Println("loaded:", disposable.Count(), "disposable domains")
```

Hot-swap the table at runtime (e.g. fed by your own internal blocklist):

```go
disposable.SetData(
    []string{"trashy.example", "also-bad.example"}, // exact-match
    []string{"*.tempmail.example"},                 // wildcard suffixes
    []string{"good.example"},                       // exceptions: never flag these
)
```

`ResetEmbedded()` reverts to the embedded list.

### Lookup semantics

- Input is **case-insensitive** (`MAILINATOR.com` ≡ `mailinator.com`).
- Whitespace is stripped.
- IDN is converted to ASCII via `golang.org/x/net/idna` before lookup.
- Resolution order: **exceptions → exact domain → wildcard suffix**.
- A wildcard entry `*.foo.example` matches both `foo.example` and any subdomain (`a.foo.example`, `a.b.foo.example`).

## CLI usage

```bash
$ disposable-check user@mailinator.com alice@example.com
disposable  	user@mailinator.com
clean       	alice@example.com

$ echo user@mailinator.com | disposable-check --quiet || echo "BAD"
BAD

$ disposable-check --count
159416
```

Exit codes:

| code | meaning |
| --- | --- |
| `0` | every input is clean |
| `1` | at least one input is disposable |
| `2` | usage / I/O error |

## Data model

```
data/
  domains.txt        # one disposable domain per line (≈ 159k entries)
  wildcards.txt      # one suffix per line, optional `*.` prefix
  exceptions.txt     # never-flag list (overrides domains.txt)
```

Lines starting with `#` are treated as comments. Each file is embedded into the binary at build time via `go:embed`.

## Updating the disposable list

Fast path — open a PR that edits the relevant file by hand:

```bash
echo new-bad-domain.example >> data/domains.txt
sort -u -o data/domains.txt data/domains.txt
go test ./...
```

Bulk path — the daily GitHub Actions workflow at `.github/workflows/daily-update.yml` pulls upstream community lists, normalizes (lowercase + IDN-to-ASCII + dedup), drops malformed entries, diffs against `data/domains.txt`, and opens a PR with the resulting changes. See `CONTRIBUTING.md` for the curation policy.

## Versioning

`v0.YYYY.MMDD` patch releases ship every time the domain table changes — pin a version in `go.mod` if you need reproducibility. The Go API is stable; only the embedded data churns.

See `RELEASE.md` for the pre-tag checklist.

## Acknowledgements

Initial seed list compiled from years of internal abuse fighting at BillionVerify, augmented by:

- [disposable-email-domains/disposable-email-domains](https://github.com/disposable-email-domains/disposable-email-domains)
- [FGRibreau/mailchecker](https://github.com/FGRibreau/mailchecker)

Both are MIT-licensed. See `THIRD_PARTY_NOTICES.md` for attribution details.
