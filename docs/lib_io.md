# `io` - input/output

Enable with `use io;`. Today the library provides `printf` and `sprintf`. Both
share a Go-style format-string mini-language.

## `printf`

Writes formatted output to standard output. Three calling shapes:

```jennifer
printf("hi\n");                              // literal string (no verbs)
printf($x);                                  // single non-string value, displayed
printf("you are %d years old!\n", $age);     // format string + arguments
printf("%s = %d, ok=%t\n", "answer", 42, true);
```

## `sprintf`

Same arguments as `printf` but **returns** the formatted string instead of
writing it.

```jennifer
def msg as string init sprintf("%d + %d = %d", 1, 2, 3);
printf("%s\n", $msg);   // "1 + 2 = 3"
```

## Format verbs

| Verb | Required kind  | Notes                          |
|------|----------------|--------------------------------|
| `%d` | `int`          | decimal                        |
| `%f` | `float`        | shortest round-trip            |
| `%s` | `string`       | raw                            |
| `%t` | `bool`         | `true` / `false`               |
| `%v` | any            | uses the value's display form  |
| `%%` | -              | literal `%`                    |

Mismatches (wrong verb for the value kind, too few or too many args, dangling
`%`, unknown verb) all produce runtime errors. A literal `%` in any string
passed to `printf`/`sprintf` must be doubled to `%%`.

## Float display

Floats always display with a decimal point so the value's type stays visible:
`5.0` prints as `"5.0"`, not `"5"`. That matters most after the Python 3
division change - `4 / 2` is the float `2.0`, and you can tell at a glance
rather than wondering whether it's an `int`.

See also: [user-guide.md](user-guide.md), [technical.md](technical.md#libraries-and-builtins).
