# Imports

Two keywords, two mechanisms:

```jennifer
use io;                  // library import - enables `io` library (printf, sprintf)
import "helpers.j";      // file import - splices helpers.j here
```

## Library imports

`use NAME;` enables a built-in module. Standard functions live in
topic-based libraries; nothing is auto-loaded except the special `core`
library. Each library has its own reference doc; the table below is the
index.

| Library   | Enable with      | Contents                                                                                                  | Reference                                  |
|-----------|------------------|-----------------------------------------------------------------------------------------------------------|--------------------------------------------|
| `io`      | `use io;`        | `printf`, `sprintf` and the format-verb mini-language                                                     | [libraries/io.md](../libraries/io.md)       |
| `convert` | `use convert;`   | `int`, `float`, `string`, `bool`, `typeOf` - explicit casts                                               | [libraries/convert.md](../libraries/convert.md) |
| `math`    | `use math;`      | `abs`, `min`, `max`, `sqrt`, `pow`, `floor`, `ceil`, `round`; constants `PI`, `E`                         | [libraries/math.md](../libraries/math.md)   |
| `strings` | `use strings;`   | `upper`, `lower`, `contains`, `startsWith`, `endsWith`, `indexOf`, `trim`, `trimLeft`, `trimRight`, `replace`, `repeat`, `substring`, `split`, `chars`, `join` | [libraries/strings.md](../libraries/strings.md) |
| `core`    | *(auto-loaded)*  | `len`, `has(map, key)`, `JENNIFER_VERSION`. No `use` needed; writing `use core;` is a runtime error.      | [libraries/core.md](../libraries/core.md)   |

See [libraries/index.md](../libraries/index.md) for a fuller catalog and
the library-organization principles, or
[libraries/cheatsheet.md](../libraries/cheatsheet.md) for an
alphabetical lookup of every builtin function and constant.

Quick orientation - if you're reading top to bottom and just want a flavor:

```jennifer
use io;
use convert;
use math;

printf("%s\n", typeOf(5 / 2));         // "float"      [convert]
printf("%d\n", floor(PI * 2.0));       // 6            [math + io]
printf("%s\n", string(true));          // "true"       [convert]
```

The per-library docs cover every function in detail along with error cases.

## File imports

`import "PATH.j";` textually includes another `.j` source file at the
point of import. The path is a **string literal** that must end in `.j`.
Relative paths resolve from the directory of the file containing the
import; absolute paths and subdirectories work:

```jennifer
import "helpers.j";          // sibling file
import "subdir/utils.j";     // subdirectory
import "../shared/util.j";   // parent dir
import "/abs/path/lib.j";    // absolute path
```

File imports may appear anywhere a statement is allowed, including inside a
block:

```jennifer
use io;
import "helpers.j";          // ← spliced here; whatever helpers.j contains lands here
printf($helper_value);
```

Circular imports (file A imports file B, B imports A) are detected and
rejected with an error.

Mixing the keywords produces a helpful error:

```
import io;           → error: use `use io;` for system libraries
use foo.j;           → error: use `import "foo.j";` for files
import foo.j;        → error: file imports take a string literal: `import "foo.j";`
```

Notes:

- The imported file's contents must be valid where the import appears. A file
  containing a top-level `def` cannot be imported inside a block (since
  definitions are only allowed at the top level).
