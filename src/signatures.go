package gostyle

import (
	"go/ast"
	"strings"
)

func (c *checker) checkSignature(ft *ast.FuncType) {
	if ft.Params != nil && countItems(ft.Params) > 1 {
		c.checkList(
			ft.Params,
			"function signature with more than one parameter",
		)
	}
	// Multi-value results are only a list when parenthesized.
	if ft.Results != nil && ft.Results.Opening.IsValid() && countItems(
		ft.Results,
	) > 1 {
		c.checkList(
			ft.Results,
			"function signature with more than one return value",
		)
	}
}

func countItems(fl *ast.FieldList) int {
	n := 0
	for _, f := range fl.List {
		if len(f.Names) == 0 {
			n++
		} else {
			n += len(f.Names)
		}
	}
	return n
}

// checkList requires every item on its own line, with the opening and closing
// parentheses on lines of their own.
func (c *checker) checkList(
	fl *ast.FieldList,
	what string,
) {
	prev := c.line(fl.Opening)
	ok := true
	for _, f := range fl.List {
		for _, n := range f.Names {
			if c.line(n.Pos()) == prev {
				ok = false
			}
			prev = c.line(n.End())
		}
		if len(f.Names) == 0 && c.line(f.Pos()) == prev {
			ok = false
		}
		prev = c.line(f.End())
	}
	if c.line(fl.Closing) == prev {
		ok = false
	}
	if !ok {
		var items []string
		for _, f := range fl.List {
			if len(f.Names) == 0 {
				items = append(
					items,
					c.text(
						f.Type.Pos(),
						f.Type.End(),
					),
				)
				continue
			}
			var names []string
			for _, n := range f.Names {
				names = append(names, n.Name)
			}
			items = append(
				items,
				strings.Join(
					names,
					",\n",
				)+" "+c.text(
					f.Type.Pos(),
					f.Type.End(),
				),
			)
		}
		c.reportList(
			fl.Opening,
			fl.Closing,
			items,
			what+": put each item on its own line with a trailing comma",
		)
	}
}
