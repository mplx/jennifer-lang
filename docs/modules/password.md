# `password` - password generation, validation, and scoring

Import with `import "password.j" as password;`. Generate passwords against a
policy `Schema`, validate a candidate password against that policy, and estimate
any password's strength in bits of entropy. Pure `.j` over `math` / `strings` /
`lists` / `convert`; runs on both binaries.

> **Security note.** Randomness comes from `math`'s shared, seedable,
> **non-cryptographic** RNG (the same source `uuid` draws from). It is
> predictable to an attacker who can reconstruct the seed, so generated
> passwords are **not** suitable for high-value secrets today. This swaps to a
> crypto-grade source when the `crypto` library lands. Use it for convenience
> passwords, fixtures, and policy checking - not for credentials that must
> resist a determined attacker.

```jennifer
import "password.j" as password;
use io;

def policy as password.Schema init password.schema();          # 16 chars, all classes
def pw as string init password.generate($policy);
io.printf("%s  valid=%t  %s\n", $pw,
          password.validate($policy, $pw).valid,
          password.complexity($pw).label);
```

Runnable: [`examples/modules/password_demo.j`](https://github.com/mplx/jennifer-lang/blob/main/examples/modules/password_demo.j).

## The policy schema

```jennifer
def struct password.Schema {
    minLength as int,          # shortest length (also the generated-length floor)
    maxLength as int,          # longest length (generation picks in the range)
    lower as bool,             # include lowercase in the alphabet
    upper as bool,             # include uppercase
    digits as bool,            # include digits
    symbols as bool,           # include symbols (from symbolSet)
    symbolSet as string,       # the symbol characters to draw from
    minLower as int,           # minimum lowercase (generation guarantees, validation requires)
    minUpper as int,
    minDigits as int,
    minSymbols as int,
    excludeAmbiguous as bool   # drop ambiguous glyphs (0 O o 1 l I |)
};
```

Build a schema with the constructor and copy-on-write modifiers (each returns a
fresh `Schema`, so they chain):

| Call | Returns | |
| ---- | ------- | - |
| `password.schema()` | `Schema` | the strong default: 16 chars, all four classes, min 1 of each |
| `password.withLength(s, lo, hi)` | `Schema` | set the length range (`lo == hi` for a fixed length) |
| `password.withClasses(s, lo, up, dig, sym)` | `Schema` | enable/disable each class (bools) |
| `password.withMinimums(s, lo, up, dig, sym)` | `Schema` | set the per-class minimum counts |
| `password.withSymbolSet(s, chars)` | `Schema` | replace the symbol pool |
| `password.withoutAmbiguous(s)` | `Schema` | exclude ambiguous glyphs from generation |

A **disabled class is authoritative over a leftover minimum**: `withClasses(s,
true, true, true, false)` on the default (which sets `minSymbols` to 1) produces
no symbols and requires none - the enable bool wins.

## Generate

```jennifer
def pw as string init password.generate(schema);
```

Picks a length in `[minLength, maxLength]`, lays down the per-class minimums,
fills the rest from the enabled alphabet, and shuffles. Throws
`Error{kind: "password"}` for an infeasible schema - no classes enabled, an
empty required pool, `minLength > maxLength`, or minimums that exceed the length.

## Validate

```jennifer
def report as password.Report init password.validate(schema, pw);
# Report { valid as bool, reasons as list of string }
```

Checks the **length bounds** and each **per-class minimum**, returning `valid`
plus a list of the failed rules (empty when valid). It checks minimums, not a
whitelist: a password is not rejected for containing characters outside the
schema's alphabet (which is how real password policies read). Disabled classes
impose no minimum.

## Complexity

```jennifer
def strength as password.Strength init password.complexity(pw);
# Strength { length, classes, poolSize, entropy as float, label }
```

Estimates strength independent of any schema. The **alphabet size** is the sum
of the class sizes present (lowercase / uppercase 26 each, digits 10, symbols
the default-set size of 28), and **entropy** is `length * log2(poolSize)` bits.
The `label` bands the entropy:

| Entropy (bits) | Label |
| -------------- | ----- |
| `< 28`         | very weak |
| `28 - 35`      | weak |
| `36 - 59`      | reasonable |
| `60 - 127`     | strong |
| `>= 128`       | very strong |

Entropy is a **ceiling on guessing difficulty given the character set**, not a
measure of memorability or dictionary resistance: `password` scores as
"reasonable" by length yet is trivially guessed. Treat the score as "how big is
the brute-force space," not "is this a good password."

## Scope

- **Non-crypto randomness** (see the security note) - swaps to `crypto` later.
- **Complexity is character-set entropy**, with no dictionary, keyboard-walk, or
  repeated-character analysis. It will happily call a common word "reasonable".
- **Rune-based length**: `len` counts runes, so multi-byte characters count as
  one, consistent with the rest of the language.

## See also

- [uuid.md](../libraries/uuid.md) - the same non-crypto RNG caveat and
  eventual crypto swap.
- [modules/index.md](index.md) - the module catalog and import rules.
