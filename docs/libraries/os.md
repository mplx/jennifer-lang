# `os` - operating-system glue

Enable with `use os;`. Every name lives behind the `os.` prefix
(`os.PLATFORM`, `os.getEnv`). Nothing here is reachable as a bare
identifier.

```jennifer
use io;
use os;

io.printf("platform:     %s\n", os.PLATFORM);
io.printf("architecture: %s\n", os.ARCH);
io.printf("dir sep:      %s\n", os.DIRSEP);
io.printf("home:         %s\n", os.getEnv("HOME"));
io.printf("args:         %d arguments\n", len(os.ARGS));
```

The library splits cleanly: **immutable per-run host facts are
uppercase constants** (no arguments to take, no reason to be a
function); **operations that take arguments are functions**.

External-program execution (`os.run`, `os.spawn`, `os.wait`, etc.)
is deferred to a later sub-milestone since it needs a
library-provided struct mechanism the language doesn't yet have.

Process exit is the language statement `exit EXPR;`, not an `os`
function - see
[rejected.md > `os.exit(n)`](../technical/rejected.md#osexitn).

## Functions

| Call               | Returns | Notes                                                                               |
| ------------------ | ------- | ----------------------------------------------------------------------------------- |
| `os.getEnv(name)`  | string  | Reads an environment variable. Unset variables return `""`, no error.               |
| `os.hasFlag(name)` | bool    | True if `name` is an exact-match element of `os.ARGS`. See "Flag inspection" below. |
| `os.flag(name)`    | string  | The element immediately after `name` in `os.ARGS`, or `""` if absent or at end.     |

### Flag inspection

`os.hasFlag` and `os.flag` are convenience helpers for the most common
"did the user pass `--verbose`?" and "what value follows `--port`?"
patterns. They are **exact-match only**:

- `os.hasFlag("--port")` is true if `"--port"` appears as a standalone
  element of `os.ARGS`. It is **false** if `"--port=8080"` appears
  (different element value).
- `os.flag("--port")` returns the element immediately after `"--port"`
  if there is one, else `""`. The `--foo=bar` form is not parsed.

This is deliberately minimal. Real CLI parsing (combined short flags
like `-rf`, `--foo=bar`, repeated flags, subcommands) belongs to a
future `cli` library; if you need any of it now, walk `os.ARGS`
yourself.

```jennifer
use io;
use os;

if (os.hasFlag("--help")) {
    io.printf("Usage: %s [options]\n", os.ARGS[0]);
    exit 0;
}
def port as string init os.flag("--port");
if ($port == "") {
    $port = "8080";
}
io.printf("listening on %s\n", $port);
```

## Constants

### Host facts

| Name          | Kind   | Value                                                                                     |
| ------------- | ------ | ----------------------------------------------------------------------------------------- |
| `os.PLATFORM` | string | Operating-system name as reported by the runtime: `"linux"`, `"darwin"`, `"windows"`, ... |
| `os.ARCH`     | string | CPU architecture: `"amd64"`, `"arm64"`, `"wasm"`, ...                                     |
| `os.EOL`      | string | Platform line ending. `"\n"` on Unix-likes, `"\r\n"` on Windows.                          |
| `os.DIRSEP`   | string | Path-component separator: `"/"` on Unix-likes, `"\\"` on Windows.                         |
| `os.PATHSEP`  | string | PATH-list separator (between entries in `$PATH`): `":"` on Unix-likes, `";"` on Windows.  |

### Process

| Name      | Kind           | Value                                                                                              |
| --------- | -------------- | -------------------------------------------------------------------------------------------------- |
| `os.ARGS` | list of string | Command-line arguments passed to the running program. Index 0 is the script path, the rest follow. |

Interpreter-self-identity constants (`VERSION`, `BUILD`) live in
[`meta`](meta.md) since they describe the interpreter binary itself
rather than the host environment.

See also: [meta.md](meta.md), [../user-guide/index.md](../user-guide/index.md), [../user-guide/imports.md](../user-guide/imports.md), [../user-guide/style-guide.md](../user-guide/style-guide.md), [index.md](index.md).
