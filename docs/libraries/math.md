# `math` - numeric functions and constants

Enable with `use math;`. A small set of frequently-needed numeric functions
plus the constants `PI` and `E`. The library is strict on undefined inputs -
anything that would produce `NaN` or `Infinity` in IEEE arithmetic instead
produces a positioned runtime error.

```jennifer
use io;
use math;

printf("%f\n", PI);                    // 3.141592653589793
printf("%d\n", abs(0 - 42));           // 42
printf("%d\n", min(3, 7));             // 3
printf("%f\n", sqrt(2));               // 1.4142135623730951
printf("%f\n", pow(2, 10));            // 1024.0
printf("%d\n", floor(3.7));            // 3
printf("%d\n", ceil(3.2));             // 4
printf("%d\n", round(2.5));            // 3 (half away from zero)
```

## Functions

| Call          | Returns               | Notes                                          |
|---------------|-----------------------|------------------------------------------------|
| `abs(x)`      | same type as `x`      | int → int, float → float                       |
| `min(a, b)`   | int or float          | int+int → int; mixed → float                   |
| `max(a, b)`   | int or float          | same rule as `min`                             |
| `sqrt(x)`     | float                 | errors on negative input                       |
| `pow(x, y)`   | float                 | errors if the result would be NaN or Infinity  |
| `floor(x)`    | int                   | toward `-∞`; accepts int (identity)            |
| `ceil(x)`     | int                   | toward `+∞`                                    |
| `round(x)`    | int                   | half-away-from-zero (`round(2.5)` = `3`)       |

`min`/`max` follow the same numeric-promotion rule as `+`: same-type
arguments return that type; any `float` involved produces a `float`.

## Constants

| Name | Kind  | Value                  |
|------|-------|------------------------|
| `PI` | float | 3.141592653589793...   |
| `E`  | float | 2.718281828459045...   |

Constants are referenced as bare identifiers, like `def const` constants.
They participate in the normal constant-lookup rules.

## Strictness

The library refuses to produce floating-point edge values:

- `sqrt(-1)` - undefined for negative input.
- `pow(0, -1)` - division-by-zero territory; result would be Infinity.
- `pow(-1, 0.5)` - would be NaN.

If a future use case needs the NaN/Infinity values, a `math.NAN` / `math.INF`
constant (or dedicated check functions) can be added later. For now Jennifer
treats them as errors at the boundary - consistent with how the language
already refuses to silently coerce types.

See also: [../user-guide.md](../user-guide.md), [../technical/interpreter.md](../technical/interpreter.md#builtins-and-libraries), [index.md](index.md).
