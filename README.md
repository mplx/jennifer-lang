# Jennifer Programming Language

**Milestone 6**

Jennifer is a small, experimental, interpreted programming language. The
interpreter is written in Go and the shipping binary is produced with
[TinyGo](https://tinygo.org/). Source files use the `.j` extension.

This project exists primarily as a learning exercise: how to design a
language and build an interpreter end-to-end (lexer → preprocessor → parser →
tree-walking evaluator → stdlib).

Jennifer currently targets **Linux**; Windows and macOS support is planned.

## Stability

Jennifer is **pre-1.0**. While the major version stays at `0.x.y`,
**anything can change at any time** - syntax, semantics, library names,
function signatures, file formats. We aim for best-effort stability
between minor versions but make no guarantees: a milestone may rename a
keyword, retype a builtin, or restructure the standard library when a
better design is found. Pin to a specific version if you need
reproducibility; expect to migrate when you upgrade.

Starting with **`1.0.0`**, Jennifer will follow [Semantic
Versioning](https://semver.org/): breaking changes only on a major
version bump, additive features on minor, fixes on patch.

## Quick start

```sh
# Build
tinygo build -o jennifer ./cmd/jennifer

# Run a program
./jennifer run examples/hello.j     # prints "42"
```

A first program:

```jennifer
use io;

def x as int init 21;
printf($x + $x);
```

## Documentation

- [docs/user-guide/](docs/user-guide/index.md) - language tutorial and
  reference split by topic: installing, first program, syntax, types
  and values, methods, control flow, imports, examples.
- [docs/technical/](docs/technical/index.md) - interpreter internals split
  by topic: lexer, grammar/parser, preprocessor, interpreter, CLI, testing,
  file map, rejected features, TinyGo notes.
- [docs/milestones.md](docs/milestones.md) - what's implemented, what's
  coming, and the rationale behind the order.

## Testing

```sh
go test ./...
```

Development tests run under regular Go because TinyGo's test runner is
limited; the shipping binary is built with TinyGo.

## License

LGPL-3.0-only. See [LICENSE.md](LICENSE.md).
