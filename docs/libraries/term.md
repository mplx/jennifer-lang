# `term` - terminal control

Enable with `use term;`. The terminal host capabilities an interactive TUI needs
and pure `.j` cannot reach: **raw mode** (unbuffered, no-echo input), the
**terminal size**, and **raw single-byte reads** from stdin. This is the low
level - screen control, key decoding, and rendering belong in a `.j` layer on
top; `term` is only the primitives that require the host.

An **output-only** TUI (a dashboard, a progress bar) needs none of this - just
[`ansi`](../modules/ansi.md) escape codes and [`os.isTerminal`](os.md). Reach
for `term` when you need to read keypresses as they happen.

`term` is a **system library** built on `golang.org/x/term` (the package the
REPL's own line editor uses). Build-tag split like [`net`](net.md): the default
`jennifer` binary has the real implementation; `jennifer-tiny` returns a friendly
error (a minimal / embedded target may have no controlling terminal).

## Surface

| Call | Returns | Notes |
| ---- | ------- | ----- |
| `term.makeRaw(stream)` | `term.State` | Put a terminal (`"stdin"`) into raw mode; returns a handle for `restore`. |
| `term.restore(state)` | `null` | Undo `makeRaw`, restoring the terminal. The handle is single-use. |
| `term.size(stream)` | `term.Size` | The terminal's `{rows, cols}` (query `"stdout"`). |
| `term.readByte()` | `int` | The next raw byte from stdin (`0`-`255`), or `-1` at end of input. |

`stream` is `"stdin"` / `"stdout"` / `"stderr"` (the same names as
`os.isTerminal`). A stream that is not a terminal - a pipe or a redirect - is a
catchable error for `makeRaw` / `size`.

## Raw mode

In the terminal's default (cooked) mode the OS buffers a whole line, echoes it,
and only hands it over on Enter. **Raw mode** turns that off: every keypress is
delivered immediately, unechoed - which is what an editor, a pager, or a game
loop needs. `term.makeRaw` enters it and returns a `term.State` handle; pass that
handle to `term.restore` to put the terminal back:

```jennifer
use io;
use os;
use term;

if (not os.isTerminal("stdin")) {
    io.printf("needs an interactive terminal\n");
} else {
    def state as term.State init term.makeRaw("stdin");
    def running as bool init true;
    while ($running) {
        def b as int init term.readByte();       # blocks until a key is pressed
        if ($b == -1 or $b == 113) {             # -1 = end of input, 113 = 'q'
            $running = false;
        } else {
            io.printf("byte %d\r\n", $b);         # raw mode: newline is not cooked
        }
    }
    term.restore($state);
}
```

**Always restore.** Raw mode is a property of the terminal device, not the
process, so a program that exits without calling `term.restore` leaves the shell
in raw mode. Jennifer has no `finally`, so put the `restore` on every path out -
and note that in raw mode the newline is no longer cooked, so prints need an
explicit `\r\n`. The `term.State` handle is **single-use**: a second
`term.restore` of the same handle is an error, so a live terminal is never
clobbered by a stale handle.

## Reading keys

`term.readByte` returns one raw byte from stdin (`0`-`255`), or `-1` at end of
input. In raw mode it returns the instant a key is pressed. It reads **bytes**,
not decoded keys: a plain key is one byte, but an arrow key or a function key
arrives as a multi-byte escape sequence (`ESC [ A` for Up, ...). Decoding those
into key events is the job of a TUI layer on top - `term` deliberately stops at
the raw byte.

`term.readByte` is refused inside the REPL, which owns the terminal for its own
line editor.

## Terminal size

`term.size(stream)` returns a `term.Size{rows, cols}` - the character grid of the
terminal behind `stream` (usually `"stdout"`):

```jennifer
def dim as term.Size init term.size("stdout");
io.printf("%d rows x %d cols\n", $dim.rows, $dim.cols);
```

The size is a snapshot; a terminal can be resized after the call, so re-query it
when you need the current value.

## See also

- [`os`](os.md) - `os.isTerminal(stream)` to gate on an interactive terminal
  before entering raw mode.
- [`ansi`](../modules/ansi.md) - escape codes for colour and cursor control,
  which pair with `term` for a full-screen TUI.
- [milestones.md](../milestones.md) - the `term` design and the `screen` / `tui`
  module planned on top of it.
