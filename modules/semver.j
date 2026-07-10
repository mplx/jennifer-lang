# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# semver.j - strict Semantic Versioning 2.0.0 (https://semver.org): parse,
# compare, and increment version numbers. The second pure-Jennifer reference
# module (after ansi): no Go, no system library. Parsing uses the canonical
# SemVer regex (via the `regex` library); precedence comparison and the sort
# are hand-written Jennifer - the real algorithmic dogfood.
#
#     import "semver.j" as semver;
#     def v as semver.Version init semver.parse("1.2.3-rc.1+build.5");
#     if (semver.lt($v, semver.parse("1.2.3"))) { ... }   # true: rc < release
#
# Range / constraint matching (`^1.2.0`, `>=1.0.0`) is intentionally out of
# scope; this module is the version values and their ordering.
use strings;
use convert;
use regex;

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

export func isValid(s as string) {
    return regex.matches(SEMVER, $s);
}

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

# compare returns -1 / 0 / 1 by SemVer precedence: numeric core, then a
# prerelease ranks below its release, then prerelease fields. Build metadata
# is ignored.
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

export func lt(a as Version, b as Version) {
    return compare($a, $b) < 0;
}
export func eq(a as Version, b as Version) {
    return compare($a, $b) == 0;
}
export func gt(a as Version, b as Version) {
    return compare($a, $b) > 0;
}

# --- classification + increment (exported) -------------------------

# isPrerelease is true when a prerelease tag is present.
export func isPrerelease(v as Version) {
    return len($v.prerelease) > 0;
}

# isStable is a released (major >= 1) version with no prerelease tag; a
# 0.y.z version is unstable by SemVer convention.
export func isStable(v as Version) {
    return $v.major >= 1 and len($v.prerelease) == 0;
}

export func incMajor(v as Version) {
    return Version{major: $v.major + 1, minor: 0, patch: 0, prerelease: "", build: ""};
}
export func incMinor(v as Version) {
    return Version{major: $v.major, minor: $v.minor + 1, patch: 0, prerelease: "", build: ""};
}
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

# sort returns a new list ordered ascending by SemVer precedence. lists.sort
# is scalar-only, so this is an insertion sort over compare().
export func sort(vs as list of Version) {
    def out as list of Version init [];
    for (def v in $vs) {
        $out = insertSorted($out, $v);
    }
    return $out;
}
