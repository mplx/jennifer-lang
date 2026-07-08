# Test runner (`cmd/jennifer/test.go`, `internal/lib/testing`)

`jennifer test <file.j>` discovers the file's test methods and runs
them, building on the `testing` library's assertion vocabulary and
runner primitives. It consolidates what would otherwise be a
Jennifer-coded framework into "primitives in Go, orchestration as
subcommand" - the same shape as `fmt` / `lint` / `profile`.

**Flow.** The subcommand parses, preprocesses, and runs the file (so its
methods hoist and its top level executes for setup), then:

- **Discovery** - `Interpreter.MethodNames()` filtered to the `test*`
  convention, overridable with `--filter=REGEX`.
- **Per test** - call `setUp` if present, run the test through
  `CallByName` (timing + `ClassifyError` into a record), call `tearDown`
  if present. Both hooks are looked up by name and absent by default.
- **Report** - `--format=text|tap|junit` via `testinglib.RenderReport`;
  exit `0` all pass / `1` failures / `2` runner error (parse / IO), the
  `jennifer lint` shape.
- **`--isolated`** - run each test in a fresh interpreter subprocess:
  `os.Executable()` re-invokes this binary as `jennifer test
  --testing-single METHOD FILE.j`, which runs exactly one method and
  prints a one-line result. The parent records the child's exit code and
  line - coarser than in-process (no structured kind/position), the trade
  for a clean interpreter state per test.

**Assertions** (`internal/lib/testing/assertions.go`, in both binaries).
Six builtins - `assertEqual` / `assertNotEqual` / `assertTrue` /
`assertFalse` / `assertContains` / `assertThrows` - reduce to
`Value.Equal` / Kind dispatch in Go. On failure they throw the canonical
`Error{kind: "assertion"}` positioned at the call site, which
`testing.run` catches and classifies exactly like a user
`throw Error{...}`. `testing.runWith(name, args)` and
`Interpreter.CallByNameWith` bind arguments for framework dispatchers;
the zero-arg `CallByName` stays the entrypoint the runner uses.

**Enabling interpreter change.** A Go builtin can now raise a *catchable*
Jennifer error. Previously `evalCall` / `evalQualifiedCall` flattened any
builtin error into a fresh `runtimeError` (losing a custom `kind`). Now
`builtinError` passes an `*ErrorSignal` / `*ExitSignal` through unwrapped
- so `interpreter.RaiseError(kind, msg, ...)` throws a real Jennifer
error - and `BuiltinCtx` carries the call-site `file/line/col` so the
error anchors at the call. No existing builtin returned a signal, so the
change is additive.

**TinyGo.** The subcommand is build-tag split like the other dev tools:
`test.go` (`!tinygo`) has it, `devtools_tinygo.go` stubs it. The
assertion vocabulary and runner primitives live in the always-built
`testing` library, so a hand-written TinyGo suite can still call
`testing.assertEqual` / `testing.run` directly.


Part of the [CLI reference](cli.md).
