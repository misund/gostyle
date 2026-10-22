# gostyle

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

An opinionated Go formatting checker and auto-fixer. It enforces one rule
family that `gofmt` and `gofumpt` deliberately leave alone: **when to break
parameter and argument lists onto one item per line.**

It is a style preference, not a correctness tool. Use it only if you want this
exact style.

The rules were written for the agent instructions in
[misund/dotfiles](https://github.com/misund/dotfiles), so they are one person's
taste.

## The rules

Put each item on its own line, with a trailing comma, when:

- **Function signatures** have more than one parameter. Multi-value return
  lists are expanded the same way. This covers func literals, func types and
  interface methods. `a, b int` counts as two parameters. Single-parameter
  signatures stay inline.
- **Function calls**:
  - have three or more arguments, or
  - exceed 80 columns (a tab counts as 4), or
  - nest function calls three or more layers deep (`a(b(c(x)))` is three).

  When nesting is split, its inner calls are split too, unless they take no
  arguments. Calls with a single plain identifier or selector argument, like
  `len(x.y)`, are exempt from the width rule. A call nested in another call's
  arguments is measured up to its own end, not to the end of the line.

```go
func NewWithConfig(
	ctx context.Context,
	cfg *pgxpool.Config,
	opts ...otelpgx.Option,
) (
	*pgxpool.Pool,
	error,
) {
	// ...
}

func addTimeZone(conn *pgx.Conn) error { /* single param stays inline */ }

logger.ConfigureGlobalLogger(
	logger.WithHook(
		loghook.NewHook(),
	),
	logger.WithLevel(logger.LevelInfo),
)

addTimeZone(conn) // short, shallow, few args -> inline
```

Generated files (`// Code generated ... DO NOT EDIT.`), `vendor/`,
`node_modules/` and dot-directories are skipped.

## Install

It has no dependencies beyond the standard library and needs Go 1.22 or newer.

```sh
go install github.com/misund/gostyle@latest
```

This puts `gostyle` in `$(go env GOBIN)`, or `$(go env GOPATH)/bin` if that is
unset. Make sure that directory is on your `PATH`. To pin a version, use a tag
instead of `@latest`, for example `@v0.1.0`.

To build from a checkout instead:

```sh
git clone https://github.com/misund/gostyle
cd gostyle
go build -o ~/bin/gostyle .
```

## Usage

```sh
gostyle [-width 80] [-tab 4] [-changed] [-fix] [path...]
```

Each path is a `.go` file or a directory (searched recursively). A trailing
`/...` is accepted.

| Flag | Meaning |
|---|---|
| `-fix` | Rewrite files to fix auto-fixable violations, then report the rest. |
| `-changed` | Only report (or fix) violations on lines added or modified relative to `HEAD`, plus all lines of untracked files. With no paths, checks every changed `.go` file in the current git repository. |
| `-width` | Maximum columns for an inline call. Default 80. |
| `-tab` | Columns per tab when measuring width. Default 4. |

Output is `file:line:col: message`. The exit status is 0 when clean, 1 when
violations remain, and 2 on errors such as a file that does not parse.

Typical use before finishing a change:

```sh
gostyle -fix -changed
```

`-changed` lets you adopt the style incrementally: existing code is left alone
and only new or modified lines are held to the rules. It needs a git
repository with at least one commit.

## What `-fix` does and does not do

`-fix` rewrites a violating list so each item sits on its own line with a
trailing comma, then runs `gofmt` on the result. It repeats until nothing
changes (at most 10 passes), so nested calls are split level by level. It
keeps the variadic `...` marker. It changes layout only, never behavior, and
refuses to write a file if the result does not parse.

It leaves these alone and only reports them:

- A list with a **comment inside its parentheses**, because rewriting could
  move the comment.
- Anything that would need a restructure, such as a line that is long because
  of a `&&` or `+` chain. Extract a variable instead.

Not implemented: the rule that table-driven test rows should use keyed fields
instead of positional one-liners.

Because the fixer runs `gofmt` over the whole file, a file that was not
gofmt-clean before can have unrelated whitespace changes, even with
`-changed`.

## Compatibility with other tools

Tested against a real service (168 Go files, about 2600 violations fixed):

- `gofmt` and `goimports` report nothing afterwards.
- `gofumpt` flags the same files as before, none new.
- `golangci-lint` with a typical config (`errcheck`, `govet`, `revive`,
  `staticcheck`, `unused`, `whitespace`, `errorlint`) stays clean.
- Tests still pass.

Alternating `gostyle -fix` with `gofmt -w`, `gofumpt -w` and `goimports -w`
reaches a fixed point after one pass, so the tools do not undo each other.
This holds because the formatters never join lines inside parentheses and
gostyle only adds line breaks and trailing commas.

A linter that joins lines or caps line length, such as `lll`, would conflict
with the exemptions above. `golines` only splits lines, so it is safe.

## Development

```sh
go test ./...
golangci-lint run ./...
gofumpt -l .
gostyle .          # the tool follows its own rules
```

`main.go` is a thin entry point so that `go install` works from the module
root; everything else lives in the `src/` package.

| File | Contents |
|---|---|
| `main.go` | Calls `gostyle.Main()` |
| `src/run.go` | Flags and the command-line driver |
| `src/check.go` | Checker, violation and edit types, source checking |
| `src/signatures.go` | Parameter and return-value rules |
| `src/calls.go` | Argument count, nesting and width rules |
| `src/fix.go` | The fix loop |
| `src/gitdiff.go` | `-changed`: parsing `git diff-index` into line ranges |
| `src/files.go` | Path collection and the generated-file check |

`-changed` uses `git diff-index -p` (plumbing) with pinned prefixes, so user
settings such as `diff.noprefix` and `diff.mnemonicPrefix` cannot break the
path parsing.

## License

[MIT](LICENSE)
