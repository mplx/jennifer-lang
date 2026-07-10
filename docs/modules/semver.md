# `semver` - Semantic Versioning 2.0.0

Import with `import "semver.j" as semver;`. Parses, compares, sorts, and
increments strict [SemVer 2.0.0](https://semver.org) version strings. Pure
Jennifer - parsing uses the canonical SemVer regex (`regex`), the
precedence comparison and sort are hand-written - so it runs on either
binary.

```jennifer
use io;
import "semver.j" as semver;

def v as semver.Version init semver.parse("1.4.2-rc.1+build.9");
io.printf("%d.%d.%d pre=%s\n", $v.major, $v.minor, $v.patch, $v.prerelease);
io.printf("rc < release: %t\n", semver.lt($v, semver.parse("1.4.2")));   # true
io.printf("next minor: %s\n", semver.toString(semver.incMinor($v)));      # 1.5.0
```

Runnable: [`examples/modules/semver_demo.j`](https://github.com/mplx/jennifer-lang/blob/main/examples/modules/semver_demo.j).

## The `Version` struct

```jennifer
export def struct Version {
    major as int,
    minor as int,
    patch as int,
    prerelease as string,   # "" when absent (the text after `-`, no dash)
    build as string         # "" when absent (the text after `+`, no plus)
};
```

A value-semantic struct you hold and pass around; `semver` stores nothing.
Build it with `semver.parse` or a literal
(`semver.Version{major: 1, minor: 0, patch: 0, prerelease: "", build: ""}`).

## Surface

| Call                       | Returns          | Notes                                                                 |
| -------------------------- | ---------------- | --------------------------------------------------------------------- |
| `semver.parse(s)`          | `Version`        | Parse a strict version string; `throw`s (`kind: "value"`) on invalid. |
| `semver.isValid(s)`        | `bool`           | Whether `s` is a strict SemVer string (no throw).                     |
| `semver.toString(v)`       | `string`         | Canonical form; round-trips `parse`.                                  |
| `semver.compare(a, b)`     | `int`            | `-1` / `0` / `1` by SemVer precedence. Build metadata ignored.        |
| `semver.lt(a, b)`          | `bool`           | `compare(a, b) < 0`.                                                   |
| `semver.eq(a, b)`          | `bool`           | `compare(a, b) == 0` (so `1.0.0+a` equals `1.0.0+b`).                  |
| `semver.gt(a, b)`          | `bool`           | `compare(a, b) > 0`.                                                   |
| `semver.isStable(v)`       | `bool`           | `major >= 1` and no prerelease. `0.y.z` is unstable by convention.    |
| `semver.isPrerelease(v)`   | `bool`           | Whether a prerelease tag is present.                                  |
| `semver.incMajor(v)`       | `Version`        | `major+1`; resets minor / patch and clears prerelease + build.        |
| `semver.incMinor(v)`       | `Version`        | `minor+1`; resets patch and clears prerelease + build.                |
| `semver.incPatch(v)`       | `Version`        | `patch+1`; clears prerelease + build.                                 |
| `semver.sort(vs)`          | `list of Version`| A new list ordered ascending by precedence.                           |

## Strict, not a loose parser

`semver.parse` accepts exactly `MAJOR.MINOR.PATCH` with an optional
`-prerelease` and `+build`, per the spec. It **rejects** everything the
grammar disallows:

```jennifer
semver.isValid("1.2.3");        # true
semver.isValid("1.2.3-rc.1");   # true
semver.isValid("1.0.0+build");  # true
semver.isValid("1.2.3.4");      # false - four segments have no defined order
semver.isValid("1.2");          # false - too few
semver.isValid("01.0.0");       # false - leading zero in a numeric part
semver.isValid("1.0.0-01");     # false - leading zero in a numeric prerelease id
semver.isValid("1.0.0-");       # false - empty prerelease
```

A looser N-segment form (`1.2.3.4`) has no defined ordering (`1.2.3` vs
`1.2.3.0`?), which would quietly break sorting, so it is invalid rather
than best-effort. Jennifer's own `meta.VERSION` is valid strict SemVer, so
`semver.parse(meta.VERSION)` works out of the box.

Use `isValid` to test without a throw; `parse` for the value (wrap it in
`try` / `catch` to handle untrusted input).

## Precedence

`compare` implements SemVer clause 11 exactly:

1. Compare `major`, then `minor`, then `patch` numerically.
2. A version **with** a prerelease ranks **below** the same version
   without one: `1.0.0-alpha < 1.0.0`.
3. Two prereleases compare field by field (split on `.`): numeric
   identifiers compare numerically and rank below alphanumeric ones, which
   compare in ASCII order; a longer run of otherwise-equal fields ranks
   higher.
4. **Build metadata is ignored** - `1.0.0+a` and `1.0.0+b` are equal.

```jennifer
# 1.0.0-alpha < 1.0.0-alpha.1 < 1.0.0-alpha.beta < 1.0.0-beta
#   < 1.0.0-beta.2 < 1.0.0-beta.11 < 1.0.0-rc.1 < 1.0.0
```

Note `beta.2 < beta.11` (numeric, not lexical) - the classic loose-parser
trap the field-by-field rule avoids.

## Sorting a list

`lists.sort` is scalar-only, so `semver.sort` runs its own pass over
`compare` and returns a new ascending list (the input is untouched -
value semantics):

```jennifer
def vs as list of semver.Version init [];
$vs[] = semver.parse("2.0.0");
$vs[] = semver.parse("1.0.0-alpha");
$vs[] = semver.parse("1.0.0");
$vs[] = semver.parse("1.10.0");
$vs[] = semver.parse("1.2.0");

for (def s in semver.sort($vs)) {
    io.printf(" %s", semver.toString($s));
}
# sorted: 1.0.0-alpha 1.0.0 1.2.0 1.10.0 2.0.0
```

## Out of scope

Range / constraint matching (`^1.2.0`, `>=1.0.0`, `~1.2.3`) is a separate,
harder parser and is not part of this module - `semver` ships the version
*values* and their ordering. Constraint solving lands later with the
package-manager work.

## See also

- [regex.md](../libraries/regex.md) - `parse` matches against the
  canonical SemVer pattern with named groups.
- [meta.md](../libraries/meta.md) - `meta.VERSION`, itself valid SemVer.
- [modules/index.md](index.md) - the module catalog and import rules.
