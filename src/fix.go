package gostyle

import (
	"fmt"
	"go/format"
	"os"
	"sort"
)

// maxFixPasses bounds the fix loop; nested splits need one pass per level.
const maxFixPasses = 10

// fixFile fixes the file at path in place, if anything changed.
func fixFile(
	path string,
	width,
	tab int,
	want func(violation) bool,
) error {
	orig, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fixed, err := fixSource(
		path,
		orig,
		width,
		tab,
		want,
	)
	if err != nil {
		return err
	}
	if string(fixed) == string(orig) {
		return nil
	}

	return os.WriteFile(
		path,
		fixed,
		0o644,
	)
}

// fixSource applies fixes for violations accepted by want, outermost first,
// repeating until nothing more changes.
func fixSource(
	path string,
	orig []byte,
	width,
	tab int,
	want func(violation) bool,
) (
	[]byte,
	error,
) {
	src := orig
	for range maxFixPasses {
		vs, err := checkSource(
			path,
			src,
			width,
			tab,
		)
		if err != nil {
			return nil, err
		}
		var edits []edit
		for _, v := range vs {
			if v.fix != nil && want(v) {
				edits = append(edits, *v.fix)
			}
		}
		sort.Slice(
			edits,
			func(
				i,
				j int,
			) bool {
				return edits[i].start < edits[j].start
			},
		)
		var out []byte
		last, applied := 0, 0
		for _, e := range edits {
			if e.start < last {
				continue // nested in an edit already applied; next pass
			}
			out = append(out, src[last:e.start]...)
			out = append(out, e.text...)
			last = e.end
			applied++
		}
		if applied == 0 {
			break
		}
		out = append(out, src[last:]...)
		formatted, err := format.Source(out)
		if err != nil {
			return nil, fmt.Errorf(
				"%s: fix produced invalid code: %w",
				path,
				err,
			)
		}
		src = formatted
	}

	return src, nil
}
