// Command gostyle checks Go files against the formatting rules described in
// README.md. It reports violations as
// file:line:col: message and exits 1 if any were found.
//
// Usage: gostyle [-width 80] [-tab 4] [-changed] [-fix] path...
//
// Each path is a .go file or a directory (searched recursively). A trailing
// "/..." is accepted. Generated files and vendor/ are skipped.
//
// With -changed, only violations on lines added or modified relative to HEAD
// (plus all lines of untracked files) are reported, so existing code is left
// alone. Without paths, -changed checks every changed .go file in the git
// repository containing the current directory.
//
// With -fix, auto-fixable violations are rewritten in place before the
// remaining ones are reported.
package main

import (
	"os"

	gostyle "github.com/misund/gostyle/src"
)

func main() {
	os.Exit(gostyle.Main())
}
