# `fs` - filesystem I/O

Enable with `use fs;`. Blocking whole-file reads and writes,
filesystem metadata, directory operations, and buffered file
handles for line-oriented reads. Non-blocking use composes with
M16.0 [`spawn`](../user-guide/concurrency.md) rather than
duplicating each call as a `*Async` variant.

```jennifer
use io;
use fs;

fs.writeString("hello.txt", "hi, world\n");
def content as string init fs.readString("hello.txt");
io.printf("%s", $content);
```

## One-shot operations

Whole-file reads and writes. Cheap to write, cheap to read.

| Call                                  | Returns  | Notes                                                                       |
| ------------------------------------- | -------- | --------------------------------------------------------------------------- |
| `fs.readString(path)`                 | `string` | Whole file as UTF-8. Invalid UTF-8 is a positioned runtime error.           |
| `fs.readBytes(path)`                  | `bytes`  | Whole file as raw bytes. See below for the handle form.                     |
| `fs.writeString(path, content)`       | `null`   | Overwrites. Creates the file if missing. See below for the handle form.     |
| `fs.writeBytes(path, content)`        | `null`   | Overwrites. Creates the file if missing. See below for the handle form.     |
| `fs.appendString(path, content)`      | `null`   | Appends. Creates the file if missing.                                       |
| `fs.appendBytes(path, content)`       | `null`   | Appends. Creates the file if missing.                                       |

Reading a file that doesn't exist is a positioned runtime error;
use `fs.exists(path)` first when the "missing file is normal"
case matters.

## Metadata

Boundary-friendly predicates plus a full `fs.stat`.

| Call                | Returns   | Notes                                                                        |
| ------------------- | --------- | ---------------------------------------------------------------------------- |
| `fs.exists(path)`   | `bool`    | True if the path resolves. Permission errors still surface.                  |
| `fs.isFile(path)`   | `bool`    | True iff the path exists and is a regular file. False for missing.           |
| `fs.isDir(path)`    | `bool`    | True iff the path exists and is a directory. False for missing.              |
| `fs.stat(path)`     | `fs.Stat` | Missing path errors. Pair with `fs.exists` when tolerating absent files.     |

### `fs.Stat`

```jennifer
def struct fs.Stat {
    path as string,        # the path as passed to fs.stat / fs.walk
    size as int,           # bytes; -1 for directories
    isDir as bool,
    mtimeNanos as int,     # Unix nanoseconds
    mode as int            # POSIX permission bits (e.g. 0o644, 0o755)
};
```

`mtimeNanos` is deliberately an `int`, not a `time.Time` - that
keeps `fs` decoupled from the `time` library at the Go-package
level. Composition is one line:

```jennifer
use fs;
use time;

def stat as fs.Stat init fs.stat("hello.txt");
def modified as time.Time init time.fromUnixNanos($stat.mtimeNanos);
io.printf("modified: %s\n", time.iso($modified));
```

`size` is `-1` for directories so callers don't accidentally
interpret it as "empty directory."

## Directory operations

The library ships **two verbs each** for create and delete
(`mkdir` / `mkdirAll`, `remove` / `removeAll`). The safe
non-recursive form keeps the short name; the recursive form
gets an explicit second name so a code review can grep for
the risky sites. This is Jennifer's "no footguns" stance
applied at the API level - a `bool` parameter or a mode
string would obscure the recursive intent at the call site.

| Call                    | Returns           | Notes                                                          |
| ----------------------- | ----------------- | -------------------------------------------------------------- |
| `fs.mkdir(path)`        | `null`            | Errors if any parent is missing (matches POSIX `mkdir`).       |
| `fs.mkdirAll(path)`     | `null`            | Creates every missing parent (matches POSIX `mkdir -p`).       |
| `fs.remove(path)`       | `null`            | Removes one file or one empty directory. Non-empty dir errors. |
| `fs.removeAll(path)`    | `null`            | Recursive delete. Explicit second verb.                        |
| `fs.rename(old, new)`   | `null`            | Same-filesystem rename; cross-fs is a boundary error.          |
| `fs.list(path)`         | `list of string`  | Sorted entry names. Non-recursive.                             |
| `fs.walk(path)`         | `list of fs.Stat` | Depth-first, sorted, includes `path` itself as the first entry. Skips symlinks. |

```jennifer
use fs;

# Safe: mkdir refuses to create with missing parents.
fs.mkdirAll("build/output/cache");        # explicit intent

# Safe: remove refuses to delete non-empty directories.
fs.removeAll("build/output/cache");       # explicit intent
```

## File handles

For line-oriented reads and files that don't fit comfortably in
memory, `fs.open` returns an `fs.File` handle backed by the
integer-registry pattern also used by `hash.Stream` and
`crc.Stream`.

```jennifer
def struct fs.File { id as int };
```

Mode strings use the codec-table shape:

| Mode        | Semantics                                                          |
| ----------- | ------------------------------------------------------------------ |
| `"read"`    | Read-only. `fs.readLine`, `fs.readChars`, `fs.readBytes` allowed. |
| `"write"`   | Write, create+truncate. `fs.writeString`, `fs.writeBytes` allowed. |
| `"append"`  | Write, create+append. Same write ops as `"write"`.                 |

Unknown mode strings error with the known set listed.

### Handle surface

| Call                       | Returns    | Notes                                                                     |
| -------------------------- | ---------- | ------------------------------------------------------------------------- |
| `fs.open(path, mode)`      | `fs.File`  | Opens per the mode string.                                                |
| `fs.close($f)`             | `null`     | Removes the handle from the registry; later ops on the id error.         |
| `fs.readLine($f)`          | `string`   | One line; `\r\n` / `\n` stripped. Errors on EOF - check `fs.eof` first.  |
| `fs.readChars($f, n)`      | `string`   | Up to `n` runes, UTF-8 decoded. Partial result on EOF, sticky-EOF flip.  |
| `fs.readBytes($f, n)`      | `bytes`    | Up to `n` bytes. Partial result on EOF, sticky-EOF flip.                 |
| `fs.writeString($f, s)`    | `null`     | Read-mode handle errors.                                                  |
| `fs.writeBytes($f, b)`     | `null`     | Read-mode handle errors.                                                  |
| `fs.eof($f)`               | `bool`     | Looks ahead: true iff the next read would error / return partial.        |

The canonical read-loop:

```jennifer
use io;
use fs;

def f as fs.File init fs.open("input.txt", "read");
while (not fs.eof($f)) {
    def line as string init fs.readLine($f);
    io.printf("[%s]\n", $line);
}
fs.close($f);
```

`fs.readLine` on a handle whose next read is EOF errors with
`end of input`; the `fs.eof($f)` guard is what keeps the loop
tight. `fs.eof` looks one byte ahead (through the buffered
reader) so a file ending cleanly with `\n` still trips the
guard after the last line comes out.

### Polymorphic verbs (path vs handle)

Three verbs are polymorphic on the first argument's kind:

| Verb              | 1-arg form        | 2-arg form                    |
| ----------------- | ----------------- | ----------------------------- |
| `fs.readBytes`    | `(path)` -> whole | `($f, n)` -> partial handle read |
| `fs.writeString`  | -                 | `(path, content)` or `($f, s)` |
| `fs.writeBytes`   | -                 | `(path, content)` or `($f, b)` |

The dispatcher picks based on whether the first argument is a
`string` (path form) or an `fs.File` (handle form). Any other
kind is a positioned boundary error.

### Handles share state between copies

An `fs.File{id}` value is small; copies share the underlying
Go `*os.File` state via the integer id. This mirrors M16.0's
`task of T` carve-out to the "value semantics everywhere"
rule:

```jennifer
def a as fs.File init fs.open("x.txt", "read");
def b as fs.File init $a;             # `b` and `a` reference the same file
fs.close($a);                          # closes for both
def s as string init fs.readLine($b);  # errors: id is not open
```

Handles are the second "handles wrap shared state" carve-out
in the language, sitting alongside `task of T`. Every other
type keeps whole-value semantics.

## Concurrency composition (M16.0)

`fs` is blocking on purpose. Non-blocking use is a one-line
composition with `spawn`:

```jennifer
use fs;
use task;

def t as task of string init spawn {
    return fs.readString("/etc/hosts");
};
def content as string init task.wait($t);
```

Multiple files in parallel:

```jennifer
use fs;
use task;

func loadOne(path as string) {
    return fs.readString($path);
}

def paths as list of string init ["a.txt", "b.txt", "c.txt"];
def tasks as list of task of string init [];
for (def p in $paths) {
    $tasks[] = spawn { return loadOne($p); };
}
def contents as list of string init task.waitAll($tasks);
```

Under the default `jennifer` binary this actually parallelises across
cores; under `jennifer-tiny` (TinyGo, cooperative single-threaded
0.41) the composition is correct but sequential. See
[../technical/tinygo.md](../technical/tinygo.md).

## Errors

Every error is positioned at the Jennifer call site with the
path or handle id in the message.

- **Missing files** on `fs.readString` / `fs.readBytes` /
  `fs.stat`: `fs.readString: PATH: open PATH: no such file or directory`.
- **Non-empty directory** given to `fs.remove`:
  `fs.remove: PATH: directory not empty`. Use `fs.removeAll` for
  recursive delete.
- **Missing parent** given to `fs.mkdir`:
  `fs.mkdir: PATH: no such file or directory`. Use `fs.mkdirAll`
  for `mkdir -p`.
- **Wrong mode for op**: `fs.readLine: fs.File "x.txt" was opened
  in mode "write"; open with mode "read" to read`. Same message
  shape in reverse for read-mode handles given a write op.
- **Unknown mode string**: `fs.open: unknown mode "rw"; known:
  "read", "write", "append"`.
- **Use after close**: `fs.readLine: fs.File id 3 is not open
  (already closed, or never opened)`.
- **Non-negative int required**: `fs.readChars: n must be
  non-negative, got -1`.

Every error is catchable with M13.2 `try` / `catch`:

```jennifer
try {
    def s as string init fs.readString("optional-config.toml");
} catch (e) {
    io.printf("no config, using defaults: %s\n", $e.message);
}
```

## What's not in v1

Recorded so the design decisions stay visible; ships if a
concrete workload forces it.

- **Streaming line iterator** (`for (def line in fs.lines(path))`).
  Compose with `fs.open` + `while (not fs.eof)` today.
- **`fs.copy(src, dst)`** and **`fs.chmod(path, mode)`**.
- **Symlink ops** (`fs.readlink`, `fs.symlink`).
- **`fs.stat($f)` on an open handle**. Only path-based `fs.stat`
  in v1.
- **Watch / notify** (inotify / kqueue / FSEvents).
- **Temp file / dir creation helpers** (`fs.tempFile`,
  `fs.tempDir`). Use `os.getEnv("TMPDIR")` plus your own naming.
- **Follow symlinks in `fs.walk`**. Symlink-loop protection is
  the reason for the deferral; a `follow=true` flag or a
  distinct `fs.walkFollowing` verb ships once the story is clear.

## See also

- [../user-guide/concurrency.md](../user-guide/concurrency.md) -
  the `spawn`-and-compose story `fs` builds on.
- [`task`](task.md) - observe handles produced by `spawn`.
- [`time`](time.md) - `time.fromUnixNanos` converts
  `fs.Stat.mtimeNanos` into a `time.Time`.
- [`os`](os.md) - process-level operations that pair with `fs`:
  env vars, argv, external commands.
- [../milestones.md](../milestones.md) - M16.1 design spec.
