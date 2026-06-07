# Control flow

## Operators

| Operator             | Meaning                                                  |
|----------------------|----------------------------------------------------------|
| `+`                  | addition (`int`/`float`); also concatenation on `string` |
| `-`, `*`             | subtraction, multiplication (`int`/`float`)              |
| `/`                  | **true division - always returns `float`**               |
| `div`                | floor (integer) division; `int div int → int`            |
| `%`                  | modulo (`int` only)                                      |
| unary `-`            | numeric negation (`int`/`float`)                         |
| `<`, `>`, `<=`, `>=` | numeric comparison; result is `bool`                     |
| `==`                 | equality; same-kind plus `int`/`float` promotion; `bool` |
| `and`, `or`          | logical; both operands `bool`; short-circuit             |
| `not`                | unary logical negation; operand `bool`                   |

**Division has two operators.** `/` always returns a `float` (Python 3
style). `div` returns the floor, keeping the type when both operands are
ints:

```jennifer
5 / 2          // 2.5 (float)
5 div 2        // 2   (int)
5.0 / 2.0      // 2.5 (float)
5.7 div 2.0    // 2.0 (float - floor of a float division)
```

So `def x as int init 5 / 2;` is rejected (right side is float). Use
`5 div 2` for an int result, or `def x as float init 5 / 2;`.

(`//` would have been the Python 3 spelling, but `//` is line-comment syntax
in Jennifer. The Pascal-style `div` keyword fills the same role.)

Precedence (low to high): `or`, `and`, `not`, comparison, additive (`+`, `-`),
multiplicative (`*`, `/`, `div`, `%`), unary `-`. Use parentheses to override:
`(1 + 2) * 3`. Examples that follow the rules:

```jennifer
not 1 == 2                  // not (1 == 2) -> true
1 > 0 and 2 > 1             // true
true or false and false     // true or (false and false) -> true
-3 + 10                     // (-3) + 10 -> 7
-3 * 2                      // (-3) * 2 -> -6
```

**`and` and `or` short-circuit.** The right operand is only evaluated when
the left doesn't already decide the result. That matters when the right side
has side effects:

```jennifer
def gate as bool init false;
def result as bool init $gate and expensive();   // expensive() not called
```

Mixed `int`/`float` arithmetic promotes the int to float and the result is a
float (`3 + 0.5` -> `3.5`). `int / int` is integer division; if either
operand is a float, division is float division.

## Conditionals and loops

```jennifer
if ($n == 0) {
    printf("zero");
} elseif ($n < 10) {
    printf("small");
} else {
    printf("large");
}

while ($i < 5) {
    $i = $i + 1;
}

for (def i as int init 0; $i < 10; $i = $i + 1) {
    printf($i);
}

// for-each over a list or map.
for (def x in $xs) {
    printf("%d ", $x);
}
for (def k in $m) {
    printf("%s=%d ", $k, $m[$k]);
}
```

Conditions in `if`, `elseif`, `while`, and `for` **must be `bool`** - there
is no implicit truthiness. Use a comparison (`$x == 0`) to get a bool.
For-each (`for (def x in $coll)`) doesn't take a condition - it walks
the whole collection.
