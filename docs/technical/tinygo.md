# TinyGo notes

The interpreter ships as a TinyGo binary named `jennifer`.
`make build` produces both binaries side by side:
the TinyGo `jennifer` (shipping) and the standard-Go `jennifer-go`
(dev/full-feature). To produce only one, use `make build-tinygo`
or `make build-go`. All three regenerate the version file before
compiling.

A few constraints shape the implementation:

- **No `reflect`-heavy code.** Tagged-union `Value` instead of interfaces
  with type assertions in hot paths.
- **No `text/template`, no goroutines in the interpreter core.** Not
  needed yet, but worth not introducing accidentally.
- **No `encoding/json` for in-binary serialization.** The reflect-based
  marshaler is fragile under TinyGo, so the AST JSON emitter is
  hand-rolled (see [CLI > Inspection](cli.md#inspection-tokens-and-ast)).
- **No `-ldflags "-X package.var=value"`.** TinyGo 0.41 silently ignores
  the `-X` directive. Use the codegen path
  (`scripts/gen-version.sh` -> `internal/version/version_gen.go`) for
  build-time string injection. See [CLI > Version injection](cli.md#version-injection).
- **No hard dependencies on a hosted runtime.** A long-term goal is to
  embed the interpreter into the **McFly OS** kernel (also TinyGo), so
  ambient stdin, dynamic linking, and the like should not be assumed.
- **`testing` runs under regular `go test`.** TinyGo's `testing` support
  is partial; we develop and verify with `go test ./...`.

Verify both builds after non-trivial changes:

```sh
make build
./jennifer run examples/hello.j     # TinyGo binary
./jennifer-go run examples/hello.j  # Go binary (full host features)
```
