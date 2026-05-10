package disposable

import (
	"bufio"
	"bytes"
	_ "embed"
	"strings"
)

//go:embed data/domains.txt
var embeddedDomains []byte

//go:embed data/wildcards.txt
var embeddedWildcards []byte

//go:embed data/exceptions.txt
var embeddedExceptions []byte

// loadEmbedded populates the in-memory tables from the embedded text files.
//
// Each file is one entry per line. Blank lines and lines starting with `#`
// are ignored, so contributors can leave comments next to entries that need
// explanation (e.g. "# kept after 2026-04 incident — looked clean to upstream
// but used by spam ring X").
func loadEmbedded() {
	ds := parseLines(embeddedDomains, false)
	es := parseLines(embeddedExceptions, false)
	wildcardSet := parseLines(embeddedWildcards, true)
	ws := make([]string, 0, len(wildcardSet))
	for w := range wildcardSet {
		ws = append(ws, w)
	}

	stateMu.Lock()
	domainSet = ds
	exceptionSet = es
	wildcardSuffix = ws
	stateMu.Unlock()
}

func parseLines(data []byte, stripWildcardPrefix bool) map[string]struct{} {
	out := make(map[string]struct{})
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if stripWildcardPrefix && strings.HasPrefix(line, "*.") {
			line = line[2:]
		}
		if line = normalize(line); line != "" {
			out[line] = struct{}{}
		}
	}
	return out
}
