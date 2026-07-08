# CLI (`cmd/jennifer`)

```
jennifer run <file.j>      run a Jennifer program
jennifer run -             read source from stdin
jennifer repl              interactive REPL
jennifer tokens <file.j>   dump the lexer's token stream
jennifer ast <file.j>      dump the preprocessed AST as JSON
jennifer fmt <file.j>      format source per docs/user-guide/style-guide.md
jennifer lint <file.j>     report compile-legal but suspect patterns
jennifer profile <file.j>  profile hit counts and wall-clock per source position
jennifer test <file.j>     discover and run the file's test methods
jennifer version           print the build version and exit
jennifer help              show usage
```

- Verifies the `.j` extension
- Reads the file, parses, runs
- On error: prints the message and a source-context caret on stderr, exits `1`
- Bad usage exits `2`
- `jennifer help` includes a `Version:` line so the build is identifiable at a glance
- `tokens`, `ast`, `fmt`, `lint`, `profile`, and `test` are development
  subcommands, present only in the default `jennifer` binary (the run-only
  `jennifer-tiny` build stubs them); each has its own page below.


## Subcommand reference

Each subcommand has its own page:

- [REPL](cli_repl.md) - the interactive `repl`, its line editor, and history.
- [Inspection](cli_inspect.md) - `tokens` and `ast` dumps.
- [Formatter](cli_fmt.md) - `fmt`, token-level normalisation.
- [Linter](cli_lint.md) - `lint`, the check catalog, suppression, config.
- [Profiler](cli_profile.md) - `profile`, hit counts and timings.
- [Test runner](cli_test.md) - `test`, discovery and reporting.

## Version injection

`internal/version.Version` holds the build version as a single `string`.
The default is `"dev"`; the `Makefile` runs `scripts/gen-version.sh` before
each build, which writes `internal/version/version_gen.go` containing a
small `init()` that overwrites `Version` with the output of
`scripts/version.sh` (a `git describe --tags --long` derivative; see
[../libraries/meta.md](../libraries/meta.md) for the format).

This codegen path replaces the more conventional `go build -ldflags
"-X .Version=..."` because TinyGo 0.41 silently ignores the `-X`
directive. Codegen works identically on both toolchains. The generated
file is `.gitignore`d so the repository never carries a stale copy.

Two consumers read `version.Version`:

- `cmd/jennifer/main.go` prints it in the `help` banner and as the body
  of the `version` subcommand.
- `internal/lib/meta/metalib.go` mirrors it into the interpreter as the
  `meta.VERSION` constant. The `meta` library is opt-in like every
  other library: `use meta; io.printf("%s\n", meta.VERSION);`.

`go test ./...` skips codegen and uses the default `"dev"`. The meta-lib
test only checks that the constant matches `version.Version`, not a
specific value, so it stays robust across builds.
