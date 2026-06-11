# Control flow

## Operators

| Operator             | Meaning                                                  |
|----------------------|----------------------------------------------------------|
| `+`                  | addition (`int`/`float`); also concatenation on `string` |
| `-`, `*`             | subtraction, multiplication (`int`/`float`)              |
| `/`                  | **true division - always returns `float`**               |
| `//`                 | floor (integer) division; `int // int -> int`            |
| `%`                  | modulo (`int` only)                                      |
| unary `-`            | numeric negation (`int`/`float`)                         |
| `<`, `>`, `<=`, `>=` | numeric comparison; result is `bool`                     |
| `==`                 | equality; same-kind plus `int`/`float` promotion; `bool` |
| `and`, `or`          | logical; both operands `bool`; short-circuit             |
| `not`                | unary logical negation; operand `bool`                   |

**Division has two operators.** `/` always returns a `float` (Python 3
style). `//` returns the floor, keeping the type when both operands are
ints:

```jennifer
5 / 2          # 2.5 (float)
5 // 2         # 2   (int)
5.0 / 2.0      # 2.5 (float)
5.7 // 2.0     # 2.0 (float - floor of a float division)
```

So `def x as int init 5 / 2;` is rejected (right side is float). Use
`5 // 2` for an int result, or `def x as float init 5 / 2;`.

(Line comments are `#`, freeing `//` for the Python-3 floor-division
operator. The `#` choice also lets Jennifer files start with a shebang:
`#!/usr/bin/env -S jennifer run`.)

Precedence (low to high): `or`, `and`, `not`, comparison, additive (`+`, `-`),
multiplicative (`*`, `/`, `//`, `%`), unary `-`. Use parentheses to override:
`(1 + 2) * 3`. Examples that follow the rules:

```jennifer
not 1 == 2                  # not (1 == 2) -> true
1 > 0 and 2 > 1             # true
true or false and false     # true or (false and false) -> true
-3 + 10                     # (-3) + 10 -> 7
-3 * 2                      # (-3) * 2 -> -6
```

**`and` and `or` short-circuit.** The right operand is only evaluated when
the left doesn't already decide the result. That matters when the right side
has side effects:

```jennifer
def gate as bool init false;
def result as bool init $gate and expensive();   # expensive() not called
```

Mixed `int`/`float` arithmetic promotes the int to float and the result is a
float (`3 + 0.5` -> `3.5`). **`/` always returns `float`**, even with two
int operands (`5 / 2` is `2.5`, not `2`). Use `//` when you want an
integer quotient: `5 // 2` is `2`. This is Python-3 division, not C/Java
division.

## Conditionals and loops

```jennifer
if ($n == 0) {
    io.printf("zero");
} elseif ($n < 10) {
    io.printf("small");
} else {
    io.printf("large");
}

while ($i < 5) {
    $i = $i + 1;
}

for (def i as int init 0; $i < 10; $i = $i + 1) {
    io.printf($i);
}

# for-each over a list or map.
for (def x in $xs) {
    io.printf("%d ", $x);
}
for (def k in $m) {
    io.printf("%s=%d ", $k, $m[$k]);
}
```

Conditions in `if`, `elseif`, `while`, and `for` **must be `bool`** - there
is no implicit truthiness. Use a comparison (`$x == 0`) to get a bool.
For-each (`for (def x in $coll)`) doesn't take a condition - it walks
the whole collection.

### Loop variable scope

C-style `for` opens its own scope. **Where you `def` the iterator
variable decides whether you can still see it after the loop.**

```jennifer
# Loop-local: declare inside the for-init. The iterator lives only for
# the duration of the loop.
for (def i as int init 0; $i < 10; $i = $i + 1) {
    io.printf("%d\n", $i);
}
io.printf("%d\n", $i);   # ERROR: `i` not in scope here
```

```jennifer
# Outer-scope: declare in the surrounding scope, assign in the for-init.
# The variable survives past the loop and holds the value that made the
# condition false (10 here).
def i as int;
for ($i = 0; $i < 10; $i = $i + 1) {
    io.printf("%d\n", $i);
}
io.printf("%d\n", $i);   # ok - prints 10
```

The loop-local form is the [recommended style](style-guide.md#loops);
reach for the outer-scope form only when you actually need to inspect
the iterator after the loop ends. For-each (`for (def x in $coll)`)
is always loop-local - the iteration variable lives in a fresh scope
each pass through the loop and is gone once the loop exits.

## `repeat ... until` (post-test loop)

For loops that should run **at least once**, then keep going until a
condition becomes true:

```jennifer
def n as int init 0;
repeat {
    io.printf("n=%d\n", $n);
    $n = $n + 1;
} until ($n >= 3);
# prints n=0, n=1, n=2 - the body runs three times before until is true.
```

The body runs unconditionally on entry, then `until (cond)` is checked
**after** each iteration. The loop stops when `cond` evaluates true.

This is the post-test counterpart to `while`. The keyword pair
`repeat`/`until` was chosen over `do { } while ...` so the condition
inversion ("loop until done") reads as English and matches the rest of
Jennifer's word-operator style (`and`, `or`, `not`). Like every other
condition slot, `cond` must be `bool`.

## `break` and `continue`

`break;` exits the **innermost** enclosing loop:

```jennifer
for (def i as int init 0; $i < 10; $i = $i + 1) {
    if ($i == 5) { break; }
    io.printf("%d ", $i);
}
# prints "0 1 2 3 4 "
```

`continue;` skips the rest of the current iteration and starts the
next one. In a C-style `for` loop, the step expression (`$i = $i + 1`)
still runs before the condition is re-checked - matching the behaviour
in C, Go, Java, and Python:

```jennifer
for (def i as int init 0; $i < 5; $i = $i + 1) {
    if ($i % 2 == 0) { continue; }
    io.printf("%d ", $i);
}
# prints "1 3 "
```

Both work in `while`, C-style `for`, for-each (`for (def x in $coll)`),
and `repeat ... until`. In `repeat`, `continue` jumps to the `until`
check (skipping the rest of the body); the loop still terminates
normally when `until` becomes true.

Misuse:

- `break` and `continue` **only exist inside a loop**. Using one at
  the top level or as a stray statement in a method body that has no
  enclosing loop is a positioned runtime error.
- They do **not** cross the method-call boundary. A `break` inside a
  method body looks for a loop in **that method**, not in the caller.
  If the called method has no loop, the `break` errors.
- They only catch the **innermost** loop. To exit several levels at
  once, use a flag variable that the outer loop checks, or refactor
  the inner work into a method that `return`s when done.

## `exit`

`exit;` terminates the whole program immediately - it skips the rest
of the current method, every caller frame, and every remaining
top-level statement. The bare form yields exit code 0:

```jennifer
use io;
io.printf("ok\n");
exit;                   # process ends with code 0
io.printf("never\n");   # not reached
```

`exit EXPR;` sets the exit code; `EXPR` must evaluate to `int`:

```jennifer
use io;
io.printf("error: input missing\n");
exit 2;                 # process ends with code 2
```

`exit` is **distinct from `return`**. `return` ends the current
method's body and yields a value to the caller; `exit` ends the
program. Use `return` when a method has done its job; use `exit` when
the whole run is over.

