# Syntax

## Tokens and whitespace

Whitespace (spaces, tabs, newlines) is **not** significant. Statements are
terminated by `;`.

## Comments

```jennifer
# line comment - runs to end of line

/* block comment -
   can span multiple lines */
```

Block comments don't nest. Because `#` starts a line comment, the first
line of a script may be a Unix shebang and the interpreter will skip it:

```jennifer
#!/usr/bin/env -S jennifer run
use io;
printf("hi\n");
```

(`env -S` splits the rest of the line into arguments, which is how
`jennifer run` reaches the interpreter on Linux.)

## Identifiers

- Variable, method, parameter and library names are letters only: `[A-Za-z]`,
  up to 64 characters. No digits or underscores.
- **Constants** are uppercase chunks joined by single `_` separators:
  `[A-Z]+(_[A-Z]+)*`, up to 64 characters. Every `_` must be immediately
  followed by another uppercase letter. `MAX`, `MAX_RETRIES`, `HTTP_OK`,
  and `A_B_C` are all legal; `_MAX`, `MAX_`, and `MAX__INT` are not.
- **Variable references** use a leading `$`: define `name`, refer to it as
  `$name`.
- **Constant references** are bare (no `$`).
- **Method calls** are bare and followed by `(...)`.
