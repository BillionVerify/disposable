# Free and Open Source Disposable Email Detection

Tiny Go library + CLI that answers one question:
**"Is this email's domain a disposable / one-time mailbox?"**

- **~198k** known disposable domains, merged from nine community lists and embedded into the binary (no network or disk I/O at runtime).
- O(1) exact-match lookup through an in-memory map; wildcard suffixes use a
  short linear scan. Safe for concurrent use.
- **Hourly** updates from upstream lists, with per-source license verification
  on every run (see `.github/workflows/update.yml`).
- MIT licensed.

Maintained by [BillionVerify](https://billionverify.com), where this same code powers the production `/v1/verify/single` endpoint.

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
197814
```

Exit codes:

| code | meaning |
| --- | --- |
| `0` | every input is clean |
| `1` | at least one input is disposable |
| `2` | usage / I/O error |

## BillionVerify API — free disposable check

If you want to skip embedding the list (e.g. you're in a stack that isn't Go,
or you also need MX/SMTP/role/role/catch-all signals), call the BillionVerify
API directly. **Disposable detection is free** — the `is_disposable` field is
populated on every verification response and is not metered separately.

Single-email verification — `POST https://api.billionverify.com/v1/verify/single`:

```bash
curl -X POST https://api.billionverify.com/v1/verify/single \
  -H "BV-API-KEY: sk_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{"email": "user@mailinator.com"}'
```

Excerpted response:

```json
{
  "email": "user@mailinator.com",
  "is_disposable": true,
  "is_role": false,
  "is_catchall": false,
  "status": "invalid"
}
```

Equivalent from Go, reusing this library's normalization:

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "os"
)

func main() {
    body, _ := json.Marshal(map[string]string{"email": "user@mailinator.com"})
    req, _ := http.NewRequest("POST",
        "https://api.billionverify.com/v1/verify/single",
        bytes.NewReader(body))
    req.Header.Set("BV-API-KEY", os.Getenv("BV_API_KEY"))
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
    json.NewDecoder(resp.Body).Decode(&struct{}{}) // parse into your own struct
}
```

Full API reference (bulk verification, file uploads, webhooks, filtering by
`disposable=true`, credits, etc.): <https://billionverify.com/docs>.

## Data model

```
data/
  domains.txt        # one disposable domain per line (≈ 198k entries)
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

Bulk path — the hourly GitHub Actions workflow at
`.github/workflows/update.yml` verifies each upstream's license, pulls its
data, normalizes (lowercase + IDN-to-ASCII + dedup), drops malformed
entries, and commits the result straight to `main` if anything changed.
See `CONTRIBUTING.md` for the curation policy and `scripts/sources.json`
for the full source list.

## Versioning

`v0.YYYY.MMDD` patch releases ship every time the domain table changes — pin a version in `go.mod` if you need reproducibility. The Go API is stable; only the embedded data churns.

See `RELEASE.md` for the pre-tag checklist.

## Acknowledgements

Initial seed list compiled from years of internal abuse fighting at BillionVerify, augmented hourly by these MIT / BSD / CC0-licensed community projects:

- [disposable-email-domains/disposable-email-domains](https://github.com/disposable-email-domains/disposable-email-domains) (CC0-1.0)
- [FGRibreau/mailchecker](https://github.com/FGRibreau/mailchecker) (MIT)
- [7c/fakefilter](https://github.com/7c/fakefilter) (BSD-3-Clause)
- [amieiro/disposable-email-domains](https://github.com/amieiro/disposable-email-domains) (MIT)
- [ivolo/disposable-email-domains](https://github.com/ivolo/disposable-email-domains) (MIT)
- [wesbos/burner-email-providers](https://github.com/wesbos/burner-email-providers) (MIT)
- [martenson/disposable-email-domains](https://github.com/martenson/disposable-email-domains) (CC0-1.0)
- [groundcat/disposable-email-domain-list](https://github.com/groundcat/disposable-email-domain-list) (MIT)
- [unkn0w/disposable-email-domain-list](https://github.com/unkn0w/disposable-email-domain-list) (MIT)

See `THIRD_PARTY_NOTICES.md` for attribution details.
