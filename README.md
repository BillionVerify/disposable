# Free and Open Source Disposable Email Detection

Tiny Go and Rust libraries, plus a Go CLI, that answer one question:
**"Is this email's domain a disposable / one-time mailbox?"**

- **~213k** known disposable domains, merged from nine community lists and embedded into the binary (no network or disk I/O at runtime).
- O(1) exact-match lookup through an in-memory map; wildcard suffixes use a
  short linear scan. Safe for concurrent use.
- **Hourly** updates from upstream lists, with per-source license verification
  on every run (see `.github/workflows/update.yml`).
- MIT licensed.

Maintained by [BillionVerify](https://billionverify.com), where the same disposable-domain data powers the production `/v1/verify/disposable` endpoint.

---

## Install

### Go

```bash
go get github.com/billionverify/disposable
```

Or grab the CLI:

```bash
go install github.com/billionverify/disposable/cmd/disposable-check@latest
```

### Rust

The native `disposable` crate lives in this repository. Until its first crates.io release, install it from GitHub:

```toml
[dependencies]
disposable = { git = "https://github.com/BillionVerify/disposable.git" }
```

## Library usage

### Go

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

### Rust

```rust
if disposable::is_domain("mailinator.com") {
    // refuse, throttle, or tag as low-trust
}

if disposable::is_email("user@mailinator.com") {
    // ...
}

println!("loaded: {} disposable domains", disposable::count());
```

Build an immutable detector from your own newline-delimited lists:

```rust
use disposable::DisposableDomains;

let domains = DisposableDomains::from_lists(
    "trashy.example\nalso-bad.example",
    "*.tempmail.example",
    "good.example",
);

assert!(domains.is_domain("also-bad.example"));
```

### Lookup semantics

- Input is **case-insensitive** (`MAILINATOR.com` ≡ `mailinator.com`).
- Whitespace is stripped.
- IDN is converted to ASCII via `golang.org/x/net/idna` in Go and `idna` in Rust before lookup.
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
212947
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
  domains.txt        # one disposable domain per line (≈ 213k entries)
  wildcards.txt      # one suffix per line, optional `*.` prefix
  exceptions.txt     # never-flag list (overrides domains.txt)
```

Lines starting with `#` are treated as comments. Each file is embedded into the binary at build time via Go's `go:embed` and Rust's `include_str!`.

## Updating the disposable list

Fast path — open a PR that edits the relevant file by hand:

```bash
echo new-bad-domain.example >> data/domains.txt
sort -u -o data/domains.txt data/domains.txt
go test ./...
cargo test --all-targets
```

Bulk path — the hourly GitHub Actions workflow at
`.github/workflows/update.yml` verifies each upstream's license, pulls its
data, normalizes (lowercase + IDN-to-ASCII + dedup), drops malformed
entries, and commits the result straight to `main` if anything changed.
See `CONTRIBUTING.md` for the curation policy and `scripts/sources.json`
for the full source list.

## Versioning

Go `v0.YYYY.MMDD` patch releases ship when the domain table changes — pin a version in `go.mod` if you need reproducibility. The Go API is stable; only the embedded data churns.

The Rust crate follows Cargo semantic versioning and starts at `0.1.0`. Its first crates.io publication remains a separate release step; this repository does not claim the package is published until that succeeds.

See `RELEASE.md` for the pre-tag checklist.

## BillionVerify API — free disposable check

- **Free:** `/v1/verify/disposable` does not consume credits.
- **Documentation:** [Disposable Email Check API reference](https://billionverify.com/docs/api-reference#disposable-email-check).

If you want to skip embedding the list, call the hosted BillionVerify endpoint directly. It performs only the disposable-domain lookup; it does not call SMTP, MX, or the verification cache.

Disposable email check — `POST https://api.billionverify.com/v1/verify/disposable`:

```bash
curl -X POST https://api.billionverify.com/v1/verify/disposable \
  -H "BV-API-KEY: sk_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{"email": "user@mailinator.com"}'
```

The endpoint also supports `GET /v1/verify/disposable?email=user%40mailinator.com`. Invalid email syntax returns HTTP 400 rather than reporting `is_disposable: false`, and `check_smtp: true` is rejected because this endpoint is lookup-only.

Response:

```json
{
  "success": true,
  "code": "0",
  "message": "Success",
  "data": {
    "email": "user@mailinator.com",
    "domain": "mailinator.com",
    "is_disposable": true,
    "checked_at": "2026-08-15T08:30:00Z"
  }
}
```

Equivalent from Go:

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
)

type disposableResponse struct {
    Data struct {
        Email        string `json:"email"`
        Domain       string `json:"domain"`
        IsDisposable bool   `json:"is_disposable"`
        CheckedAt    string `json:"checked_at"`
    } `json:"data"`
}

func main() {
    body, _ := json.Marshal(map[string]string{"email": "user@mailinator.com"})
    req, _ := http.NewRequest("POST",
        "https://api.billionverify.com/v1/verify/disposable",
        bytes.NewReader(body))
    req.Header.Set("BV-API-KEY", os.Getenv("BV_API_KEY"))
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    var result disposableResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        panic(err)
    }
    fmt.Println(result.Data.IsDisposable)
}
```

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
