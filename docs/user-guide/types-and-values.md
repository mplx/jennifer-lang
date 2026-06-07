# Types and values

## Types

| Type     | Example literals                       | Notes                                     |
|----------|----------------------------------------|-------------------------------------------|
| `int`    | `0`, `42`, `9001`                      | 64-bit signed                             |
| `float`  | `3.14`, `0.5`                          | 64-bit; promoted from int in mixed math   |
| `string` | `"hello"`, `'single quotes'`           | Supports escape sequences                 |
| `bool`   | `true`, `false`                        | Produced by comparison operators          |
| `null`   | `null`                                 | A type with a single value (the unit)     |
| `list of T`         | `[1, 2, 3]`                 | Ordered sequence; 0-indexed; mutable      |
| `map of K to V`     | `{"a": 1, "b": 2}`          | Key→value; insertion-ordered; mutable     |

Lists and maps are compound types - they hold other Jennifer values.
Nesting works: `list of list of int`, `map of string to list of int`,
etc. Both are **value-typed**: `$ys = $xs;` makes an independent copy,
function parameters bind by copy, and `const` is deep (you cannot
mutate the contents of a `const` list or map).

Note: Jennifer's `list` is an array-backed sequence (Go slice
underneath), not a Lisp linked list. You get O(1) random access via
`$xs[i]`, but no O(1) prepend.

### String escape sequences

Both `"..."` and `'...'` are valid string delimiters. The following escapes
are recognized:

| Escape | Meaning            |
|--------|--------------------|
| `\n`   | newline            |
| `\r`   | carriage return    |
| `\t`   | tab                |
| `\\`   | backslash          |
| `\"`   | double quote       |
| `\'`   | single quote       |
| `\0`   | null character     |

## Variables and constants

```jennifer
def name as int init 5;            // declare and initialize
def count as int;                  // declare with the zero value of int (0)
def const MAX as int init 100;     // constant: uppercase name, init required
```

Uninitialized variables get the **zero value** of their declared type:
`0`, `0.0`, `""`, `false`, or `null`.

**At the def site, names are bare identifiers (no `$`).** The `$` sigil is
reserved for use-site references that read or assign a variable. So:

```jennifer
def x as int init 5;     // def site - bare name
printf($x);              // use site - $ prefix
$x = 42;                 // assignment - $ prefix

def $x as int init 5;    // ERROR: drop the $ here
```

Constants don't use `$` anywhere (they're not mutable, so the sigil would have
no meaning):

```jennifer
def const MAX as int init 100;
printf(MAX);             // use site - bare name
MAX = 200;               // ERROR: cannot assign to constant
```

Assignment uses `=`:

```jennifer
def x as int init 0;
$x = 42;          // ok
$x = "string";    // error: cannot assign string to int variable
```

## Scoping

- Each `{...}` block introduces a new scope.
- A binding is visible from its `def` to the end of the enclosing block, and
  is inherited by inner blocks.
- Inner scopes can **read** outer bindings but **cannot redefine** a name
  already in scope (no shadowing). The interpreter rejects shadowing at
  runtime.
- A `for` loop opens a private scope wrapping `init`/`cond`/`step`/body, so
  the loop variable does not leak out.
- Constants follow the same scoping rules and reject any later assignment.

## Lists and maps

Two compound types let you hold collections of values.

```jennifer
use io;

// A list is an ordered, 0-indexed, mutable sequence.
def xs as list of int init [10, 20, 30];
printf("%d\n", $xs[0]);          // 10
$xs[1] = 99;                     // index write
printf("%d\n", len($xs));        // 3

// A map is a key->value lookup. Iteration is in insertion order.
def m as map of string to int init {"a": 1, "b": 2};
printf("%d\n", $m["a"]);         // 1
$m["c"] = 3;                     // adds new key
$m["a"] = 99;                    // updates existing

// Iterate a list's elements, or a map's keys.
for (def x in $xs) { printf("%d ", $x); }      printf("\n");
for (def k in $m) { printf("%s ", $k); }       printf("\n");
```

A few rules worth knowing up front:

- **Out-of-bounds list reads and writes are errors**, not silent
  no-ops. Same for reads of missing map keys - use
  [`has($m, key)`](../libraries/core.md) to test for presence first.
- **Lists and maps copy on assignment and on function-call binding.**
  `$ys = $xs;` makes an independent copy; mutating `$ys[0]` doesn't
  change `$xs`.
- **`const` is deep.** `def const NUMS as list of int init [1, 2, 3];`
  rejects both `$NUMS = ...` and `$NUMS[0] = ...`. Nested const
  lists/maps follow the same rule transitively.
- **Nesting works**: `list of list of int`,
  `map of string to list of int`, and so on. There's no depth cap, but
  4+ levels of nesting is a code smell - reach for a struct once
  Jennifer has them.
- **Empty literals require a declared type**: `[]` and `{}` are valid
  literals but the surrounding `def x as list of T` decides what they
  hold.
