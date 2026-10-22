package gostyle

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runGit runs git in dir, failing the test on error.
func runGit(
	t *testing.T,
	dir string,
	args ...string,
) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(
		os.Environ(),
		"GIT_AUTHOR_NAME=t",
		"GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t",
		"GIT_COMMITTER_EMAIL=t@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf(
			"git %v: %v\n%s",
			args,
			err,
			out,
		)
	}
}

func TestChangedLines(t *testing.T) {
	tests := []struct {
		name   string
		config []string // git config key=value pairs
		file   string
	}{
		{name: "defaults", file: "a.go"},
		{
			name:   "diff.noprefix",
			config: []string{"diff.noprefix=true"},
			file:   "a.go",
		},
		{
			name:   "diff.mnemonicPrefix",
			config: []string{"diff.mnemonicPrefix=true"},
			file:   "a.go",
		},
		{
			name:   "non-ASCII file name is quoted by git",
			config: []string{"core.quotePath=true"},
			file:   "ål.go",
		},
		{
			name:   "file in subdirectory",
			config: []string{"diff.renames=copies"},
			file:   "sub/a.go",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			runGit(
				t,
				dir,
				"init",
				"-q",
				".",
			)
			for _, kv := range tc.config {
				key, value, _ := strings.Cut(kv, "=")
				runGit(
					t,
					dir,
					"config",
					key,
					value,
				)
			}
			path := filepath.Join(dir, tc.file)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			write := func(content string) {
				if err := os.WriteFile(
					path,
					[]byte(content),
					0o644,
				); err != nil {
					t.Fatal(err)
				}
			}
			write("package a\n")
			runGit(
				t,
				dir,
				"add",
				".",
			)
			runGit(
				t,
				dir,
				"commit",
				"-q",
				"-m",
				"init",
			)
			write("package a\n\nfunc f() {}\n")

			orig, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.Chdir(orig); err != nil {
					t.Errorf("restore working directory: %v", err)
				}
			})
			got, err := changedLines()
			if err != nil {
				t.Fatal(err)
			}
			ranges := got[path]
			if len(ranges) != 1 || ranges[0].from != 2 || ranges[0].to != 3 {
				t.Fatalf(
					"want lines 2-3 changed in %s, got %v",
					path,
					got,
				)
			}
		})
	}
}
