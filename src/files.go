package gostyle

import (
	"go/ast"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func collect(root string) (
	[]string,
	error,
) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{root}, nil
	}
	var files []string
	err = filepath.WalkDir(root, func(
		path string,
		d fs.DirEntry,
		err error,
	) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (name == "vendor" || name == "node_modules" || strings.HasPrefix(
				name,
				".",
			)) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func isGenerated(f *ast.File) bool {
	for _, cg := range f.Comments {
		if cg.Pos() > f.Package {
			break
		}
		for _, cm := range cg.List {
			if strings.HasPrefix(
				cm.Text,
				"// Code generated",
			) && strings.Contains(
				cm.Text,
				"DO NOT EDIT",
			) {
				return true
			}
		}
	}
	return false
}
