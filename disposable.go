// Package disposable answers a single question — "is this email's domain a
// disposable / one-time mailbox?" — using an embedded list of ~160k known
// disposable domains.
//
// Use it to short-circuit registration flows, throw-away accounts in growth
// loops, abuse pipelines, or paid email-verification stacks where running an
// SMTP probe against a one-time domain would be wasted work.
//
//	import "github.com/billionverify/disposable"
//
//	if disposable.IsDomain("mailinator.com") {
//	    // refuse, throttle, or tag as low-trust
//	}
//
// The package is safe for concurrent use. The domain table is loaded once at
// package init time from an embedded data/domains.txt file (no network or
// disk I/O at runtime).
//
// Maintained by BillionVerify (https://billionverify.com). Domain table updates
// land daily via the GitHub Actions workflow under .github/workflows; see
// CONTRIBUTING.md for how to add or remove entries.
package disposable

import (
	"strings"
	"sync"

	"golang.org/x/net/idna"
)

// State lives behind a RWMutex so SetData can hot-swap the table without
// racing readers. The embedded list is populated by package init() so the
// first IsDomain call has zero load cost. Post-init runs are O(1)
// exact/exception map lookups plus an O(N_wildcards) suffix walk.
var (
	stateMu        sync.RWMutex
	domainSet      map[string]struct{}
	wildcardSuffix []string
	exceptionSet   map[string]struct{}
)

func init() { ResetEmbedded() }

// SetData replaces the in-memory disposable table. Useful for tests or for
// hot-loading a fresher list at runtime. Pass nil/empty slices to clear.
//
// Resolution order at lookup time: exceptions → exact domain → wildcard suffix.
func SetData(domains []string, wildcards []string, exceptions []string) {
	ds := make(map[string]struct{}, len(domains))
	for _, d := range domains {
		if d = normalize(d); d != "" {
			ds[d] = struct{}{}
		}
	}
	es := make(map[string]struct{}, len(exceptions))
	for _, d := range exceptions {
		if d = normalize(d); d != "" {
			es[d] = struct{}{}
		}
	}
	ws := make([]string, 0, len(wildcards))
	for _, w := range wildcards {
		w = strings.TrimSpace(w)
		if strings.HasPrefix(w, "*.") {
			w = w[2:]
		}
		if w = normalize(w); w != "" {
			ws = append(ws, w)
		}
	}
	stateMu.Lock()
	domainSet = ds
	wildcardSuffix = ws
	exceptionSet = es
	stateMu.Unlock()
}

// ResetEmbedded reloads the in-memory tables from the embedded data files.
// Tests use it to undo SetData overrides; production callers normally do not
// need to call it because init() already loaded the embedded data.
func ResetEmbedded() {
	loadEmbedded()
}

// IsDomain reports whether `domain` is a known disposable domain.
//
// Input is case-insensitive and IDN-aware ("xn--..." Punycode is matched
// directly; international labels are converted to ASCII before lookup).
func IsDomain(domain string) bool {
	d := normalize(domain)
	if d == "" {
		return false
	}
	stateMu.RLock()
	defer stateMu.RUnlock()
	if _, ok := exceptionSet[d]; ok {
		return false
	}
	if _, ok := domainSet[d]; ok {
		return true
	}
	for _, suffix := range wildcardSuffix {
		if d == suffix || strings.HasSuffix(d, "."+suffix) {
			return true
		}
	}
	return false
}

// IsEmail extracts the domain part of `email` and runs IsDomain.
// A malformed input (no `@`, empty local part, empty domain) returns false.
//
// IsEmail is convenience sugar — callers that already parsed the address
// should prefer IsDomain to avoid re-splitting.
func IsEmail(email string) bool {
	at := strings.LastIndex(email, "@")
	if at < 1 || at == len(email)-1 {
		return false
	}
	return IsDomain(email[at+1:])
}

// Count returns the number of distinct domains currently loaded — useful for
// startup logs ("disposable: 159k domains loaded") and CI assertions that
// the embedded list did not regress in size.
func Count() int {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return len(domainSet)
}

func normalize(domain string) string {
	d := strings.TrimSpace(strings.ToLower(domain))
	if d == "" {
		return ""
	}
	ascii, err := idna.ToASCII(d)
	if err != nil || !isValidASCIIHostname(ascii) {
		return ""
	}
	return ascii
}

func isValidASCIIHostname(domain string) bool {
	if domain == "" || len(domain) > 253 {
		return false
	}
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
				continue
			}
			return false
		}
	}
	return true
}
