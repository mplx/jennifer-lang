# Profiler (`cmd/jennifer/profile.go`, `internal/profile`)

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


Part of the [CLI reference](cli.md).
