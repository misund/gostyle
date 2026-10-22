package gostyle

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
)

type checker struct {
	fset  *token.FileSet
	src   []byte
	width int
	tab   int
	out   []violation

	comments []*ast.CommentGroup

	// nested holds calls that appear inside another call's arguments.
	nested map[*ast.CallExpr]bool
}

type violation struct {
	pos token.Position
	msg string
	fix *edit // nil when not auto-fixable (for example, comments inside)
}

// edit replaces src[start:end] with text.
type edit struct {
	start, end int
	text       string
}

func checkFile(
	path string,
	width,
	tab int,
) (
	[]violation,
	error,
) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return checkSource(
		path,
		src,
		width,
		tab,
	)
}

func checkSource(
	path string,
	src []byte,
	width,
	tab int,
) (
	[]violation,
	error,
) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(
		fset,
		path,
		src,
		parser.ParseComments|parser.SkipObjectResolution,
	)
	if err != nil {
		return nil, err
	}
	if isGenerated(f) {
		return nil, nil
	}
	c := &checker{
		fset:     fset,
		src:      src,
		width:    width,
		tab:      tab,
		comments: f.Comments,
		nested:   nestedCalls(f),
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncType:
			c.checkSignature(n)
		case *ast.CallExpr:
			c.checkTopCall(n)
		}
		return true
	})
	sort.Slice(c.out, func(
		i,
		j int,
	) bool {
		a, b := c.out[i].pos, c.out[j].pos
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Column < b.Column
	})
	return c.out, nil
}

func (c *checker) line(p token.Pos) int { return c.fset.Position(p).Line }

// reportList reports a violation for the parenthesized list opening at open
// and closing at closing, with the given item source texts.
func (c *checker) reportList(
	open,
	closing token.Pos,
	items []string,
	msg string,
) {
	v := violation{pos: c.fset.Position(open), msg: msg}
	lo, hi := c.fset.Position(open).Offset, c.fset.Position(closing).Offset
	if !c.hasComment(lo, hi) {
		v.fix = &edit{
			start: lo + 1,
			end:   hi,
			text:  "\n" + strings.Join(items, ",\n") + ",\n",
		}
	}
	c.out = append(c.out, v)
}

func (c *checker) hasComment(
	lo,
	hi int,
) bool {
	for _, cg := range c.comments {
		o := c.fset.Position(cg.Pos()).Offset
		if o > lo && o < hi {
			return true
		}
	}
	return false
}

func (c *checker) text(
	from,
	to token.Pos,
) string {
	return string(
		c.src[c.fset.Position(from).Offset:c.fset.Position(to).Offset],
	)
}
