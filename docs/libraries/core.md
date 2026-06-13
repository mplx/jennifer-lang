# `core` - auto-loaded structural builtins

The `core` library is special: it is **automatically available** in every
Jennifer program. You do **not** write `use core;` - the interpreter
pre-imports it at startup. Writing `use core;` is a runtime error to
keep source files honest about which libraries the program actively
invokes.

```jennifer
use io;

io.printf("five: %d\n", len("hello"));
```

(no `use core;` needed; `len` is already in scope.)

## Why `core` is special

Jennifer's library discipline is "nothing for free": every standard
library is opt-in via `use NAME;`. `core` is the deliberate exception,
reserved for a tiny handful of *polymorphic structural* builtins that
every program needs without ceremony - things that are more like
language operators than library functions. Reserve membership
carefully; almost everything new belongs in `io`, `math`, `convert`,
`strings`, `meta`, or a new domain-library, not here.

## Functions

| Call     | Returns | Notes                          |
| -------- | ------- | ------------------------------ |
| `len(v)` | int     | Structural length, polymorphic |

### `len`

Polymorphic - the same name covers every type where "structural length"
is well-defined:

- **string** → rune count (Unicode code points, not bytes)
- **list**   → element count
- **map**    → entry count
- **bytes**  → byte count

Passing any other kind (`int`, `float`, `bool`, `null`) is a positioned
runtime error.

```jennifer
io.printf("%d\n", len("hello"));            # 5
io.printf("%d\n", len("héllo"));            # 5 (rune count)
io.printf("%d\n", len([1, 2, 3]));          # 3
io.printf("%d\n", len({"a": 1, "b": 2}));   # 2
```

### `has` was here (moved to `maps.has`)

The map-membership test that used to live in core was relocated to
the `maps` library; call it as `maps.has($m, key)` after `use maps;`.
The move keeps core focused on truly polymorphic primitives - `len`
works across four kinds, `has` only ever worked on maps. See
[maps.md](maps.md#maps-has).

### `JENNIFER_VERSION` was here (moved to `meta.JENNIFER_VERSION`)

The build-version constant moved out of `core` to the new `meta`
library, which holds interpreter-self-identity facts. After
`use meta;` it is `meta.JENNIFER_VERSION`. The move keeps core's
charter strictly "polymorphic structural primitive that's more like
an operator than a library function" - the version string was always
a special case under that rule.

See also: [meta.md](meta.md), [../user-guide/index.md](../user-guide/index.md), [../technical/interpreter.md](../technical/interpreter.md#builtins-and-libraries), [index.md](index.md).
