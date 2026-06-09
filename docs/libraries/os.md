# `os` - operating-system glue

Enable with `use os;`. The `os` library is the first **namespaced**
library in Jennifer: every name lives behind the `os.` prefix
(`os.platform()`, `os.JENNIFER_OS`). Nothing here is reachable as a
bare identifier, so the library cannot collide with user methods or
other libraries' bare names.

```jennifer
use io;
use os;

printf("platform: %s\n", os.platform());
printf("os tag:   %s\n", os.JENNIFER_OS);
printf("line:     %s",   os.JENNIFER_LF);
```

`os` is namespaced, so `use os as o;` is permitted and renames the
prefix to `o.` at the use site. The mechanics, the rule that
aliasing is only available for namespaced libraries, and the
canonical-name-shadowing behaviour are documented once in
[user-guide/imports.md > Namespaced libraries and aliasing](../user-guide/imports.md#namespaced-libraries-and-aliasing)
(language angle) and
[technical/interpreter.md > Namespaced libraries](../technical/interpreter.md#namespaced-libraries)
(implementation angle) - they're not repeated here.

Currently ships a deliberately tiny slice so the language can demonstrate
namespaced calls, namespaced constants, and `use os as ALIAS;`
aliasing end-to-end. The library expands in
[M13.1](../milestones.md#m131---os) (`os.args`, `os.exit(n)`, the rest
of the `JENNIFER_*` constants).

## Functions

| Call                | Returns | Notes                                                                |
|---------------------|---------|----------------------------------------------------------------------|
| `os.platform()`     | string  | Operating-system name as reported by the runtime (`"linux"` today).  |
| `os.getEnv(name)`   | string  | Reads an environment variable. Unset variables return `""`, no error.|

## Constants

| Name             | Kind   | Value                                                |
|------------------|--------|------------------------------------------------------|
| `os.JENNIFER_LF` | string | Platform line ending. `"\n"` on Linux today; future cross-platform builds may emit `"\r\n"` on Windows. |
| `os.JENNIFER_OS` | string | OS tag; same value as `os.platform()`, available as a constant for compile-time-style branching. |

See also: [../user-guide/index.md](../user-guide/index.md), [../user-guide/imports.md](../user-guide/imports.md), [../user-guide/style-guide.md](../user-guide/style-guide.md), [index.md](index.md).
