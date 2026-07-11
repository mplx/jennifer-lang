# Profiler (`cmd/jennifer/profile.go`, `internal/profile`)

`jennifer profile <prog.j>` runs the program with the evaluator instrumented and
attributes work back to Jennifer source positions (`file:line:col`) - the gap
`go tool pprof` leaves, since it profiles the *interpreter binary*, not the `.j`
program inside it. The program's own output is redirected to **stderr** so the
profile owns **stdout** cleanly, even in the binary form:

```sh
jennifer profile prog.j                       # table to stdout, program output to stderr
jennifer profile --allocs prog.j              # value-semantics (copy) profile
jennifer profile --format=pprof p.j > p.pb.gz # gzipped protobuf for go tool pprof / speedscope
```

## Statement profile

The default profile records, per source position, how many times a statement
ran and how long it took - **self** (this statement alone) versus **cumulative**
(this statement plus everything it called), highest self-time first:

```text
$ jennifer profile examples/profile.j
Jennifer statement profile (wall-clock, self = excluding nested statements)

  HITS        SELF          CUM  POSITION
     1  2.208054ms  16.438373ms  examples/profile.j:76:1
     1   321.116µs  10.776339ms  examples/profile.j:36:5
   200  5.292611ms   5.553693ms  examples/profile.j:38:9
   200  4.755398ms   4.755398ms  examples/profile.j:37:9
     8  1.438971ms   1.438971ms  examples/profile.j:51:30
     ...
```

| Column     | Meaning                                                                          |
| ---------- | -------------------------------------------------------------------------------- |
| `HITS`     | How many times the statement at this position executed.                          |
| `SELF`     | Wall-clock spent *in this statement*, excluding nested statements it called.      |
| `CUM`      | Cumulative wall-clock: this statement plus everything it called.                 |
| `POSITION` | The `file:line:col` of the statement. Rows are sorted by `SELF` descending.       |

A high `SELF` is a hot line; a high `CUM` with low `SELF` is a line that mostly
waits on the work it dispatches.

## Modes and formats

| Flag              | Effect                                                                                     |
| ----------------- | ------------------------------------------------------------------------------------------ |
| *(default)*       | Statement profile: hit counts + self / cumulative time per position.                       |
| `--allocs`        | Value-semantics profile instead: where compound values are copied (see below).             |
| `--format=table`  | Human-readable text (default).                                                             |
| `--format=pprof`  | Gzipped protobuf, hand-encoded to keep the zero-dependency stance; `go tool pprof` and speedscope.app read it. |
| `--format=trace`  | Chrome-trace JSON of the method-call timeline (open in `chrome://tracing` / Perfetto).     |

Unknown `--format` and the unsupported `--allocs --format=trace` combination
(allocation events have no timeline) are rejected at argument parse, not deferred
to output.

## Allocation profile (`--allocs`)

Because Jennifer is value-semantic, copies are where hidden cost hides.
`--allocs` reports three copy paths per source position:

```text
$ jennifer profile --allocs examples/profile.j
Jennifer allocation profile (value-semantics copies)

COW detachments - an Ensure() that copied a shared backing:
  (none)

Eager copies - a def / assignment / parameter binding that deep-copied a compound value:
  COUNT  POSITION
    200  examples/profile.j:38:28
    200  examples/profile.j:37:9
     50  examples/profile.j:69:9
     ...
  453 copies across 6 sites
```

| Copy path             | What it is                                                                                                       |
| --------------------- | --------------------------------------------------------------------------------------------------------------- |
| **Eager copies**      | A `def` / assignment / parameter binding that deep-copies a compound value (`Value.Copy()`). Where the real allocation cost lives. |
| **COW detachments**   | An `Ensure()` that copied a *shared* backing at a mutation site. The interpreter copies eagerly at every store and keeps the append / index hot loop unshared, so a mutation target almost never holds a shared value - these stay at or near **zero** for ordinary `.j` code (the counter is kept for correctness if a future storage path defers its copy). |
| **Spawn-frame copies**| The scope snapshot taken when a `spawn` launches (`snapshotForSpawn`).                                            |

`examples/profile.j` exercises all three - read it for the eager-vs-COW contrast
in practice.

## Reading a parallel profile

`spawn` bodies are profiled too (each on its own goroutine, onto the shared,
mutex-guarded collector with a per-goroutine self/cumulative accumulator). Two
things to keep in mind:

- **Self time aggregates across goroutines**, so total self time can exceed
  wall-clock elapsed. Four workers each spending 5s at one position report ~20s
  of self time there though only ~5s of wall-clock passed - time-at-position
  summed over all goroutines, like `pprof`'s CPU time exceeding wall time.
- **Blocking counts as self time.** A statement that waits (`task.wait` /
  `task.waitAll` on in-flight workers, or `time.sleep`) attributes that
  wall-clock wait to itself, so a `waitAll` line can show large self time with a
  hit count of one. It is real elapsed time the statement occupied, not
  computation.

## Instrumentation (implementation)

The interpreter carries an optional `Profiler` interface
(`internal/interpreter/profiling.go`) and three gate flags; `nil` means no
profiling, the only hot-path cost being a nil check. The concrete collector
lives in `internal/profile` and is injected only by this subcommand, so no
profiling machinery compiles into either binary's run path. Hook points:

- **`execStmt`** wraps `execStmtRaw`, timing each statement. A `profChild`
  accumulator splits self from cumulative time (the standard nested-timing
  subtraction). It lives on each goroutine's **root `Environment`**
  (`env.root.profChild`), not the shared `Interpreter`, so parallel `spawn`
  bodies each accumulate into their own snapshot root instead of racing one
  field; the collector's maps are mutex-guarded for the same reason.
- **`evalCall`** times each method-call body for the trace timeline.
- **`ensureCOW`** (replacing the bare `Value.Ensure()` at the four mutation
  sites) records a COW detachment when a shared backing is actually copied.
- **`evalSpawn`** times the `snapshotForSpawn` deep copy.

`evalExpr` is deliberately *not* timed: a `time.Now()` around every literal read
would swamp the profile with its own overhead.

**TinyGo.** Build-tag split like the linter: `profile.go` (`!tinygo`) is the only
importer of `internal/profile`; `devtools_tinygo.go` stubs the subcommand in the
run-only `jennifer-tiny` binary.

Part of the [CLI reference](cli.md).
