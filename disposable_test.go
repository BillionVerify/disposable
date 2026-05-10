package disposable

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestIsDomain_Hit(t *testing.T) {
	if !IsDomain("mailinator.com") {
		t.Fatal("mailinator.com must be flagged as disposable in the embedded table")
	}
	if !IsDomain("MAILINATOR.com") {
		t.Fatal("lookup must be case-insensitive")
	}
	if !IsDomain("  mailinator.com  ") {
		t.Fatal("leading/trailing whitespace must be stripped")
	}
}

func TestIsDomain_Miss(t *testing.T) {
	if IsDomain("example.com") {
		t.Fatal("example.com is not disposable; lookup must return false")
	}
	if IsDomain("") {
		t.Fatal("empty input must not produce a hit")
	}
}

func TestIsEmail(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"user@mailinator.com", true},
		{"user@MAILINATOR.com", true},
		{"alice@example.com", false},
		{"not-an-email", false},
		{"", false},
		{"@example.com", false},
		{"alice@", false},
	}
	for _, c := range cases {
		if got := IsEmail(c.in); got != c.want {
			t.Errorf("IsEmail(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestSetData_OverridesEmbeddedAndExceptions(t *testing.T) {
	defer reload()
	SetData(
		[]string{"trashy.example", "also-bad.example"},
		[]string{"*.tempmail.example"},
		[]string{"trashy.example"}, // exception wins over the same listing
	)
	if IsDomain("trashy.example") {
		t.Fatal("exceptions must beat exact-match disposable entries")
	}
	if !IsDomain("also-bad.example") {
		t.Fatal("override list must take effect")
	}
	if IsDomain("mailinator.com") {
		t.Fatal("override should fully replace the embedded list, not merge")
	}
	if !IsDomain("foo.tempmail.example") {
		t.Fatal("wildcard suffix match must work")
	}
	if !IsDomain("tempmail.example") {
		t.Fatal("the wildcard suffix itself must also match exactly (not just subdomains)")
	}
}

func TestSetData_NormalizesWildcardIDN(t *testing.T) {
	defer reload()
	SetData(nil, []string{"*.B\u00fccher.example"}, nil)
	if !IsDomain("shop.xn--bcher-kva.example") {
		t.Fatal("wildcard entries must be normalized to IDN ASCII form")
	}
	if !IsDomain("b\u00fccher.example") {
		t.Fatal("wildcard suffix itself must match after IDN normalization")
	}
}

func TestParseLines_NormalizesAndDropsInvalidDomains(t *testing.T) {
	entries := parseLines([]byte("M\u00f6ller.example\n404: not found0-00.usa.cc\n"), false)
	wantIDN := normalize("M\u00f6ller.example")
	if wantIDN == "" {
		t.Fatal("test fixture must normalize to a non-empty ASCII domain")
	}
	if _, ok := entries[wantIDN]; !ok {
		t.Fatal("parseLines must normalize IDN entries to ASCII")
	}
	if _, ok := entries["404: not found0-00.usa.cc"]; ok {
		t.Fatal("parseLines must drop malformed domains")
	}

	wildcards := parseLines([]byte("*.B\u00fccher.example\n"), true)
	if _, ok := wildcards["xn--bcher-kva.example"]; !ok {
		t.Fatal("parseLines must strip wildcard prefixes before normalization")
	}
}

func TestEmbeddedDataFiles_NormalizedSortedUnique(t *testing.T) {
	assertNormalizedSortedUnique(t, "domains", embeddedDomains, false)
	assertNormalizedSortedUnique(t, "wildcards", embeddedWildcards, true)
	assertNormalizedSortedUnique(t, "exceptions", embeddedExceptions, false)
}

func TestCount_NonEmpty(t *testing.T) {
	defer reload()
	if c := Count(); c < 100_000 {
		// Sanity guard: we've shipped ~160k entries since the initial import,
		// catching a regression that emptied data/domains.txt is more useful
		// than asserting an exact number that drifts every day.
		t.Fatalf("Count() = %d; embedded list must contain at least 100k domains", c)
	}
}

// reload resets the package state so tests can re-init from the embedded
// data after they've called SetData. Without this, table tests would see
// each other's overrides.
func reload() { ResetEmbedded() }

func assertNormalizedSortedUnique(t *testing.T, name string, data []byte, stripWildcardPrefix bool) {
	t.Helper()
	var prev string
	sc := bufio.NewScanner(bytes.NewReader(data))
	lineNo := 0
	for sc.Scan() {
		lineNo++
		raw := strings.TrimSpace(sc.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		entry := raw
		if stripWildcardPrefix && strings.HasPrefix(entry, "*.") {
			entry = entry[2:]
		}
		norm := normalize(entry)
		if norm == "" {
			t.Fatalf("%s line %d is malformed: %q", name, lineNo, raw)
		}
		if norm != entry {
			t.Fatalf("%s line %d is not normalized: got %q, want %q", name, lineNo, raw, norm)
		}
		if prev != "" && entry <= prev {
			t.Fatalf("%s line %d is not strictly sorted and unique: %q after %q", name, lineNo, entry, prev)
		}
		prev = entry
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan %s: %v", name, err)
	}
}
