package gostyle

import (
	"fmt"
	"go/ast"
	"strings"
)

// checkTopCall is invoked for every call; calls nested in an argument list are
// handled from their outermost required ancestor, so only visit calls that are
// not arguments of another call here. (ast.Inspect visits all calls, so we
// evaluate each and rely on the "forced" logic via depth only for roots.)
func (c *checker) checkTopCall(call *ast.CallExpr) {
	if len(call.Args) == 0 {
		return
	}
	if reason := c.requirement(call); reason != "" {
		c.requireSplit(call, reason)
		return
	}
	// Forced splitting: a call with depth >= 3 forces its inner calls. That is
	// handled when the root is visited, see requireSplit.
}

func callDepth(e ast.Expr) int {
	call, ok := unparen(e).(*ast.CallExpr)
	if !ok {
		return 0
	}
	deepest := 0
	for _, a := range call.Args {
		if d := callDepth(a); d > deepest {
			deepest = d
		}
	}
	return 1 + deepest
}

func unparen(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}

// requirement explains why call must be split, or returns "".
func (c *checker) requirement(call *ast.CallExpr) string {
	if len(call.Args) >= 3 {
		return "call with three or more arguments"
	}
	if callDepth(call) >= 3 {
		return "call nesting function calls three or more layers deep"
	}
	if c.inline(call) && !plainSingleArg(call) && c.endColumn(call) > c.width {
		return fmt.Sprintf("call exceeds %d columns", c.width)
	}
	return ""
}

// plainSingleArg reports whether call takes exactly one argument that is a
// plain identifier or selector chain such as a or a.b.c. Such calls are exempt
// from the width rule: splitting them gains nothing when the line is long
// because of the surrounding expression.
func plainSingleArg(call *ast.CallExpr) bool {
	if len(call.Args) != 1 || call.Ellipsis.IsValid() {
		return false
	}
	e := call.Args[0]
	for {
		switch x := e.(type) {
		case *ast.Ident:
			return true
		case *ast.SelectorExpr:
			e = x.X
		default:
			return false
		}
	}
}

// nestedCalls returns every call that occurs within the arguments of another
// call, at any depth.
func nestedCalls(f *ast.File) map[*ast.CallExpr]bool {
	nested := map[*ast.CallExpr]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		for _, arg := range call.Args {
			ast.Inspect(arg, func(m ast.Node) bool {
				if inner, ok := m.(*ast.CallExpr); ok {
					nested[inner] = true
				}
				return true
			})
		}
		return true
	})
	return nested
}

func (c *checker) inline(call *ast.CallExpr) bool {
	return c.line(call.Lparen) == c.line(call.Rparen)
}

// endColumn is the width of the source line on which the call ends. For a call
// nested in another call's arguments it is only the width up to the end of the
// call itself, so that the call is not blamed for overflow that comes from its
// parent or from what follows it. Only the outermost call is measured against
// the whole line, which includes trailing text such as "; err != nil {".
func (c *checker) endColumn(call *ast.CallExpr) int {
	file := c.fset.File(call.Pos())
	line := c.line(call.End())
	start := file.Offset(file.LineStart(line))
	end := len(c.src)
	if line < file.LineCount() {
		end = file.Offset(file.LineStart(line + 1))
	}
	if c.nested[call] {
		end = file.Offset(call.End())
	}
	text := strings.TrimRight(string(c.src[start:end]), "\r\n")
	cols := 0
	for _, r := range text {
		if r == '\t' {
			cols += c.tab
		} else {
			cols++
		}
	}
	return cols
}

// requireSplit verifies call is expanded one argument per line and, when the
// call is split, that its argument calls (which take arguments) are too.
func (c *checker) requireSplit(
	call *ast.CallExpr,
	reason string,
) {
	prev := c.line(call.Lparen)
	ok := true
	for _, a := range call.Args {
		if c.line(a.Pos()) == prev {
			ok = false
		}
		prev = c.line(a.End())
	}
	if c.line(call.Rparen) == prev {
		ok = false
	}
	if !ok {
		var items []string
		for _, a := range call.Args {
			items = append(
				items,
				c.text(
					a.Pos(),
					a.End(),
				),
			)
		}
		if call.Ellipsis.IsValid() && len(items) > 0 {
			// The spread marker lies outside the last argument's range.
			items[len(items)-1] += "..."
		}
		c.reportList(
			call.Lparen,
			call.Rparen,
			items,
			reason+": put each argument on its own line with a trailing comma",
		)
	}
	// Inner calls of a split call are split too.
	for _, a := range call.Args {
		if inner, isCall := unparen(
			a,
		).(*ast.CallExpr); isCall && len(
			inner.Args,
		) > 0 && callDepth(
			call,
		) >= 3 {
			c.requireSplit(inner, "inner call of a split nested call")
		}
	}
}
