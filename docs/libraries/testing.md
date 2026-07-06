# `testing` - test-runner primitives

Enable with `use testing;`. Five verbs, one struct. Ships the
irreducible system-side surface a Jennifer-coded test framework
needs: name-based method invocation, a per-process result
accumulator, and a format dispatcher for human / TAP / JUnit
output. The `.j` half (assertion vocabulary, suite organisation,
CLI harness) lives in the M18.x `testing` module and layers on
top of these primitives.

```jennifer
use io;
use testing;

func addPasses() {
    if (1 + 1 != 2) {
        throw Error{kind: "assertion", message: "1 + 1 != 2",
            file: "", line: 0, col: 0};
    }
}

testing.run("addPasses");
io.printf("%s", testing.report(testing.results(), "text"));
```

## Why this is a system library

A pure `.j` test runner would have to take *the test body* as
a value. Jennifer has no function references / first-class
methods today - you can't say `testing.run(myTest)` because
`myTest` is a name, not a value. The interpreter already does
name-based method lookup at every call site; `testing.run`
exposes that as a builtin so a Jennifer-coded module can
dispatch user methods by name without inventing its own
indirection.

`testing.run` is also the one place in the language where
`exit` is intercepted. Language-level `try`/`catch`
deliberately does **not** catch `exit` (see
[control-flow.md](../user-guide/control-flow.md#try-catch-throw));
the testing runner catches it at the Go level so a runaway
`exit` in a test body stays scoped to the runner. This
carve-out is why the primitive can't live in `.j`.

## Surface

| Call                                        | Returns                  | Notes                                                                             |
| ------------------------------------------- | ------------------------ | --------------------------------------------------------------------------------- |
| `testing.run(name)`                         | `testing.Result`         | Look up a zero-arg method by name, call it, catch every failure mode, append.     |
| `testing.results()`                         | `list of testing.Result` | Snapshot of the accumulator. Value semantics - safe to modify.                    |
| `testing.reset()`                           | `null`                   | Clear the accumulator between independent runs.                                   |
| `testing.report(results, format)`           | `string`                 | Render `results` to `"text"`, `"tap"`, or `"junit"`.                              |

## The `testing.Result` struct

```jennifer
def struct testing.Result {
    name as string,
    ms as int,               # elapsed wall time in milliseconds
    passed as bool,
    errorKind as string,     # "" on pass; "runtime" / "error" / "exit" / "unknown" on fail
    errorMessage as string,
    file as string,          # position where the failure originated (if known)
    line as int,
    col as int
};
```

`errorKind` mirrors the strings surfaced by
[`try`/`catch`](../user-guide/control-flow.md#try-catch-throw)
plus a new `"exit"` value for the exit-intercept case:

| Value       | Meaning                                                                         |
| ----------- | ------------------------------------------------------------------------------- |
| `""`        | The test passed.                                                                |
| `"runtime"` | An interpreter runtime error (out-of-bounds, missing key, type mismatch, ...).  |
| `"error"`   | A `throw` whose thrown value wasn't an `Error` struct.                          |
| `"assertion"` etc. | A `throw Error{kind: "assertion", ...}` - `errorKind` mirrors the struct's `kind` field. |
| `"exit"`    | The test body called `exit`. `errorMessage` is `"exit code N"`.                 |
| `"unknown"` | Anything else (method not found, wrong parameter count, ...).                   |

## How `testing.run` handles each failure mode

```jennifer
# Pass path
func passing() {
    return;                     # any normal return counts as a pass
}

# Failure via user throw
func failing() {
    throw Error{
        kind: "assertion",
        message: "expected 42, got 41",
        file: "", line: 0, col: 0
    };
}

# Failure via runtime error
func indexing() {
    def xs as list of int init [];
    def x as int init $xs[5];   # out-of-bounds; kind=runtime
}

# Exit inside a test - captured, doesn't propagate
func earlyExit() {
    exit 1;                     # kind=exit; program keeps running
}
```

Every call to `testing.run` appends exactly one `testing.Result`
to the accumulator. The result is also returned, so the caller
can inspect it immediately.

## Reports

`testing.report(results, format)` takes any list of
`testing.Result` and returns a rendered string. Three formats
ship in v1; format strings are case-sensitive to match the
codec-table shape used by `hash.compute`, `encoding.toText`,
`fs.open`.

### `"text"` - human-readable

```
PASS addPasses (0 ms)
FAIL addFails (1 ms)
     [assertion] 1 + 1 != 2
FAIL earlyExit (0 ms)
     [exit] exit code 1

1 passed, 2 failed, 3 total
```

### `"tap"` - Test Anything Protocol v14

```
TAP version 14
1..3
ok 1 - addPasses
not ok 2 - addFails
  ---
  kind: assertion
  message: 1 + 1 != 2
  ...
not ok 3 - earlyExit
  ---
  kind: exit
  message: exit code 1
  ...
```

Works with the `prove` command and most CI harnesses.

### `"junit"` - JUnit XML

```xml
<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="jennifer" tests="3" failures="2">
  <testcase name="addPasses" time="0.000"></testcase>
  <testcase name="addFails" time="0.001">
    <failure type="assertion" message="1 + 1 != 2">1 + 1 != 2</failure>
  </testcase>
  <testcase name="earlyExit" time="0.000">
    <failure type="exit" message="exit code 1">exit code 1</failure>
  </testcase>
</testsuite>
```

The ubiquitous CI input format.

Unknown format strings error at the boundary:
`testing.report: unknown format "html"; known: "text", "tap",
"junit"`.

## Errors

- **Wrong argument count.** Boundary error:
  `testing.run expects 1 argument (name), got 0`.
- **Wrong argument type.** Boundary error:
  `testing.run: name must be string, got int`.
- **Method with parameters.** `testing.run` in v1 only invokes
  zero-argument methods. Calling it with a method that takes
  parameters records a failing Result with `errorKind="unknown"`
  and the reason in `errorMessage`.
- **Unknown format.** `testing.report: unknown format "html";
  known: "text", "tap", "junit"`.

## Concurrency

The accumulator is guarded by a mutex, so `spawn { testing.run(...) }`
from multiple tasks doesn't race. Ordering is by completion time,
not spawn time. A test runner that wants stable ordering should
run tests sequentially or sort the results before rendering.

## What's not in v1

Recorded so the design decisions stay visible.

- **Subprocess isolation.** A runaway infinite loop in one test
  still holds up the runner (only `exit` is intercepted, not
  divergence). A future `--isolated` flag on the M18.x CLI harness
  would re-invoke `jennifer run testfile.j --testing-single name`
  per test via M15.3 `os.spawn` for complete isolation.
- **Setup / teardown / fixtures.** Belong in the .j-side
  framework; primitives stay narrow.
- **Test discovery by prefix.** `Interpreter.MethodNames()` is
  exported for the M18.x harness; discovery lives up there, not
  here.
- **Skip / xfail.** Same rationale - runner-level policy.
- **Parameterised tests.** Zero-argument methods only in v1; a
  parameterised variant would need first-class function values
  or an explicit `testing.runWith(name, args)` shape.

## See also

- [milestones.md](../milestones.md) - M16.4 design spec and the
  M18.x follow-on.
- [control-flow.md](../user-guide/control-flow.md#try-catch-throw)
  - the `try`/`catch` machinery `testing.run` builds on.
- [concurrency.md](../user-guide/concurrency.md) - `spawn` under
  the mutex-guarded accumulator.
