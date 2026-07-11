# CLI (`cmd/jennifer`)

```
jennifer run [flags] <file.j> [args...]  run a Jennifer program
jennifer run -             read source from stdin
jennifer repl              interactive REPL
jennifer tokens <file.j>   dump the lexer's token stream
jennifer ast <file.j>      dump the preprocessed AST as JSON
jennifer fmt <file.j>      format source per docs/user-guide/style-guide.md
jennifer lint <file.j>     report compile-legal but suspect patterns
jennifer profile <file.j>  profile hit counts and wall-clock per source position
jennifer test <file.j>     discover and run the file's test methods
jennifer version [-v]      print the build version (-v adds module-path layers)
jennifer help              show usage
```

## Module resolution flags (`run`)

`jennifer run` accepts interpreter flags before the source file; anything
after the file is the program's own `os.ARGS`.

- `--sysmoddir DIR` (or `--sysmoddir=DIR`) - the system module directory
  for bare `import "name.j";`. Overrides `JENNIFER_SYSMODDIR`, which
  overrides the compile-time default. A named (CLI or env) dir that is
  missing or not a directory refuses to start; the compile-time default is
  best-effort. The resolved value is `meta.SYSMODDIR`.
- `-I DIR` (or `-I=DIR`, repeatable) - add a directory to the module
  search path after the system dir. A `-I` dir *adds* names; a module name
  appearing in two search dirs is a hard error at load. (Resolution lives
  in `internal/module`; the `import` statement consumes it via the loader
  wired in `main.go`'s `runFile` - `in.EnableModules(baseDir, searchDirs,
  loadModuleProgram, installLibraries)`, where `searchDirs` is the system
  dir followed by each `-I` dir.)

`jennifer version -v` reports the resolved system module dir and the
layers (compile default, `JENNIFER_SYSMODDIR`) behind it.

- Verifies the `.j` extension
- Reads the file, parses, runs
- On error: prints the message and a source-context caret on stderr, exits `1`
- Bad usage exits `2`
- `jennifer help` includes a `Version:` line so the build is identifiable at a glance
- `tokens`, `ast`, `fmt`, `lint`, `profile`, and `test` are development
  subcommands, present only in the default `jennifer` binary (the run-only
  `jennifer-tiny` build stubs them); each has its own page below.


## Shell pipelines and aliases

Because `jennifer run -` reads a program from stdin (the `run -` form
above), Jennifer drops into a shell pipeline like any other filter. One
caveat sets the shape: stdin can carry *either* the program *or* the data,
not both. So a one-liner alias pipes the program in and passes the data as
an argument (read back through `os.ARGS`); a reusable filter keeps the
program in a file and leaves stdin free for the data.

**Inline, program piped via `run -`.** A `json-pretty` that reformats a
single JSON argument. The program arrives on stdin, so the JSON is
`os.ARGS[1]` (`ARGS[0]` is the `-`):

```sh
alias json-pretty="printf '%s' 'use json; use os; use io; io.printf(\"%s\\n\", json.encodePretty(json.decode(os.ARGS[1])));' | jennifer run -"

json-pretty '{"b":2,"a":1}'
# {
#   "b": 2,
#   "a": 1
# }
```

Use `printf '%s'` rather than `echo` to pass the program verbatim, so the
`\n` reaches Jennifer as a two-character escape instead of being expanded
by the shell.

**Reusable filter, data on stdin.** For a true pipe (`... | json-pretty`,
`json-pretty < file.json`) keep the program in a file and let stdin carry
the data. Save this as, say, `~/.local/share/jennifer/json-pretty.j`:

```jennifer
use json;
use io;

def src as string init "";
while (not io.eof()) {
    $src = $src + io.readLine() + "\n";
}
io.printf("%s\n", json.encodePretty(json.decode($src)));
```

```sh
alias json-pretty='jennifer run ~/.local/share/jennifer/json-pretty.j'

echo '{"b":2,"a":1}' | json-pretty
curl -s https://api.example.com/thing | json-pretty
```

The same shape extends to any decode / re-encode pair: swap `json` for
`xml` once that library lands ([M20.2](../milestones.md#m202---xml)) and the
file becomes a `pretty-xml`.


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
