// Package gostyle implements the gostyle command: a checker and auto-fixer for
// one-item-per-line parameter and argument lists. See the README for the rules.
package gostyle

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Main runs the command-line interface using os.Args and returns the process
// exit status: 0 when clean, 1 when violations remain, 2 on errors.
func Main() int {
	return Run(
		os.Args[1:],
		os.Stdout,
		os.Stderr,
	)
}

// Run is Main with explicit arguments and output streams, so that it can be
// tested. args excludes the program name.
func Run(
	args []string,
	stdout,
	stderr io.Writer,
) int {
	flags := flag.NewFlagSet("gostyle", flag.ContinueOnError)
	flags.SetOutput(stderr)

	width := flags.Int(
		"width",
		80,
		"maximum columns for an inline call",
	)
	tab := flags.Int(
		"tab",
		4,
		"columns per tab when measuring line width",
	)
	changed := flags.Bool(
		"changed",
		false,
		"only report violations on lines changed relative to HEAD",
	)
	fix := flags.Bool(
		"fix",
		false,
		"rewrite files to fix auto-fixable violations, then report the rest",
	)
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}

		return 2
	}

	var files []string
	var ranges map[string][]lineRange
	if *changed {
		var err error
		ranges, err = changedLines()
		if err != nil {
			printErr(stderr, err)
			return 2
		}
		if flags.NArg() == 0 {
			for f := range ranges {
				files = append(files, f)
			}
			sort.Strings(files)
		}
	}
	for _, arg := range flags.Args() {
		arg = strings.TrimSuffix(strings.TrimSuffix(arg, "..."), "/")
		if arg == "" {
			arg = "."
		}
		found, err := collect(arg)
		if err != nil {
			printErr(stderr, err)
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
				printErr(stderr, err)
				return 2
			}
		}
		vs, err := checkFile(
			file,
			*width,
			*tab,
		)
		if err != nil {
			printErr(stderr, err)
			return 2
		}
		for _, v := range vs {
			if *changed && !inRanges(ranges, v.pos) {
				continue
			}
			failed = true
			printViolation(stdout, v)
		}
	}
	if failed {
		return 1
	}

	return 0
}

// printErr writes err to w. A failure to write diagnostics has no useful
// recovery, so it is deliberately ignored.
func printErr(
	w io.Writer,
	err error,
) {
	_, _ = fmt.Fprintln(w, err)
}

// printViolation writes v to w as "file:line:col: message".
func printViolation(
	w io.Writer,
	v violation,
) {
	_, _ = fmt.Fprintf(
		w,
		"%s: %s\n",
		v.pos,
		v.msg,
	)
}
