# `dotenv` - `.env` configuration files

Import with `import "dotenv.j" as dotenv;`. Read the `KEY=VALUE` files that keep
configuration and secrets out of source. `parse` turns text into a `map of
string to string`, `read` parses a file, and `load` parses a file **and** sets
each variable in the process environment. Over `fs` + `strings` + `os`; pure
`.j`, runs on **both** binaries.

```jennifer
import "dotenv.j" as dotenv;

def cfg as map of string to string init dotenv.parse("PORT=8080\n# note\nNAME=\"ada\"");
io.printf("%s\n", $cfg["PORT"]);     # 8080

dotenv.load(".env");                  # or set them in the environment
```

Runnable: [`examples/modules/dotenv_demo.j`](https://github.com/mplx/jennifer-lang/blob/main/examples/modules/dotenv_demo.j).

## Functions

| Call | Returns | Notes |
| ---- | ------- | ----- |
| `dotenv.parse(text)` | `map of string to string` | Parse `.env` text; touches nothing else. |
| `dotenv.read(path)` | `map of string to string` | Read and parse a file. |
| `dotenv.load(path)` | `map of string to string` | Read, parse, and `os.setEnv` each variable; returns the map. |

## Syntax

Each non-blank, non-comment line is a `KEY=VALUE` assignment:

```dotenv
# a comment line
PORT=8080
export NAME="ada lovelace"     # a leading `export` is ignored
GREETING='hi # not a comment'  # single quotes are literal
PATH="/a\t/b"                  # double quotes expand \n \t \r
TOKEN=abc123        # trailing inline comment (unquoted values only)
EMPTY=
```

- **Comments** - a line starting with `#` is skipped; on an **unquoted** value a
  ` #` (space then hash) starts an inline comment. Inside quotes, `#` is literal.
- **`export`** - a leading `export ` prefix is stripped, so a file that doubles as
  a shell script parses the same.
- **Double quotes** expand the escapes `\n`, `\t`, `\r` (and `\"` for a literal
  quote); an unknown escape keeps its character.
- **Single quotes** are literal - no escapes, no interpolation.
- **Unquoted** values are trimmed of surrounding whitespace.
- The value may contain `=` (only the first `=` splits the line); a line with no
  `=` or an empty key is ignored; a later duplicate key wins.

## Scope

- **No variable interpolation.** `${OTHER}` inside a value is not expanded (the
  value is taken literally); reference other variables in your program instead.
- **No multi-line values.** Each assignment is one line.
- **`load` overwrites.** It sets every parsed variable with `os.setEnv`,
  replacing any existing value. To make `.env` a *default* (don't override a real
  environment variable), read the map and set only the keys `os.getEnv` reports
  as empty.

## See also

- [fs.md](../libraries/fs.md) - the file read `read` / `load` use.
- [os.md](../libraries/os.md) - `os.setEnv` / `os.getEnv` for the environment.
- [toml.md](../libraries/toml.md) - for richer, typed, nested configuration.
- [modules/index.md](index.md) - the module catalog and import rules.
