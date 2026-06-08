# Jennifer libraries

Jennifer's standard library is split into topic-based modules. Each is
enabled explicitly with `use NAME;`; nothing is auto-loaded. This page
catalogs every library that ships with the interpreter today and links
to the reference doc for each.

> **Looking for one specific function?** See the
> [cheatsheet](cheatsheet.md) - alphabetical list of every builtin
> with its library and a one-line description.

| Library   | Enable with     | Contents                                                                                                                                   | Reference                  |
|-----------|-----------------|--------------------------------------------------------------------------------------------------------------------------------------------|----------------------------|
| `io`      | `use io;`       | `printf`, `sprintf`, and a `%d %f %s %t %v %%` format-verb mini-language                                                                   | [io.md](io.md)             |
| `convert` | `use convert;`  | `int`, `float`, `string`, `bool`, `typeOf` - explicit casts; canonical-only `bool` conversion                                              | [convert.md](convert.md)   |
| `math`    | `use math;`     | `abs`, `min`, `max`, `sqrt`, `pow`, `floor`, `ceil`, `round`; constants `PI`, `E`                                                          | [math.md](math.md)         |
| `strings` | `use strings;`  | `upper`, `lower`, `contains`, `startsWith`, `endsWith`, `indexOf`, `trim`, `trimLeft`, `trimRight`, `replace`, `repeat`, `substring`, `split`, `chars`, `join` (`len` lives in [`core`](core.md)) | [strings.md](strings.md)   |
| `core`    | *(auto-loaded)* | `len` (polymorphic over string/list/map), `has(map, key)`, `JENNIFER_VERSION`. Pre-imported by the interpreter; writing `use core;` is a runtime error. | [core.md](core.md)         |

A quick taste:

```jennifer
use io;
use math;
use strings;

printf("Jennifer %s\n", JENNIFER_VERSION);   # auto-loaded from core
printf("pi is roughly %f\n", PI);
printf("sqrt(2) = %f\n", sqrt(2));
printf("upper: %s\n", upper("hello"));
printf("len: %d\n", len("hello"));           # auto-loaded from core
```

## How libraries are organized

The standard library favors many small, focused modules over a few large
ones. The organizing principle, captured for future extensions:

- Touches I/O (stdin/stdout/files/network/clock) -> `io` (which can later
  split into `fs`, `net`, `time` as it grows).
- Pure value transformation across kinds -> `convert`.
- Pure numeric -> `math`.
- String manipulation -> `strings`.
- Interpreter introspection (version, host, build info) -> auto-loaded
  `core` (reserve carefully; this is the only escape hatch from the
  "nothing for free" rule).
- A genuinely new topic with three or more functions -> a new library.
- A single function with no clear topic -> the most-related existing
  library.

## Naming convention

Library names look mixed at first glance - `strings` is plural but
`math` is singular. The rule:

- **Plural for count nouns**: when the library operates on instances of
  something you can have multiples of. `strings`, `lists` (planned),
  `maps` (planned), `bytes`, `files`.
- **Singular for mass nouns and conceptual wholes**: `math`, `core`,
  `time` (planned), `regex` (planned).
- **Bare verb when the library is named for what it does**, not what
  it touches: `convert`.
- **Idiomatic abbreviations are fine**: `os`, `fs`, `net`, `regex`.

Three practical constraints reinforce the count/mass rule:

1. Type keywords are reserved. `string`, `int`, `float`, `bool`, `list`,
   `map`, `null` cannot be library names because they tokenize as type
   tokens, not IDENTs. The plural form (`strings`, `lists`, `maps`)
   sidesteps this naturally.
2. The rule matches Go's stdlib: `strings` and `bytes` are plural;
   `math`, `io`, `os` are singular. Since the interpreter is written in
   Go, the convention transfers cleanly to library author intuition.
3. Within a library, function names are lowercase / camelCase
   (`upper`, `startsWith`, `typeOf`). Constants are uppercase
   (`PI`, `E`, `JENNIFER_VERSION`).

For implementation notes on how libraries register themselves with the
interpreter (`Register`, `RegisterConst`, the `use`-gated lookup),
see [../technical/interpreter.md > Builtins and libraries](../technical/interpreter.md#builtins-and-libraries).
