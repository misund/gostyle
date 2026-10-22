package gostyle

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func all(violation) bool { return true }

// fix runs the fixer on src with the default width and tab size.
func fix(
	t *testing.T,
	src string,
) string {
	t.Helper()
	out, err := fixSource(
		"x.go",
		[]byte(src),
		80,
		4,
		all,
	)
	if err != nil {
		t.Fatalf("fixSource: %v", err)
	}

	return string(out)
}

// check returns the violation messages for src.
func check(
	t *testing.T,
	src string,
) []string {
	t.Helper()
	vs, err := checkSource(
		"x.go",
		[]byte(src),
		80,
		4,
	)
	if err != nil {
		t.Fatalf("checkSource: %v", err)
	}
	var msgs []string
	for _, v := range vs {
		msgs = append(msgs, v.msg)
	}

	return msgs
}

func TestFix(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "multi-parameter signature",
			in:   "package a\n\nfunc f(a int, b string) {}\n",
			want: "package a\n\nfunc f(\n\ta int,\n\tb string,\n) {\n}\n",
		},
		{
			name: "grouped names count as separate parameters",
			in:   "package a\n\nfunc f(a, b int) {}\n",
			want: "package a\n\nfunc f(\n\ta,\n\tb int,\n) {\n}\n",
		},
		{
			name: "multiple return values",
			in:   "package a\n\nfunc f() (int, error) { return 0, nil }\n",
			want: "package a\n\nfunc f() (\n\tint,\n\terror,\n) {\n\treturn 0, nil\n}\n",
		},
		{
			name: "single parameter stays inline",
			in:   "package a\n\nfunc f(a int) error { return nil }\n",
			want: "package a\n\nfunc f(a int) error { return nil }\n",
		},
		{
			name: "func literal",
			in:   "package a\n\nvar _ = func(a, b int) {}\n",
			want: "package a\n\nvar _ = func(\n\ta,\n\tb int,\n) {\n}\n",
		},
		{
			name: "interface method",
			in:   "package a\n\ntype I interface{ M(a int, b int) }\n",
			want: "package a\n\ntype I interface {\n\tM(\n\t\ta int,\n\t\tb int,\n\t)\n}\n",
		},
		{
			name: "three arguments",
			in:   "package a\n\nfunc g(a, b, c int) {}\n\nfunc f() { g(1, 2, 3) }\n",
			want: "package a\n\nfunc g(\n\ta,\n\tb,\n\tc int,\n) {\n}\n\nfunc f() {\n\tg(\n\t\t1,\n\t\t2,\n\t\t3,\n\t)\n}\n",
		},
		{
			name: "variadic spread is preserved",
			in:   "package a\n\nfunc g(a, b int, c ...int) {}\n\nfunc f(x []int) { g(1, 2, x...) }\n",
			want: "package a\n\nfunc g(\n\ta,\n\tb int,\n\tc ...int,\n) {\n}\n\nfunc f(x []int) {\n\tg(\n\t\t1,\n\t\t2,\n\t\tx...,\n\t)\n}\n",
		},
		{
			name: "nesting three layers deep splits inner calls",
			in:   "package a\n\nfunc f() { a(b(c(1))) }\n\nfunc a(int) int { return 0 }\nfunc b(int) int { return 0 }\nfunc c(int) int { return 0 }\n",
			want: "package a\n\nfunc f() {\n\ta(\n\t\tb(\n\t\t\tc(1),\n\t\t),\n\t)\n}\n\nfunc a(int) int { return 0 }\nfunc b(int) int { return 0 }\nfunc c(int) int { return 0 }\n",
		},
		{
			name: "two layers stay inline",
			in:   "package a\n\nfunc f() { a(b(1)) }\n\nfunc a(int) int { return 0 }\nfunc b(int) int { return 0 }\n",
			want: "package a\n\nfunc f() { a(b(1)) }\n\nfunc a(int) int { return 0 }\nfunc b(int) int { return 0 }\n",
		},
		{
			name: "over 80 columns splits the outermost call only",
			in: "package a\n\nfunc g(a, b string) {}\nfunc h(a string, n int) string { return a }\n\n" +
				"func f() {\n\tg(h(\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\", 1), h(\"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\", 2))\n}\n",
			want: "package a\n\nfunc g(\n\ta,\n\tb string,\n) {\n}\nfunc h(\n\ta string,\n\tn int,\n) string {\n\treturn a\n}\n\n" +
				"func f() {\n\tg(\n\t\th(\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\", 1),\n\t\th(\"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\", 2),\n\t)\n}\n",
		},
		{
			name: "single plain argument is exempt from the width rule",
			in: "package a\n\nfunc u(string) string { return \"\" }\n\nfunc f(someVeryLongIdentifierNameHere struct{ field string }) {\n" +
				"\tveryLongVariableNameForTheResultOfTheCall := u(someVeryLongIdentifierNameHere.field)\n\t_ = veryLongVariableNameForTheResultOfTheCall\n}\n",
			want: "package a\n\nfunc u(string) string { return \"\" }\n\nfunc f(someVeryLongIdentifierNameHere struct{ field string }) {\n" +
				"\tveryLongVariableNameForTheResultOfTheCall := u(someVeryLongIdentifierNameHere.field)\n\t_ = veryLongVariableNameForTheResultOfTheCall\n}\n",
		},
		{
			name: "call in an if header keeps the header",
			in: "package a\n\nfunc g(a, b string) error { return nil }\n\n" +
				"func f() {\n\tif err := g(\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\", \"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\"); err != nil {\n\t}\n}\n",
			want: "package a\n\nfunc g(\n\ta,\n\tb string,\n) error {\n\treturn nil\n}\n\n" +
				"func f() {\n\tif err := g(\n\t\t\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\",\n\t\t\"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\",\n\t); err != nil {\n\t}\n}\n",
		},
		{
			name: "comment inside the parentheses is not auto-fixed",
			in:   "package a\n\nfunc g(a, b, c int) {}\n\nfunc f() { g(1 /* one */, 2, 3) }\n",
			want: "package a\n\nfunc g(\n\ta,\n\tb,\n\tc int,\n) {\n}\n\nfunc f() { g(1 /* one */, 2, 3) }\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fix(t, tc.in)
			if got != tc.want {
				t.Errorf(
					"fix mismatch\n--- got ---\n%s\n--- want ---\n%s",
					got,
					tc.want,
				)
			}
			if _, err := parser.ParseFile(
				token.NewFileSet(),
				"x.go",
				got,
				0,
			); err != nil {
				t.Errorf("fixed code does not parse: %v", err)
			}
			// Fixing is idempotent.
			if again := fix(t, got); again != got {
				t.Errorf(
					"fix is not idempotent\n--- first ---\n%s\n--- second ---\n%s",
					got,
					again,
				)
			}
		})
	}
}

func TestCheckReports(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string // substring of an expected message; "" means none expected
	}{
		{
			name: "clean code",
			in:   "package a\n\nfunc f(\n\ta int,\n\tb int,\n) {\n}\n",
			want: "",
		},
		{
			name: "signature",
			in:   "package a\n\nfunc f(a, b int) {}\n",
			want: "function signature with more than one parameter",
		},
		{
			name: "return values",
			in:   "package a\n\nfunc f() (int, error) { return 0, nil }\n",
			want: "more than one return value",
		},
		{
			name: "argument count",
			in:   "package a\n\nfunc g(\n\ta,\n\tb,\n\tc int,\n) {\n}\n\nfunc f() { g(1, 2, 3) }\n",
			want: "three or more arguments",
		},
		{
			name: "width",
			in: "package a\n\nfunc g(\n\ta,\n\tb string,\n) {\n}\n\nfunc f() {\n" +
				"\tg(\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\", \"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\")\n}\n",
			want: "exceeds 80 columns",
		},
		{
			name: "tabs count as four columns",
			in: "package a\n\nfunc g(\n\ta,\n\tb string,\n) {\n}\n\nfunc f() {\n" +
				"\t\t\tg(\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\", \"bbbbbbbbbbbbbbbbbbbbbbbbb\")\n}\n",
			want: "exceeds 80 columns",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgs := check(t, tc.in)
			if tc.want == "" {
				if len(msgs) != 0 {
					t.Fatalf("want no violations, got %v", msgs)
				}

				return
			}
			for _, m := range msgs {
				if strings.Contains(m, tc.want) {
					return
				}
			}
			t.Fatalf(
				"want a violation containing %q, got %v",
				tc.want,
				msgs,
			)
		})
	}
}

func TestGeneratedFilesSkipped(t *testing.T) {
	src := "// Code generated by x. DO NOT EDIT.\n\npackage a\n\nfunc f(a, b int) {}\n"
	if msgs := check(t, src); len(msgs) != 0 {
		t.Fatalf("generated file should be skipped, got %v", msgs)
	}
}
