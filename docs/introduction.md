# Jennifer

Jennifer is a small, experimental, interpreted programming language
built as a learning exercise. The interpreter is written in Go and
ships as two binaries:

- **`jennifer`** - TinyGo build, small and embeddable. The canonical
  shipping binary. Some host features (like `os/exec`) are
  unsupported on TinyGo today and return a friendly limitation
  error.
- **`jennifer-go`** - standard Go build, full host-feature surface.
  Same source, same language; the right pick when you want the
  performance variant for compute-bound code.

Source files use the `.j` extension. Whitespace is not significant
anywhere; statements end with `;`.

```jennifer
use io;
use time;

def start as time.Time init time.now();
io.printf("hello, world\n");
def gap as time.Duration init time.sub(time.now(), $start);
io.printf("ran for %d ms\n", time.milliseconds($gap));
```

## What's in this site

- **[Getting started](user-guide/installing.md)** - install the
  interpreter and run your first program.
- **[Language reference](user-guide/index.md)** - syntax, types,
  methods, control flow, imports, style.
- **[Libraries](libraries/index.md)** - per-library reference plus
  an alphabetical [cheatsheet](libraries/cheatsheet.md) of every
  builtin.
- **[Technical reference](technical/index.md)** - implementation
  details for the lexer, parser, interpreter, and CLI.
- **[Project](milestones.md)** - milestones, design stances,
  glossary.

## Status

Pre-1.0. The major version stays at `0.x.y` while the language is
still finding its shape; breaking changes can happen at milestone
boundaries and are called out in
[docs/milestones.md](milestones.md). Once Jennifer reaches `1.0.0`,
semver applies and breaking changes need a major bump.

## Source

Source, issues, and pull requests live at
<https://github.com/mplx/jennifer-lang>. License: LGPL-3.0-only.
