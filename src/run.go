// Package gostyle implements the gostyle command: a checker and auto-fixer for
// one-item-per-line parameter and argument lists. See the README for the rules.
package gostyle

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Main runs the command-line interface using os.Args and returns the process
// exit status: 0 when clean, 1 when violations remain, 2 on errors.
func Main() int {
	width := flag.Int(
		"width",
		80,
		"maximum columns for an inline call",
	)
	tab := flag.Int(
		"tab",
		4,
		"columns per tab when measuring line width",
	)
	changed := flag.Bool(
		"changed",
		false,
		"only report violations on lines changed relative to HEAD",
	)
	fix := flag.Bool(
		"fix",
		false,
		"rewrite files to fix auto-fixable violations, then report the rest",
	)
	flag.Parse()

	var files []string
	var ranges map[string][]lineRange
	if *changed {
		var err error
		ranges, err = changedLines()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		if flag.NArg() == 0 {
			for f := range ranges {
				files = append(files, f)
			}
			sort.Strings(files)
		}
	}
	for _, arg := range flag.Args() {
		arg = strings.TrimSuffix(strings.TrimSuffix(arg, "..."), "/")
		if arg == "" {
			arg = "."
		}
		found, err := collect(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		files = append(files, found...)
	}

	failed := false
	for _, file := range files {
		if *fix {
			if err := fixFile(
				file,
				*width,
				*tab,
				func(v violation) bool {
					return !*changed || inRanges(ranges, v.pos)
				},
			); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 2
			}
		}
		vs, err := checkFile(
			file,
			*width,
			*tab,
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		for _, v := range vs {
			if *changed && !inRanges(ranges, v.pos) {
				continue
			}
			failed = true
			fmt.Printf(
				"%s: %s\n",
				v.pos,
				v.msg,
			)
		}
	}
	if failed {
		return 1
	}

	return 0
}
