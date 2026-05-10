// disposable-check is a tiny CLI that reports whether each input is a
// disposable email or domain. Useful for shell pipelines, ad hoc lookups,
// and CI smoke tests.
//
//	$ echo user@mailinator.com | disposable-check
//	disposable	user@mailinator.com
//
//	$ disposable-check alice@example.com user@10minutemail.com
//	clean       	alice@example.com
//	disposable  	user@10minutemail.com
//
// Exit code is 0 if all inputs are clean, 1 if any input is disposable, 2 on
// argument or I/O errors. This makes it easy to wire into preflight scripts:
//
//	if ! disposable-check "$EMAIL"; then ...
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/billionverify/disposable"
)

func main() {
	count := flag.Bool("count", false, "print only the loaded domain count and exit")
	quiet := flag.Bool("quiet", false, "suppress per-line output, use exit code only")
	flag.Parse()

	if *count {
		fmt.Println(disposable.Count())
		return
	}

	inputs := flag.Args()
	if len(inputs) == 0 {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			inputs = append(inputs, sc.Text())
		}
		if err := sc.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "read stdin:", err)
			os.Exit(2)
		}
	}

	if len(inputs) == 0 {
		fmt.Fprintln(os.Stderr, "usage: disposable-check [flags] <email|domain> [...]   (or pipe one entry per line on stdin)")
		os.Exit(2)
	}

	anyDisposable := false
	for _, raw := range inputs {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		hit := classify(entry)
		if hit {
			anyDisposable = true
		}
		if *quiet {
			continue
		}
		label := "clean"
		if hit {
			label = "disposable"
		}
		fmt.Printf("%-11s\t%s\n", label, entry)
	}
	if anyDisposable {
		os.Exit(1)
	}
}

// classify routes the entry through IsEmail when it looks like one, otherwise
// IsDomain. Treating the @ as the discriminator keeps the CLI terse — no flags
// needed to switch modes.
func classify(entry string) bool {
	if strings.Contains(entry, "@") {
		return disposable.IsEmail(entry)
	}
	return disposable.IsDomain(entry)
}
