# `strings` - text utilities

Enable with `use strings;`. Twelve functions for the common things you do
with strings: length, case conversion, search, trim, replace, repeat, and
substring extraction.

**String positions are 0-relative.** The first character is at index `0`, not
`1`. So `indexOf("hello", "h")` returns `0`, `substring("hello", 0, 1)` returns
`"h"`, and `len("hello")` is the same as the index just past the last
character (`5`). This matches Python, JavaScript, Go, Rust, Java, C, C++,
C#, Swift, Ruby. Lua/MATLAB/Pascal are 1-relative; Jennifer is not.

**All indices and lengths are rune-based** (Unicode code points), not bytes.
`len("héllo")` is `5`, not `6`. `indexOf` and `substring` agree.

The combination of "0-relative" plus "exclusive end" plus "rune-based" means
`substring(s, indexOf(s, x), len(s))` always does what you'd guess - the
same units come out of every function.

```jennifer
use io;
use strings;

printf("%d\n", len("hello"));               // 5
printf("%s\n", upper("hello"));             // "HELLO"
printf("%t\n", contains("hello", "ell"));   // true
printf("%t\n", startsWith("hello", "he"));  // true
printf("%d\n", indexOf("hello", "l"));      // 2
printf("[%s]\n", trim("  hi  "));           // "[hi]"
printf("%s\n", replace("a-b-c", "-", "/")); // "a/b/c"
printf("%s\n", repeat("ab", 3));            // "ababab"
printf("%s\n", substring("hello", 1, 4));   // "ell"
```

## Functions

| Call                          | Returns | Notes                                                  |
|-------------------------------|---------|--------------------------------------------------------|
| `len(s)`                      | int     | Rune count (Unicode code points)                       |
| `upper(s)`, `lower(s)`        | string  | Case conversion (Unicode-aware)                        |
| `contains(s, sub)`            | bool    | Substring search                                       |
| `startsWith(s, prefix)`       | bool    |                                                        |
| `endsWith(s, suffix)`         | bool    |                                                        |
| `indexOf(s, sub)`             | int     | Rune index of first occurrence; `-1` if not found      |
| `trim(s)`                     | string  | Strip leading and trailing whitespace                  |
| `trimLeft(s)`, `trimRight(s)` | string  | One-sided trim                                         |
| `replace(s, old, new)`        | string  | Replace **all** occurrences of `old` with `new`        |
| `repeat(s, n)`                | string  | `n` copies concatenated; `n` must be non-negative      |
| `substring(s, start)`         | string  | Rune-indexed; from `start` to the end of the string    |
| `substring(s, start, end)`    | string  | Rune-indexed; **exclusive end**                        |

## Indexing rules

`substring`, `indexOf`, and `len` all agree on rune indices. So given
`s = "héllo"`:
- `len(s)` = `5`
- `indexOf(s, "l")` = `2`
- `substring(s, 0, 2)` = `"hé"`
- `substring(s, 2, 5)` = `"llo"`
- `substring(s, 2)` = `"llo"` (2-arg form, end defaults to `len(s)`)

The 2-arg `substring(s, start)` is the same as `substring(s, start, len(s))`
- a common case worth shortening.

## Errors

- `substring(s, -1, 3)` - negative start.
- `substring(s, 0, 99)` - end past the string length.
- `substring(s, 4, 2)` - end before start.
- `repeat(s, -1)` - negative count.
- Non-string arguments where strings are required (`len(42)`).
- Non-int arguments where ints are required (`repeat("x", "3")`).
- Arity errors (too many or too few arguments).

## Whitespace

`trim`/`trimLeft`/`trimRight` use Unicode whitespace (Go's `unicode.IsSpace`):
ASCII spaces, tabs, newlines, plus characters like non-breaking space
(`U+00A0`) and Unicode line separators.

If you need to trim specific characters instead of whitespace, that's not
in v1 - propose `trimChars(s, chars)` as a follow-up if it comes up.

## Coming later

`split(s, sep)`, `chars(s)`, `join(parts, sep)` are deferred until Jennifer
gets arrays (planned in a later milestone). Right now there's no way to
return multiple strings cleanly.

See also: [user-guide.md](user-guide.md), [technical.md](technical.md#libraries-and-builtins).
