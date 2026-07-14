# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>

/**
 * Strict Semantic Versioning 2.0.0 (https://semver.org): parse, compare,
 * increment, and range-match version numbers - the full surface a package
 * registry or dependency resolver needs. A pure-Jennifer reference module (no
 * Go, no system library): parsing uses the canonical SemVer regex (via the
 * `regex` library); precedence comparison, sorting, and range matching are
 * hand-written Jennifer.
 *
 * Ranges follow the npm / Composer grammar: caret `^1.2.0`, tilde `~1.2`,
 * comparators `>=1.0.0 <2.0.0` (space or comma = AND), OR sets `^1 || ^2`,
 * hyphen ranges `1.2.3 - 2.3.4`, x-ranges `1.x` / `1.2.*`, and `*` for any.
 * A prerelease version satisfies a range only when a comparator in the same
 * clause pins a prerelease at the same `major.minor.patch` (the npm rule).
 * @module semver
 * @example
 * def v as semver.Version init semver.parse("1.2.3-rc.1+build.5");
 * if (semver.lt($v, semver.parse("1.2.3"))) { ... }        # true: rc < release
 * semver.satisfies("1.4.0", "^1.2.0 || >=2.5.0");          # true
 * semver.maxSatisfying(["1.0.0", "1.4.3", "2.0.0"], "^1.2.0");   # "1.4.3"
 */
use strings;
use convert;
use regex;

/**
 * A parsed SemVer version: numeric core plus optional prerelease / build tags.
 * @field major {int} the major version
 * @field minor {int} the minor version
 * @field patch {int} the patch version
 * @field prerelease {string} the prerelease tag, or "" if none
 * @field build {string} the build metadata, or "" if none
 */
export def struct Version {
    major as int,
    minor as int,
    patch as int,
    prerelease as string,
    build as string
};

# The official SemVer 2.0.0 grammar as an anchored RE2 pattern with named
# groups: three numeric core parts (no leading zeros), an optional
# dot-separated prerelease (numeric ids have no leading zero; alphanumeric ids
# are free), and an optional dot-separated build.
def const SEMVER as string init "^(?P<major>0|[1-9][0-9]*)\\.(?P<minor>0|[1-9][0-9]*)\\.(?P<patch>0|[1-9][0-9]*)(?:-(?P<prerelease>(?:0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)(?:\\.(?:0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\\+(?P<build>[0-9a-zA-Z-]+(?:\\.[0-9a-zA-Z-]+)*))?$";  # lint-disable: L203

# --- small helpers (private) ---------------------------------------

func sign(n as int) {
    if ($n < 0) {
        return -1;
    }
    if ($n > 0) {
        return 1;
    }
    return 0;
}

# charCode returns the ASCII byte value of a single-character string.
func charCode(c as string) {
    def b as bytes init convert.bytesFromString($c, "utf-8");
    return $b[0];
}

# isNum reports whether s is a run of decimal digits.
func isNum(s as string) {
    return regex.matches("^[0-9]+$", $s);
}

func invalid(s as string) {
    throw Error{kind: "value", message: "invalid semver: " + $s, file: "", line: 0, col: 0};
}

# --- parse / validate / format (exported) --------------------------

/**
 * Parse a version string into a Version.
 * @param s {string} the version text (e.g. "1.2.3-rc.1+build.5")
 * @return {Version} the parsed version
 * @throws {Error} when s is not a valid SemVer 2.0.0 string
 */
export func parse(s as string) {
    def m as regex.Match init regex.find(SEMVER, $s);
    if ($m.start < 0) {
        invalid($s);
    }
    # Every named group is present in groupsNamed even when it did not
    # participate (an absent prerelease / build reads as "").
    return Version{
        major: convert.toInt($m.groupsNamed["major"]),
        minor: convert.toInt($m.groupsNamed["minor"]),
        patch: convert.toInt($m.groupsNamed["patch"]),
        prerelease: $m.groupsNamed["prerelease"],
        build: $m.groupsNamed["build"]
    };
}

/**
 * Report whether a string is a valid SemVer 2.0.0 version.
 * @param s {string} the candidate version text
 * @return {bool} true when s parses as a valid version
 */
export func isValid(s as string) {
    return regex.matches(SEMVER, $s);
}

/**
 * Render a Version back to its canonical string form.
 * @param v {Version} the version to format
 * @return {string} the "major.minor.patch[-prerelease][+build]" text
 */
export func toString(v as Version) {
    def out as string init convert.toString($v.major);
    $out = $out + "." + convert.toString($v.minor);
    $out = $out + "." + convert.toString($v.patch);
    if (len($v.prerelease) > 0) {
        $out = $out + "-" + $v.prerelease;
    }
    if (len($v.build) > 0) {
        $out = $out + "+" + $v.build;
    }
    return $out;
}

# --- comparison (exported) -----------------------------------------

# compareStr does an ASCII-lexical comparison of two identifier strings.
func compareStr(a as string, b as string) {
    def ca as list of string init strings.chars($a);
    def cb as list of string init strings.chars($b);
    def n as int init len($ca);
    if (len($cb) < $n) {
        $n = len($cb);
    }
    def i as int init 0;
    while ($i < $n) {
        def d as int init charCode($ca[$i]) - charCode($cb[$i]);
        if (not ($d == 0)) {
            return sign($d);
        }
        $i = $i + 1;
    }
    return sign(len($ca) - len($cb));
}

# compareIdent compares one prerelease identifier: numeric ids rank below
# alphanumeric ids; numeric ids compare numerically, alphanumeric lexically.
func compareIdent(a as string, b as string) {
    def an as bool init isNum($a);
    def bn as bool init isNum($b);
    if ($an and $bn) {
        return sign(convert.toInt($a) - convert.toInt($b));
    }
    if ($an and not $bn) {
        return -1;
    }
    if (not $an and $bn) {
        return 1;
    }
    return compareStr($a, $b);
}

# comparePre compares two non-empty prerelease strings field by field; a
# longer run of otherwise-equal fields ranks higher.
func comparePre(a as string, b as string) {
    def ai as list of string init strings.split($a, ".");
    def bi as list of string init strings.split($b, ".");
    def n as int init len($ai);
    if (len($bi) < $n) {
        $n = len($bi);
    }
    def i as int init 0;
    while ($i < $n) {
        def c as int init compareIdent($ai[$i], $bi[$i]);
        if (not ($c == 0)) {
            return $c;
        }
        $i = $i + 1;
    }
    return sign(len($ai) - len($bi));
}

/**
 * Compare two versions by SemVer precedence: numeric core, then a prerelease
 * ranks below its release, then prerelease fields. Build metadata is ignored.
 * @param a {Version} the left version
 * @param b {Version} the right version
 * @return {int} -1 if a < b, 0 if equal, 1 if a > b
 */
export func compare(a as Version, b as Version) {
    if (not ($a.major == $b.major)) {
        return sign($a.major - $b.major);
    }
    if (not ($a.minor == $b.minor)) {
        return sign($a.minor - $b.minor);
    }
    if (not ($a.patch == $b.patch)) {
        return sign($a.patch - $b.patch);
    }
    def ap as bool init len($a.prerelease) > 0;
    def bp as bool init len($b.prerelease) > 0;
    if ($ap and not $bp) {
        return -1;
    }
    if (not $ap and $bp) {
        return 1;
    }
    if (not $ap and not $bp) {
        return 0;
    }
    return comparePre($a.prerelease, $b.prerelease);
}

/**
 * Report whether a orders before b.
 * @param a {Version} the left version
 * @param b {Version} the right version
 * @return {bool} true when a < b by SemVer precedence
 */
export func lt(a as Version, b as Version) {
    return compare($a, $b) < 0;
}
/**
 * Report whether a and b have equal precedence (build metadata ignored).
 * @param a {Version} the left version
 * @param b {Version} the right version
 * @return {bool} true when a and b compare equal
 */
export func eq(a as Version, b as Version) {
    return compare($a, $b) == 0;
}
/**
 * Report whether a orders after b.
 * @param a {Version} the left version
 * @param b {Version} the right version
 * @return {bool} true when a > b by SemVer precedence
 */
export func gt(a as Version, b as Version) {
    return compare($a, $b) > 0;
}
/**
 * Report whether a orders at or before b.
 * @param a {Version} the left version
 * @param b {Version} the right version
 * @return {bool} true when a <= b
 */
export func lte(a as Version, b as Version) {
    return compare($a, $b) <= 0;
}
/**
 * Report whether a orders at or after b.
 * @param a {Version} the left version
 * @param b {Version} the right version
 * @return {bool} true when a >= b
 */
export func gte(a as Version, b as Version) {
    return compare($a, $b) >= 0;
}
/**
 * Report whether a and b differ in precedence.
 * @param a {Version} the left version
 * @param b {Version} the right version
 * @return {bool} true when a and b are not equal
 */
export func neq(a as Version, b as Version) {
    return not (compare($a, $b) == 0);
}

# --- classification + increment (exported) -------------------------

/**
 * Report whether the version carries a prerelease tag.
 * @param v {Version} the version to classify
 * @return {bool} true when a prerelease tag is present
 */
export func isPrerelease(v as Version) {
    return len($v.prerelease) > 0;
}

/**
 * Report whether the version is stable: a released (major >= 1) version with no
 * prerelease tag. A 0.y.z version is unstable by SemVer convention.
 * @param v {Version} the version to classify
 * @return {bool} true when major >= 1 and there is no prerelease tag
 */
export func isStable(v as Version) {
    return $v.major >= 1 and len($v.prerelease) == 0;
}

/**
 * The kind of change from a to b: "major", "minor", "patch", "prerelease" (only
 * the prerelease tag differs), or "" when the two are equal. The highest-order
 * difference wins.
 * @param a {Version} the left version
 * @param b {Version} the right version
 * @return {string} the release-type difference
 */
export func diff(a as Version, b as Version) {
    if (not ($a.major == $b.major)) {
        return "major";
    }
    if (not ($a.minor == $b.minor)) {
        return "minor";
    }
    if (not ($a.patch == $b.patch)) {
        return "patch";
    }
    if (not ($a.prerelease == $b.prerelease)) {
        return "prerelease";
    }
    return "";
}

/**
 * Bump the major version, resetting minor / patch and clearing the tags.
 * @param v {Version} the starting version
 * @return {Version} a new version with major + 1 and minor = patch = 0
 */
export func incMajor(v as Version) {
    return Version{major: $v.major + 1, minor: 0, patch: 0, prerelease: "", build: ""};
}
/**
 * Bump the minor version, resetting patch and clearing the tags.
 * @param v {Version} the starting version
 * @return {Version} a new version with minor + 1 and patch = 0
 */
export func incMinor(v as Version) {
    return Version{major: $v.major, minor: $v.minor + 1, patch: 0, prerelease: "", build: ""};
}
/**
 * Bump the patch version, clearing the tags.
 * @param v {Version} the starting version
 * @return {Version} a new version with patch + 1
 */
export func incPatch(v as Version) {
    def next as int init $v.patch + 1;
    return Version{major: $v.major, minor: $v.minor, patch: $next, prerelease: "", build: ""};
}

# --- sort (exported) -----------------------------------------------

# insertSorted places v into an already-sorted list (ascending by compare).
func insertSorted(sorted as list of Version, v as Version) {
    def out as list of Version init [];
    def placed as bool init false;
    for (def w in $sorted) {
        if (not $placed and compare($v, $w) < 0) {
            $out[] = $v;
            $placed = true;
        }
        $out[] = $w;
    }
    if (not $placed) {
        $out[] = $v;
    }
    return $out;
}

/**
 * Return a new list ordered ascending by SemVer precedence. lists.sort is
 * scalar-only, so this is an insertion sort over compare().
 * @param vs {list of Version} the versions to order
 * @return {list of Version} a new list sorted ascending
 */
export func sort(vs as list of Version) {
    def out as list of Version init [];
    for (def v in $vs) {
        $out = insertSorted($out, $v);
    }
    return $out;
}

/**
 * Return a new list ordered descending by SemVer precedence (highest first).
 * @param vs {list of Version} the versions to order
 * @return {list of Version} a new list sorted descending
 */
export func rsort(vs as list of Version) {
    def asc as list of Version init sort($vs);
    def out as list of Version init [];
    def i as int init len($asc) - 1;
    while ($i >= 0) {
        $out[] = $asc[$i];
        $i = $i - 1;
    }
    return $out;
}

# --- ranges (private machinery) ------------------------------------

# A partial numeric core plus how many components the text specified (1-3).
# Missing / wildcard components read as 0; count drives the caret / tilde /
# x-range upper-bound rules.
def struct Core {
    major as int,
    minor as int,
    patch as int,
    count as int
};

# rest returns s with its first n characters dropped.
func rest(s as string, n as int) {
    return strings.substring($s, $n, len($s));
}

# relVersion builds a plain release Version (no prerelease / build).
func relVersion(major as int, minor as int, patch as int) {
    return Version{major: $major, minor: $minor, patch: $patch, prerelease: "", build: ""};
}

# isWild reports whether an operand means "any": empty, *, x, X, or "any".
func isWild(s as string) {
    def t as string init strings.trim($s);
    return $t == "" or $t == "*" or $t == "x" or $t == "X" or $t == "any";
}

# parseCore reads the numeric core of a version / range operand, stopping at the
# first non-numeric component (a wildcard or the prerelease / build suffix).
# count is how many numeric components were consumed (0-3).
func parseCore(s as string) {
    def core as Core init Core{major: 0, minor: 0, patch: 0, count: 0};
    def base as string init $s;
    def dash as int init strings.indexOf($base, "-");
    if ($dash >= 0) {
        $base = strings.substring($base, 0, $dash);
    }
    def plus as int init strings.indexOf($base, "+");
    if ($plus >= 0) {
        $base = strings.substring($base, 0, $plus);
    }
    def idx as int init 0;
    for (def part in strings.split($base, ".")) {
        if ($idx >= 3 or not isNum($part)) {
            break;
        }
        def n as int init convert.toInt($part);
        if ($idx == 0) {
            $core.major = $n;
        } elseif ($idx == 1) {
            $core.minor = $n;
        } else {
            $core.patch = $n;
        }
        $core.count = $idx + 1;
        $idx = $idx + 1;
    }
    return $core;
}

func coreLower(core as Core) {
    return relVersion($core.major, $core.minor, $core.patch);
}

# caretUpper is the exclusive upper bound of a caret range: it bumps the
# left-most non-zero component, widening a partial "^0" per npm semantics.
# ^1.2.3 -> <2.0.0, ^0.2.3 -> <0.3.0, ^0.0.3 -> <0.0.4, ^0.0 -> <0.1.0, ^0 -> <1.0.0.
func caretUpper(core as Core) {
    if ($core.major > 0) {
        return relVersion($core.major + 1, 0, 0);
    }
    if ($core.minor > 0) {
        return relVersion(0, $core.minor + 1, 0);
    }
    if ($core.count >= 3) {
        return relVersion(0, 0, $core.patch + 1);
    }
    if ($core.count == 2) {
        return relVersion(0, 1, 0);
    }
    return relVersion(1, 0, 0);
}

# tildeUpper is the exclusive upper bound of a tilde range and of a bare partial
# / x-range: a specified minor allows patch moves, else major moves. ~1.2.3 /
# ~1.2 / 1.2 / 1.2.x -> <1.3.0; ~1 / 1 / 1.x -> <2.0.0.
func tildeUpper(core as Core) {
    if ($core.count >= 2) {
        return relVersion($core.major, $core.minor + 1, 0);
    }
    return relVersion($core.major + 1, 0, 0);
}

func boundedMatch(v as Version, lo as Version, hi as Version) {
    return compare($v, $lo) >= 0 and compare($v, $hi) < 0;
}

# operandOf strips a leading comparator operator (^ ~ >= <= > < =) off a token.
func operandOf(c as string) {
    def s as string init strings.trim($c);
    if (strings.startsWith($s, ">=") or strings.startsWith($s, "<=")) {
        return rest($s, 2);
    }
    if (strings.startsWith($s, "^") or strings.startsWith($s, "~") or
        strings.startsWith($s, ">") or strings.startsWith($s, "<") or strings.startsWith($s, "=")) {
        return rest($s, 1);
    }
    return $s;
}

# cmpOp evaluates a comparator (>= <= > < =) against v, expanding a partial
# operand the npm way (>1.2 means >=1.3.0, <=1.2 means <1.3.0, etc.).
func cmpOp(v as Version, operand as string, op as string) {
    def o as string init strings.trim($operand);
    if (isWild($o)) {
        return true;
    }
    if (isValid($o)) {
        def c as int init compare($v, parse($o));
        if ($op == ">=") {
            return $c >= 0;
        }
        if ($op == "<=") {
            return $c <= 0;
        }
        if ($op == ">") {
            return $c > 0;
        }
        if ($op == "<") {
            return $c < 0;
        }
        return $c == 0;
    }
    def core as Core init parseCore($o);
    if ($core.count == 0) {
        return true;
    }
    def lo as Version init coreLower($core);
    def hi as Version init tildeUpper($core);
    if ($op == ">=") {
        return compare($v, $lo) >= 0;
    }
    if ($op == ">") {
        return compare($v, $hi) >= 0;
    }
    if ($op == "<=") {
        return compare($v, $hi) < 0;
    }
    if ($op == "<") {
        return compare($v, $lo) < 0;
    }
    return boundedMatch($v, $lo, $hi);
}

# eqOrRange evaluates a bare / `=` operand: an exact match for a full version
# (prerelease honoured), else the partial / x-range it denotes.
func eqOrRange(v as Version, operand as string) {
    def o as string init strings.trim($operand);
    if (isWild($o)) {
        return true;
    }
    if (isValid($o)) {
        return compare($v, parse($o)) == 0;
    }
    def core as Core init parseCore($o);
    if ($core.count == 0) {
        return true;
    }
    if ($core.count >= 3) {
        return compare($v, coreLower($core)) == 0;
    }
    return boundedMatch($v, coreLower($core), tildeUpper($core));
}

# rawComparator evaluates one comparator numerically (no prerelease gate).
func rawComparator(v as Version, c as string) {
    def s as string init strings.trim($c);
    if (isWild($s)) {
        return true;
    }
    if (strings.startsWith($s, "^")) {
        def core as Core init parseCore(rest($s, 1));
        return boundedMatch($v, coreLower($core), caretUpper($core));
    }
    if (strings.startsWith($s, "~")) {
        def core as Core init parseCore(rest($s, 1));
        return boundedMatch($v, coreLower($core), tildeUpper($core));
    }
    if (strings.startsWith($s, ">=")) {
        return cmpOp($v, rest($s, 2), ">=");
    }
    if (strings.startsWith($s, "<=")) {
        return cmpOp($v, rest($s, 2), "<=");
    }
    if (strings.startsWith($s, ">")) {
        return cmpOp($v, rest($s, 1), ">");
    }
    if (strings.startsWith($s, "<")) {
        return cmpOp($v, rest($s, 1), "<");
    }
    if (strings.startsWith($s, "=")) {
        return eqOrRange($v, rest($s, 1));
    }
    return eqOrRange($v, $s);
}

# preAtTuple reports whether a comparator pins a prerelease at v's exact
# major.minor.patch - the npm gate that lets a prerelease satisfy a range.
func preAtTuple(comparator as string, v as Version) {
    def o as string init strings.trim(operandOf($comparator));
    if (not isValid($o)) {
        return false;
    }
    def t as Version init parse($o);
    return len($t.prerelease) > 0 and $t.major == $v.major and $t.minor == $v.minor and $t.patch == $v.patch;
}

# tokenize splits a clause into comparators on whitespace or commas.
func tokenize(clause as string) {
    def spaced as string init strings.replace($clause, ",", " ");
    def out as list of string init [];
    for (def tok in strings.split($spaced, " ")) {
        def t as string init strings.trim($tok);
        if (not ($t == "")) {
            $out[] = $t;
        }
    }
    return $out;
}

# matchesHyphen evaluates a "A - B" range: >=lower(A) and <=B (or <upper(B)
# when B is partial), with the same prerelease gate as a clause.
func matchesHyphen(v as Version, clause as string) {
    def idx as int init strings.indexOf($clause, " - ");
    def aStr as string init strings.trim(strings.substring($clause, 0, $idx));
    def bStr as string init strings.trim(strings.substring($clause, $idx + 3, len($clause)));
    def lo as Version init coreLower(parseCore($aStr));
    if (compare($v, $lo) < 0) {
        return false;
    }
    if (isValid($bStr)) {
        if (compare($v, parse($bStr)) > 0) {
            return false;
        }
    } else {
        if (compare($v, tildeUpper(parseCore($bStr))) >= 0) {
            return false;
        }
    }
    if (isPrerelease($v)) {
        return preAtTuple($aStr, $v) or preAtTuple($bStr, $v);
    }
    return true;
}

# matchesClause evaluates one AND-clause (space / comma separated comparators),
# applying the prerelease gate across the whole clause.
func matchesClause(v as Version, clause as string) {
    def cl as string init strings.trim($clause);
    if (strings.contains($cl, " - ")) {
        return matchesHyphen($v, $cl);
    }
    def comps as list of string init tokenize($cl);
    if (len($comps) == 0) {
        return not isPrerelease($v);
    }
    for (def c in $comps) {
        if (not rawComparator($v, $c)) {
            return false;
        }
    }
    if (isPrerelease($v)) {
        def ok as bool init false;
        for (def c in $comps) {
            if (preAtTuple($c, $v)) {
                $ok = true;
            }
        }
        if (not $ok) {
            return false;
        }
    }
    return true;
}

# --- ranges (exported) ---------------------------------------------

/**
 * Report whether a concrete version satisfies a range. The range grammar is the
 * npm / Composer set: caret `^1.2.0`, tilde `~1.2`, comparators
 * `>=1.0.0 <2.0.0` (space or comma = AND), OR sets `^1 || ^2`, hyphen ranges
 * `1.2.3 - 2.3.4`, x-ranges `1.x` / `1.2.*`, and `*` / `""` / `"any"` for any
 * release. A prerelease version matches only when a comparator in the same
 * clause pins a prerelease at the same major.minor.patch. An invalid version
 * never satisfies anything.
 * @param version {string} the concrete version to test (e.g. "1.4.0")
 * @param range {string} the range expression
 * @return {bool} true when version satisfies range
 */
export func satisfies(version as string, range as string) {
    if (not isValid($version)) {
        return false;
    }
    def v as Version init parse($version);
    def r as string init strings.trim($range);
    if ($r == "" or $r == "*" or $r == "any") {
        return not isPrerelease($v);
    }
    for (def clause in strings.split($r, "||")) {
        if (matchesClause($v, $clause)) {
            return true;
        }
    }
    return false;
}

# partialValid reports whether a partial / operand is structurally well-formed
# (each component a digit run or a wildcard; at most three; tags allowed).
func partialValid(p as string) {
    def t as string init strings.trim($p);
    if (isWild($t)) {
        return true;
    }
    def base as string init $t;
    def dash as int init strings.indexOf($base, "-");
    if ($dash >= 0) {
        $base = strings.substring($base, 0, $dash);
    }
    def plus as int init strings.indexOf($base, "+");
    if ($plus >= 0) {
        $base = strings.substring($base, 0, $plus);
    }
    def parts as list of string init strings.split($base, ".");
    if (len($parts) == 0 or len($parts) > 3) {
        return false;
    }
    for (def part in $parts) {
        if (not (isNum($part) or $part == "*" or $part == "x" or $part == "X")) {
            return false;
        }
    }
    return true;
}

func comparatorValid(c as string) {
    def s as string init strings.trim($c);
    if (isWild($s)) {
        return true;
    }
    return partialValid(operandOf($s));
}

/**
 * Report whether a range expression is well-formed (parseable). Does not
 * evaluate it against any version.
 * @param range {string} the range expression
 * @return {bool} true when the range is valid
 */
export func validRange(range as string) {
    def r as string init strings.trim($range);
    if ($r == "" or $r == "*" or $r == "any") {
        return true;
    }
    for (def clause in strings.split($r, "||")) {
        def cl as string init strings.trim($clause);
        if (not ($cl == "")) {
            if (strings.contains($cl, " - ")) {
                def idx as int init strings.indexOf($cl, " - ");
                def a as string init strings.trim(strings.substring($cl, 0, $idx));
                def b as string init strings.trim(strings.substring($cl, $idx + 3, len($cl)));
                if (not partialValid($a) or not partialValid($b)) {
                    return false;
                }
            } else {
                def comps as list of string init tokenize($cl);
                if (len($comps) == 0) {
                    return false;
                }
                for (def c in $comps) {
                    if (not comparatorValid($c)) {
                        return false;
                    }
                }
            }
        }
    }
    return true;
}

/**
 * Pick the highest version from a list that satisfies the range. Versions that
 * are not valid SemVer are skipped. Returns "" when none match.
 * @param versions {list of string} the candidate versions
 * @param range {string} the range expression
 * @return {string} the highest satisfying version, or "" if none match
 */
export func maxSatisfying(versions as list of string, range as string) {
    def chosen as string init "";
    def have as bool init false;
    def bestVer as Version;
    for (def ver in $versions) {
        if (satisfies($ver, $range)) {
            def parsed as Version init parse($ver);
            if (not $have or compare($parsed, $bestVer) > 0) {
                $bestVer = $parsed;
                $chosen = $ver;
                $have = true;
            }
        }
    }
    return $chosen;
}

/**
 * Pick the lowest version from a list that satisfies the range. Versions that
 * are not valid SemVer are skipped. Returns "" when none match.
 * @param versions {list of string} the candidate versions
 * @param range {string} the range expression
 * @return {string} the lowest satisfying version, or "" if none match
 */
export func minSatisfying(versions as list of string, range as string) {
    def chosen as string init "";
    def have as bool init false;
    def lowVer as Version;
    for (def ver in $versions) {
        if (satisfies($ver, $range)) {
            def parsed as Version init parse($ver);
            if (not $have or compare($parsed, $lowVer) < 0) {
                $lowVer = $parsed;
                $chosen = $ver;
                $have = true;
            }
        }
    }
    return $chosen;
}
