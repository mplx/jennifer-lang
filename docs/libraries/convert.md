# `convert` - explicit type conversion

Enable with `use convert;`. Provides one-argument conversion functions plus
`typeOf` for runtime kind introspection. Every function either returns the
converted value or produces a positioned runtime error.

```jennifer
use io;
use convert;

def n as int init int("42");        # parse string -> 42
def f as float init float(5);       # int -> 5.0
def s as string init string(3.14);  # any -> "3.14"
def b as bool init bool(0);         # 0 -> false

printf("%s\n", typeOf(5 / 2));      # "float" (after Python 3 / change)
printf("%s\n", typeOf(5 // 2));     # "int"
```

## Behavior summary

| Call          | Source kinds                  | Behavior                                                             |
|---------------|-------------------------------|----------------------------------------------------------------------|
| `int(v)`      | int / float / string / bool   | identity / truncate / parse / `true`=1, `false`=0                    |
| `float(v)`    | int / float / string / bool   | convert / identity / parse / `true`=1.0, `false`=0.0                 |
| `string(v)`   | any                           | always succeeds; uses the value's display form                       |
| `bool(v)`     | bool / int / float / string   | identity / canonical only (`0`/`1`, `0.0`/`1.0`, `"true"`/`"false"`) |
| `typeOf(v)`   | any                           | returns the kind as a string: `"int"`, `"float"`, etc.               |

## Errors

- `int("abc")` - parse failure (string doesn't represent a valid integer).
- `int(null)` - no conversion defined.
- `bool("maybe")` - strings: only `"true"` and `"false"` accepted.
- `bool(123)`, `bool(-1)` - ints: only `0` and `1` accepted.
- `bool(1.5)` - floats: only `0.0` and `1.0` accepted.
- Arity errors (too many or too few arguments).

For "any nonzero counts as true" semantics, write the comparison explicitly:

```jennifer
def b as bool init $count != 0;     // not bool($count)
```

This matches the strict-conditions rule everywhere else in Jennifer - if you
want to project an arbitrary value into a bool, state the criterion.

## Notes on the type-name syntax

The names `int`, `float`, `string`, `bool` are type keywords. The parser
allows them in expression position **only** when immediately followed by `(`.
Writing `def x as int init int;` (bare keyword, no call) errors with a hint
to either use it as a conversion call or supply a value. `typeOf` is a normal
identifier and is not a type keyword.

See also: [../user-guide/index.md](../user-guide/index.md), [../technical/interpreter.md](../technical/interpreter.md#builtins-and-libraries), [index.md](index.md).
