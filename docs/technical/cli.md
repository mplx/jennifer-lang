# CLI (`cmd/jennifer`)

```
jennifer run <file.j>     run a Jennifer program
jennifer run -            read source from stdin
jennifer repl             interactive REPL
jennifer tokens <file.j>  dump the lexer's token stream
jennifer ast <file.j>     dump the preprocessed AST as JSON
jennifer fmt <file.j>     format source per docs/user-guide/style-guide.md
jennifer version          print the build version and exit
jennifer help             show usage
```

- Verifies the `.j` extension
- Reads the file, parses, runs
- On error: prints the message and a source-context caret on stderr, exits `1`
- Bad usage exits `2`
- `jennifer help` includes a `Version:` line so the build is identifiable at a glance

## REPL (`cmd/jennifer/repl.go`)

The REPL drives a read-eval-print loop on top of the standard pipeline. Each
input is lexed, preprocessed, parsed, and fed to `Interpreter.EvalInteractive`
(not `Run`). `EvalInteractive` differs from `Run` in three documented ways:
the global env is lazy-initialized and preserved across calls, library
imports and method definitions are idempotent / re-assignable so the user
can iterate, and the value of a trailing `ExprStmt` is returned so the loop
can print it.

Multi-line input is handled by a small `inputComplete(tokens)` helper that
balances `{`/`(` against `}`/`)` (using the lexer's tokens so string and
comment contents are ignored) and requires the input to end in `;` or `}`.
Anything else triggers a `... ` continuation prompt. Unbalanced *closing*
delimiters intentionally fall through to the parser for diagnosis since no
amount of additional input would fix them.

REPL input is tagged with the synthetic file label `<repl>`. The
cross-file-error snippet loader in `printErrorContext` treats `<repl>` like
`<stdin>`: no external file lookup is attempted, and the current input
buffer is used as the snippet source. Lex errors discard the buffer (since
they cannot become valid by reading more); parse and runtime errors print
and the loop continues.

`:quit` / `:exit` / EOF terminate cleanly; `:help` prints a short reminder.
Directives are only recognized at a fresh prompt so a literal `:quit` inside
a block doesn't short-circuit.

### Line editor (`cmd/jennifer/lineedit.go`, `cmd/jennifer/history.go`)

When stdin is a terminal the REPL installs raw mode via
`golang.org/x/term` and reads lines through a small built-in editor.
The editor is a single state machine over a rune buffer plus a cursor
index; each keystroke updates the state and triggers a `redraw()`.

Supported input:

| Key                          | Action                                                                       |
| ---------------------------- | ---------------------------------------------------------------------------- |
| Printable rune               | Insert at cursor                                                             |
| Backspace, Ctrl+H            | Delete char before cursor                                                    |
| Delete (CSI `3~`)            | Delete char at cursor                                                        |
| Left / Right (CSI `D` / `C`) | Move cursor by one char                                                      |
| Home / End (CSI `H` / `F`)   | Jump to line start / end                                                     |
| Ctrl+A / Ctrl+E              | Same as Home / End                                                           |
| Ctrl+Left / Ctrl+Right       | Move by word                                                                 |
| Alt+B / Alt+F                | Same as Ctrl+Left / Ctrl+Right (macOS terminals send these for option-arrow) |
| Ctrl+W, Ctrl+Backspace       | Delete word backward                                                         |
| Ctrl+U                       | Kill from line start to cursor                                               |
| Ctrl+K                       | Kill from cursor to line end                                                 |
| Up / Down                    | History navigation                                                           |
| Ctrl+C                       | Cancel current line (fresh prompt)                                           |
| Ctrl+D on empty buffer       | EOF (exits the REPL)                                                         |
| Ctrl+D on non-empty buffer   | Forward-delete                                                               |

Word boundaries use a small punctuation + whitespace ruleset that's
predictable for source-code editing without needing a full Unicode
word-break implementation. History is an in-memory ring (`replHistory`,
100 entries by default, adjacent duplicates collapsed); on-disk
persistence is a future enhancement.

Non-TTY stdin falls back to the original `bufio` line reader. This
keeps `echo ... | jennifer repl` and integration tests working
unchanged - the editor would do nothing useful on a non-interactive
stream anyway.

Raw mode disables the kernel's `OPOST` flag, so `\n` written to stdout
no longer auto-translates to `\r\n`. The REPL works around this with
a tiny `crlfWriter` wrapper that performs the translation in user
space for error/help/result prints. Cooked-mode output (the banner
printed before raw mode is entered, and anything after raw mode is
restored) goes to `os.Stderr` / `os.Stdout` directly.

The editor only handles single-line editing. Multi-line input via the
continuation prompt (`... `) is still driven by the surrounding REPL
loop's `inputComplete()` check, so unclosed `{` / `(` accumulate
across calls to `editor.readLine`.

## Inspection: `tokens` and `ast`

`cmd/jennifer/dump.go` and `cmd/jennifer/astjson.go` implement two
read-only inspection subcommands. `tokens` runs only the lexer and
prints one token per line in column-aligned `LINE:COL TYPE [lexeme]`
form - useful for tracing scanning issues and as a teaching tool.
`ast` runs lex + preproc + parse and writes the resulting AST as
two-space-indented JSON; every node carries `type`, `file`, `line`,
`col`, plus its node-specific fields.

The JSON emitter is hand-rolled in `astjson.go`'s `emitNode` (a switch
over every concrete AST type). We avoid `encoding/json` because its
reflect-based marshaling is fragile under TinyGo and at odds with the
tagged-union `Value` discipline used elsewhere; a switch over ~20 node
kinds is small enough to keep readable. Each field-emitter
(`emitStringField`, `emitBoolField`, `emitNodeListField`, etc.) writes
`"key": value,` and the closing `endObj` trims the trailing comma so
the output is valid JSON.

## Formatter (`cmd/jennifer/fmt.go`)

`jennifer fmt` formats source per [../user-guide/style-guide.md](../user-guide/style-guide.md).
It operates on the lexer's token stream rather than the AST, for two
reasons:

1. **`import "file.j";` survives.** The preprocessor consumes file
   imports before the parser sees them; an AST-based formatter would
   inline every import, which is the opposite of what a developer
   wants from `fmt`. The token-level formatter sees IMPORT tokens
   unchanged and re-emits them.
2. **User-written parens survive.** The AST records grouping only
   through nesting structure; redundant parens are erased. A token
   walker preserves LPAREN/RPAREN exactly as written, so
   `($a + $b) * $c` stays parenthesized after a round trip.

`formatTokens(tokens)` drives a small state machine (`fmtState`): for
each token it computes the separator (`writeSeparator`) - none, a
space, or a newline-plus-indent - and then writes the token's canonical
spelling (`writeToken`). Key state fields:

- `indent` bumps on `{` and drops on `}` (the closing brace dedents
  *before* it's written so it lands at the outer indent).
- `prevIsOperand` answers "is the next `-` binary or unary?" - flipped
  by `isOperandToken` after every emit.
- `prevIsUnaryMinus` suppresses the right-side space after a `-` that
  was determined to be unary.
- `insideForHeader` is a small backward scan that lets the two `;`s
  inside `for (...; ...; ...)` stay on the same line.

Strings are re-quoted with `quoteJenniferString` (double quotes plus
standard escapes), mirroring the lexer's `readString` on the way in.

Comments and blank lines survive a `fmt` round-trip: the lexer
emits them as trivia tokens, and `emitTrivia` writes them inline
without disturbing the surrounding state machine. Leading
comments land on their own line at the current indent; trailing
same-line comments stay on the same line; runs of blank lines
collapse to one. Block comments may nest.

## Linter (`cmd/jennifer/lint.go`, `internal/lint`)

`jennifer lint <file.j>` reports patterns that are compile-legal but
stylistically or semantically suspect - the slot between `fmt` (which
normalises lexical shape) and the parser (which rejects the outright
illegal). The checks live in `internal/lint`; the subcommand in
`cmd/jennifer/lint.go` wraps them with file I/O, config resolution, and
output rendering.

The check set, each with a stable ID so suppression and configuration
stay portable and greppable:

| ID     | Check                       | Severity | Flags                                                              |
| ------ | --------------------------- | -------- | ------------------------------------------------------------------ |
| `L001` | unused-local                | warning  | a local `def` binding never read (skips spawn-body declarations)   |
| `L002` | dead-code-after-terminator  | warning  | a statement after `return`/`throw`/`exit`/`break`/`continue`       |
| `L003` | empty-catch                 | warning  | a `catch` block with no body                                       |
| `L004` | throw-non-error             | warning  | a `throw` whose value isn't statically an `Error`                  |
| `L005` | method-too-long             | info     | method body over the statement threshold (default 60)              |
| `L006` | nesting-too-deep            | info     | block nesting over the depth threshold (default 4)                 |
| `L007` | constant-condition          | warning  | `if (true)`, `while (true)` with no escape, `if ($x == $x)`, ...   |
| `L008` | deprecation                 | warning  | reserved family, empty until an API is deprecated                  |
| `L009` | removed-api                 | warning  | use of a removed API (e.g. `use core;`)                            |
| `L010` | line-too-long               | info     | a source line over the column limit (default 100)                  |

**Traversal.** The parser exposes no generic visitor, so `internal/lint`
carries two: a flat `walker` (`walk.go`) with list/stmt/expr hooks for
checks that match node shapes (L002/L003/L005/L006/L007), and a
scope-aware traversal (`scope.go`) mirroring the resolver's frame model
for the checks that need binding visibility (L001/L004). Both descend
into `SpawnExpr.Body`, which the resolver deliberately skips: a read
inside a spawn still marks an outer local used, but a *declaration*
inside a spawn is left unreported (the resolver's spawn carve-out, which
the linter inherits). The linter runs on the parsed AST alone - it does
not call `parser.Resolve`, so it can lint code that would fail
resolution, and it tracks its own bindings and declared types.

**Severity and exit code.** A finding at or above `SeverityFloor`
(warning) makes the run exit 1; an info-only run exits 0; a linter
failure (bad flags, IO, parse error, bad config, unknown ID in a
directive) exits 2. Same triaging shape as `gofmt -l` / `shellcheck`.

**Suppression.** `# lint-disable: L003` (trailing) silences an ID on
that line; `# lint-disable-file: L005, L006` silences file-wide. There
is no blanket disable-all - a directive names IDs. Because the parser
strips comments, `applySuppressions` reads directives off the raw
`lexer.TokenizeWithFile` stream and correlates them to findings by
`file`/`line`; unknown IDs raise a positioned error.

**Configuration.** `--checks=IDS` (per run) or a `.jennifer-lint` file at
the tree root (per project) select checks with one `IDS` / `!IDS`
direction - all includes ("run only these") or all excludes ("run
everything except"); mixing is an error. Unknown IDs are always an error,
everywhere. `--format=human|json|github` picks the output shape:
positioned carets (reusing `printErrorContextTo`), a JSON array of
`{id,file,line,col,message,severity}` objects (valid JSON, `[]` when
empty), or GitHub Actions annotations.

**TinyGo.** The subcommand is build-tag split: `lint.go` (`!tinygo`)
carries the real implementation and is the only importer of
`internal/lint`, so the whole AST-walking machinery stays out of the
`jennifer-tiny` binary; `lint_tinygo.go` (`tinygo`) is a friendly stub
pointing at the default `jennifer` binary, mirroring the `os.run` / `net`
pattern.

## Profiler (`cmd/jennifer/profile.go`, `internal/profile`)

`jennifer profile <prog.j>` runs the program with the evaluator
instrumented and attributes work back to Jennifer source positions
(file:line:col) - the gap `go tool pprof` leaves, since it profiles the
interpreter binary, not the .j program inside it. The program's own
output is redirected to stderr so the profile owns stdout cleanly, even
in the binary pprof form: `jennifer profile --format=pprof p.j > p.pb.gz`.

**Instrumentation.** The interpreter carries an optional `Profiler`
interface (`internal/interpreter/profiling.go`) and three gate flags;
nil means no profiling, the only hot-path cost being a nil check. The
concrete collector lives in `internal/profile` and is injected only by
this subcommand, so no profiling machinery compiles into either binary's
run path. Hook points:

- **`execStmt`** wraps `execStmtRaw`, timing each statement. A
  `profChild` accumulator splits *self* time (this statement) from
  *cumulative* time (this statement plus everything it called), the
  standard nested-timing subtraction.
- **`evalCall`** times each method-call body for the trace timeline.
- **`ensureCOW`** (replacing the bare `Value.Ensure()` at the four
  mutation sites) records a COW detachment when a shared backing is
  actually copied. Because `Value` and `Interpreter` share a package,
  the interpreter reads the shared marker directly.
- **`evalSpawn`** times the `snapshotForSpawn` deep copy.

`evalExpr` is deliberately *not* timed: a `time.Now()` around every
literal read would swamp the profile with its own overhead.

**Modes.** The default statement profile records per-position hit counts
and self/cumulative time. `--allocs` switches to the value-semantics
profile, which surfaces three copy paths per source position:

- **Eager copies** - a `def` / assignment / parameter binding that
  deep-copies a compound value (`execDefine` / `execAssign` /
  `bindParamValue` call `Value.Copy()`). This is where the real
  allocation cost lives.
- **COW detachments** - an `Ensure()` that copied a shared backing at a
  mutation site. Because the interpreter copies eagerly at every store
  and keeps the append/index hot loop unshared by design, a mutation
  target almost never holds a shared value, so these stay at or near
  zero for ordinary `.j` code - `Ensure`'s detach branch is effectively
  reachable only from the Go-level value API. The counter is kept for
  correctness (if a future storage path defers its copy, it shows up
  here).
- **Spawn-frame deep copies** - the scope snapshot taken when a `spawn`
  launches (`snapshotForSpawn`).

`examples/profile.j` exercises all three. See it for the eager-vs-COW
contrast in practice.

**Formats.** `--format=table` (default, human-readable), `--format=pprof`
(gzipped protobuf, hand-encoded in `pprof.go` to keep the zero-dependency
stance - `go tool pprof` and speedscope.app read it), `--format=trace`
(Chrome-trace JSON of the call timeline). Unknown `--format` and the
unsupported `--allocs --format=trace` combination (allocation events
have no timeline) are rejected at argv parse, not deferred to output.

**TinyGo.** Build-tag split like the linter: `profile.go` (`!tinygo`) is
the only importer of `internal/profile`; `devtools_tinygo.go` stubs the
subcommand in the run-only `jennifer-tiny` binary.

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
