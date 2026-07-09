# REPL (`cmd/jennifer/repl.go`)

The REPL drives a read-eval-print loop on top of the standard pipeline. Each
input is lexed, preprocessed, parsed, and fed to `Interpreter.EvalInteractive`
(not `Run`). `EvalInteractive` differs from `Run` in three documented ways:
the global env is lazy-initialized and preserved across calls, library
imports and method definitions are idempotent / re-assignable so the user
can iterate, and the value of a trailing `ExprStmt` is returned so the loop
can print it.

## Echoing a value

Because `EvalInteractive` returns the trailing `ExprStmt`'s value, you can
inspect any variable by typing **its bare reference followed by `;`** - the
REPL prints the value's `Display()` form:

```
>>> def x as int init 41;
>>> $x;
41
>>> def doc as json.Value init json.decode("{\"a\":[1,2]}");
>>> $doc;
{"a":[1,2]}
```

This is a REPL-only convenience: `Run` (the batch path) evaluates an
expression statement but discards its value, so a bare `$x;` in a `.j`
**script prints nothing**. To show a value from a script, format it
explicitly - `io.printf("%v\n", $x);` (or `io.sprintf($x)` /
`convert.toString($x)`). Opaque values render through their registered
displayer, so `$doc;` shows a `json.Value` as its JSON rather than
`<json.Value>`.

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

## Line editor (`cmd/jennifer/lineedit.go`, `cmd/jennifer/history.go`)

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


Part of the [CLI reference](cli.md).
