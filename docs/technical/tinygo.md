# TinyGo notes

Jennifer ships as two binaries built from the same source:

- `jennifer` - default, built with the standard Go toolchain.
  Full host-feature surface: file I/O, `os/exec`, network stack,
  everything.
- `jennifer-tiny` - constrained variant, built with TinyGo. Smaller
  binary, embeddable; missing `os/exec` and the network stack (see
  [TinyGo restrictions](#tinygo-restrictions) below).

`make build` produces both. Use `make build-go` or `make
build-tinygo` for just one; both regenerate the version file
before compiling.

**The language is written to stay TinyGo-clean** even though the
default binary is standard-Go. The `jennifer-tiny` build sits in
CI so any change that breaks TinyGo compatibility surfaces
immediately. A few constraints shape the implementation:

- **No `reflect`-heavy code.** Tagged-union `Value` instead of
  interfaces with type assertions in hot paths. `task.waitAny`
  uses `reflect.Select` (verified under TinyGo); other new
  reflect uses need justification.
- **No `text/template`.** Not needed yet; would drag in fragile
  runtime paths under TinyGo.
- **No `encoding/json` for in-binary serialization.** The
  reflect-based marshaler is fragile under TinyGo, so the AST
  JSON emitter is hand-rolled (see
  [CLI > Inspection](cli_inspect.md#inspection-tokens-and-ast)).
- **Goroutines are allowed** (used for `spawn`),
  but need `-stack-size=2mb` under TinyGo - see
  [the goroutine-stack note below](#tinygo-goroutine-stack).
- **No `-ldflags "-X package.var=value"`.** TinyGo 0.41 silently
  ignores the `-X` directive. Use the codegen path
  (`scripts/gen-version.sh` -> `internal/version/version_gen.go`)
  for build-time string injection. See
  [CLI > Version injection](cli.md#version-injection).
- **No hard dependencies on a hosted runtime.** `jennifer-tiny`
  targets embedded systems, minimal containers, and small-footprint
  scripting hosts where ambient stdin, dynamic linking, and a full
  hosted runtime are not guaranteed.
- **`testing` runs under regular `go test`.** TinyGo's `testing`
  support is partial; we develop and verify with `go test ./...`.

Verify both builds after non-trivial changes:

```sh
make build
./jennifer      run examples/hello.j   # default (standard Go); full host features
./jennifer-tiny run examples/hello.j   # constrained (TinyGo); no os/exec, no net
```

## TinyGo restrictions

A few standard-library features depend on TinyGo runtime support
that isn't there today. Calls into them from `jennifer-tiny` error
with a friendly Jennifer-level message pointing the user at the
default `jennifer` binary. The default binary always supports the
full surface.

| Library | Affected names                                        | What happens on `jennifer-tiny`                                                                       |
| ------- | ----------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| `os`    | `os.run`, `os.spawn`, `os.wait`, `os.poll`, `os.kill` | Runtime error pointing at the default `jennifer` binary. TinyGo's `os/exec` syscalls aren't implemented yet. |
| `net`   | Every entry point (TCP, UDP, DNS)                     | Runtime error pointing at the default `jennifer` binary. TinyGo 0.41 needs a netdev driver at runtime (not registered) and has no `net.ListenPacket` for UDP. Build-tag split: `netlib_tinygo.go` returns friendly errors. |

The constants and the env / argv / flag helpers in `os`
(`os.PLATFORM`, `os.ARCH`, `os.EOL`, `os.DIRSEP`, `os.PATHSEP`,
`os.ARGS`, `os.getEnv`, `os.hasFlag`, `os.flag`) all work fully
on both binaries. Every other shipped library (`io`, `convert`,
`math`, `strings`, `lists`, `maps`, `meta`, `time`, `hash`,
`crc`, `encoding`, `task`, `fs`, `regex`, `testing`) has full
TinyGo support.

**Development subcommands are default-binary only.** `jennifer-tiny`
is a run-only interpreter: `run` and `repl` execute Jennifer source,
but the development subcommands `tokens`, `ast`, `fmt`, `lint`, `profile`, and `test` are
build-tag-excluded (`cmd/jennifer/devtools_tinygo.go`) and return a
friendly error pointing at the default `jennifer` binary. They pull in
lexer-dump, AST-JSON, formatter, and lint machinery that a
minimal-footprint embedding has no use for; build the standard-Go
`jennifer` binary for development work.

**TinyGo goroutine stack**. Jennifer's tree-walking
evaluator wraps each Jennifer-level call in many Go-stack frames
(`execBlock` + `evalCall` + `evalExpr` + ...), so even a
modest-depth recursion (fib 23) easily exceeds TinyGo's default
goroutine stack of ~8KB and segfaults. The Makefile passes
`-stack-size=2mb` to `tinygo build` for `jennifer-tiny` so it
can run recursive `spawn` bodies (and the parallel section of
`examples/benchmark.j`). The default `jennifer` binary doesn't
need this - Go's goroutine stacks grow automatically.

**TinyGo scheduler**. `jennifer-tiny` pins the cooperative
single-thread scheduler (`-scheduler=tasks` in the Makefile).
`spawn` works fully (semantics, loud-fail, registry), but every
goroutine shares one OS thread, so it gives concurrency without
multi-core parallelism: **parallel speedups stay close to 1.0**,
and `-stack-size=2mb` reliably covers recursive `spawn` bodies.
The pin is deliberate - the threads-capable default briefly showed
real multi-core speedups (161% CPU) but segfaulted on recursive
`spawn` bodies, because `-stack-size` doesn't govern OS-thread
stacks. Real multi-core on `jennifer-tiny` is separate future
work, not a default flip; the default `jennifer` binary already
reaches multi-core speedup via Go's scheduler.

Future library work will grow the restrictions table if further
TinyGo runtime gaps surface. Each new gap lands with the same
friendly-message pattern.

## Binary size

The constrained build is the smaller one, which is the point of
`jennifer-tiny` targeting minimal-footprint deployments. Sizes from
`make build` on linux/amd64 (Go 1.26.3, TinyGo 0.41.1, unstripped).
The absolute numbers move with toolchain version and platform; the
ratio is the stable part.

| Binary          | Size                        |
| --------------- | --------------------------- |
| `jennifer`      | ~7.0 MB (7,250,750 bytes)   |
| `jennifer-tiny` | ~4.5 MB (4,621,624 bytes)   |

`jennifer-tiny` comes in at ~64% of the default binary (about a
third smaller). Most of that gap is TinyGo's smaller runtime versus
the standard Go runtime; the run-only trim (excluding the
`tokens` / `ast` / `fmt` / `lint` / `profile` / `test` development subcommands) shaves an
incremental slice on top.

These are unstripped `make build` (dev) sizes. Release builds strip:
the Go binary adds `-trimpath -ldflags "-s -w"` (down to ~5.0 MB, a
third smaller) and the TinyGo binary adds `-no-debug` (down to
~1.8 MB, a ~60% cut). Shipped artifacts are therefore well under the
dev numbers above; dev builds keep symbols for debugging.

## Single-binary benchmark results

Reference numbers from `examples/benchmark.j` on an **AMD Ryzen 5
7600X3D** (6 cores, 12 threads; desktop active, load low). The
serial section is single-threaded by design; the parallel section
fans out to `PARALLEL_WORKERS = 4` spawn tasks per workload. The
interpreter build is the current one (shared-marker COW, lexical
slot resolution, parse-time constant folding), so append-in-a-loop
is amortised O(1).

### `jennifer` (standard-Go binary, default)

```
=== Jennifer benchmark suite ===
build:   go
version: 0.15.0-dev+30.e1e3909

----------------------------------------------------------------------
Workload                               base        iters      time_ms
----------------------------------------------------------------------
fib(N) recursive                         23            1           76
primes up to LIMIT                   100000            1        19406
newton sqrt batch                     10000        10000          317
monte carlo pi                       500000       500000          994
list sort/reverse/slice               10000          500         1536
struct list build+read                10000        10000           34
string join                           10000        10000           10
map insert+read                       10000        10000         1227
----------------------------------------------------------------------
total                                                           23600

Parallel comparison (workers = 4, scheduler = go)
----------------------------------------------------------------------
Workload                          serial_ms       par_ms      speedup
----------------------------------------------------------------------
primes up to LIMIT                    19406         6536         2.97
newton sqrt batch                       317           85         3.73
monte carlo pi                          994          296         3.36
fib(N) x workers                        304           93         3.27
----------------------------------------------------------------------

user   44.67s
sys     0.33s
real   30.62s     (146% CPU)
```

`user + sys` (45.0s) exceeds `real` (30.6s) by ~14.4s - that
gap is Go's concurrent GC running on a second CPU during the
serial section, plus the four spawn workers running in
parallel during the parallel section. Sys time is tiny (0.3s)
because Go's runtime coordinates goroutines with cheap in-process
sync primitives.

### `jennifer-tiny` (TinyGo binary)

```
=== Jennifer benchmark suite ===
build:   tinygo
version: 0.15.0-dev+30.e1e3909

----------------------------------------------------------------------
Workload                               base        iters      time_ms
----------------------------------------------------------------------
fib(N) recursive                         23            1           72
primes up to LIMIT                   100000            1        12128
newton sqrt batch                     10000        10000          212
monte carlo pi                       500000       500000          843
list sort/reverse/slice               10000          500         1085
struct list build+read                10000        10000           31
string join                           10000        10000           21
map insert+read                       10000        10000         1657
----------------------------------------------------------------------
total                                                           16049

Parallel comparison (workers = 4, scheduler = tinygo)
----------------------------------------------------------------------
Workload                          serial_ms       par_ms      speedup
----------------------------------------------------------------------
primes up to LIMIT                    12128        11650         1.04
newton sqrt batch                       212          232         0.91
monte carlo pi                          843          776         1.09
fib(N) x workers                        288          245         1.18
----------------------------------------------------------------------

user   28.74s
sys     0.05s
real   28.97s     (99% CPU)
```

This is the pinned build (`-scheduler=tasks -stack-size=2mb`): the
whole suite completes, serial and parallel. The parallel column
hovers at ~1.0 by design - `spawn` under the cooperative
single-thread scheduler is concurrency, not multi-core throughput -
and `user ~= real` at 99% CPU confirms the single-thread execution
(contrast Go's 146% above). The 2MB stack (up from 1MB) is what lets
the serial recursive `fib` run at all: at 1MB it fit bare but
overflowed nested one call frame deeper, inside `benchFib`.

### Per-workload comparison (serial section)

Ratios are `tiny_ms / go_ms`; > 1.0 means `jennifer-tiny` is slower, < 1.0 means it is faster.

| Workload                  | tiny (ms) | go (ms) | Ratio  | Where the time goes                                                            |
| ------------------------- | --------- | ------- | ------ | ------------------------------------------------------------------------------ |
| `fib(N) recursive`        |        72 |      76 | *0.9x* | Tight interpreter dispatch loop; effectively tied now.                         |
| `primes up to LIMIT`      |     12128 |   19406 | *0.6x* | Long numeric dispatch loop; TinyGo pulls well ahead at scale.                  |
| `newton sqrt batch`       |       212 |     317 | *0.7x* | Float arithmetic + dispatch; TinyGo ahead.                                     |
| `monte carlo pi`          |       843 |     994 | *0.8x* | Float arithmetic + RNG calls; TinyGo ahead.                                    |
| `list sort/reverse/slice` |      1085 |    1536 | *0.7x* | Allocation-heavy; TinyGo's simpler GC beats Go's concurrent GC at this scale.  |
| `struct list build+read`  |        31 |      34 | *0.9x* | Append hot loop is O(1). Both binaries are effectively free.                   |
| `string join`             |        21 |      10 | 2.1x   | Build-up-a-string pattern is O(1); both free in absolute terms (sub-30ms), Go's runtime a step ahead. |
| `map insert+read`         |      1657 |    1227 | 1.4x   | Go's runtime map implementation outperforms TinyGo's at this churn rate.       |
| **total**                 |     16049 |   23600 | *0.7x* | Compute- and alloc-heavy averages favour TinyGo; Go leads only on small stdlib-churn workloads. |

The story has inverted from earlier builds: on the workloads that
carry real time TinyGo is faster across the board, and it posts the
lower serial total (16.0s vs 23.6s). The old "Go wins CPU-bound work
by 2-5x" gap is gone - eliminating the compound-write O(N^2)
deep-copy pattern (which once pushed the tiny total to ~49s) plus the
dispatch-path optimizations closed it and then flipped it.

Two things stand behind the inversion. The dispatch-path work
(constant folding, slot resolution, the `Share()` scalar fast path,
COW) cut per-call cost, and TinyGo's per-call overhead was higher to
start, so it gained *disproportionately* - its serial total fell
~49s -> 16s while Go moved only ~25s -> 24s. And tiny's lead is
*concentrated*: of the ~7.6s gap, `primes` alone is ~7.3s (the
tightest, longest dispatch loop), with `list` adding most of the
rest. Go still wins the stdlib-churn workloads (`map insert+read`,
`string join`), so those stay Go's, but they're a shrinking minority
of the total.

### Parallel section

Speedup is `serial_ms / par_ms`; > 1.0 means the four-worker
version beat serial. Both columns are the pinned build now (no more
crashed `fib` row): Go gets real multi-core speedup, TinyGo's
cooperative scheduler stays at ~1.0 by design.

| Workload             | Go serial (ms) | Go par (ms) | Go speedup | TinyGo serial (ms) | TinyGo par (ms) | TinyGo speedup |
| -------------------- | -------------- | ----------- | ---------- | ------------------ | --------------- | -------------- |
| `primes up to LIMIT` |          19406 |        6536 |   **2.97** |              12128 |           11650 |      1.04      |
| `newton sqrt batch`  |            317 |          85 |   **3.73** |                212 |             232 |      0.91      |
| `monte carlo pi`     |            994 |         296 |   **3.36** |                843 |             776 |      1.09      |
| `fib(N) x workers`   |            304 |          93 |   **3.27** |                288 |             245 |      1.18      |

Go reaches real multi-core speedup (2.97x-3.73x on four workers).
`jennifer-tiny` pins the cooperative scheduler, so `spawn` there
is concurrency without multi-core throughput (~1.0 by design); use
the default binary when parallel throughput matters.

Picking a binary, in short: tiny leads CPU-bound dispatch (well
ahead on the long numeric loops, and it posts the lower serial
total); both are essentially free on allocation-heavy workloads
(tiny's simpler GC slightly ahead); Go leads on string/map churn;
Go is the choice for parallel `spawn`; and tiny trades all of this
for a larger resident-memory footprint (below).

### Memory and page faults

Same machine, `/bin/time` (GNU time) on an equivalent run (per-workload
timings within noise of the tables above):

| Metric              | `jennifer` (Go) | `jennifer-tiny` (TinyGo) |
| ------------------- | --------------- | ------------------------ |
| peak resident (RSS) | ~34 MB          | ~70 MB                   |
| minor page faults   | ~116,500        | ~11,600                  |
| CPU                 | 146%            | 99%                      |

The two runtimes trade opposite resources. **TinyGo uses ~2x the peak
RSS** - its cooperative scheduler reserves each goroutine's full
`-stack-size` up front (the four parallel `spawn` workers each hold a
2MB stack, ~8MB before the interpreter's own data), where Go grows
goroutine stacks on demand from ~8KB. Bumping the stack from 1MB to 2MB
(to fit the recursive `fib` body) doubled that reservation, and it shows
here. **Go, in turn, churns ~10x the page faults** (~116k vs ~12k): its
concurrent GC allocates and reclaims pages aggressively across cores -
the same activity behind its 146% CPU and the `user` >> `real` gap.
TinyGo's simpler GC touches far fewer pages and runs single-threaded at
99% CPU. So TinyGo buys faster compute and low GC churn with a larger,
flatter memory footprint; Go buys a smaller footprint with heavy GC
activity.
