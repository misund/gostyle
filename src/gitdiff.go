package gostyle

import (
	"fmt"
	"go/token"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type lineRange struct{ from, to int }

// changedLines maps absolute .go file paths to the line ranges that were added
// or modified relative to HEAD. Untracked files map to their whole extent.
func changedLines() (
	map[string][]lineRange,
	error,
) {
	top, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	top = strings.TrimSpace(top)
	out := map[string][]lineRange{}

	// diff-index is plumbing: unlike `git diff` it ignores user config such as
	// diff.noprefix and diff.mnemonicPrefix, which change the "+++ b/" paths
	// parsed below. The prefixes are pinned as well, for good measure.
	diff, err := git(
		"diff-index",
		"-p",
		"-U0",
		"--no-color",
		"--no-ext-diff",
		"--no-renames",
		"--src-prefix=a/",
		"--dst-prefix=b/",
		"HEAD",
		"--",
		"*.go",
	)
	if err != nil {
		return nil, err
	}
	var cur string
	for _, l := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(l, "+++ "):
			cur = ""
			if p, ok := diffPath(strings.TrimPrefix(l, "+++ ")); ok {
				cur = filepath.Join(top, p)
			}
		case strings.HasPrefix(l, "@@") && cur != "":
			// @@ -a,b +c,d @@
			plus := l[strings.Index(l, "+")+1:]
			plus = plus[:strings.Index(plus, " ")]
			start, count := plus, "1"
			if before, after, ok := strings.Cut(plus, ","); ok {
				start, count = before, after
			}
			s, err := strconv.Atoi(start)
			if err != nil {
				return nil, fmt.Errorf(
					"parse hunk header %q: %w",
					l,
					err,
				)
			}
			n, err := strconv.Atoi(count)
			if err != nil {
				return nil, fmt.Errorf(
					"parse hunk header %q: %w",
					l,
					err,
				)
			}
			if n > 0 {
				out[cur] = append(out[cur], lineRange{s, s + n - 1})
			}
		}
	}

	untracked, err := git(
		"ls-files",
		"--others",
		"--exclude-standard",
		"-z",
		"--",
		"*.go",
	)
	if err != nil {
		return nil, err
	}
	for _, p := range strings.Split(untracked, "\x00") {
		if p != "" {
			out[filepath.Join(top, p)] = []lineRange{{1, 1 << 30}}
		}
	}
	return out, nil
}

func git(args ...string) (
	string,
	error,
) {
	b, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", fmt.Errorf(
			"git %s: %w",
			strings.Join(args, " "),
			err,
		)
	}
	return string(b), nil
}

func inRanges(
	ranges map[string][]lineRange,
	pos token.Position,
) bool {
	abs, err := filepath.Abs(pos.Filename)
	if err != nil {
		return true
	}
	for _, r := range ranges[abs] {
		if pos.Line >= r.from && pos.Line <= r.to {
			return true
		}
	}
	return false
}

// diffPath extracts the file path from the target of a "+++" header. Git
// quotes paths containing special characters (C-style escapes). It reports
// false for /dev/null and unparsable targets.
func diffPath(target string) (
	string,
	bool,
) {
	if strings.HasPrefix(target, `"`) {
		unquoted, err := strconv.Unquote(target)
		if err != nil {
			return "", false
		}
		target = unquoted
	}
	p, ok := strings.CutPrefix(target, "b/")

	return p, ok
}
