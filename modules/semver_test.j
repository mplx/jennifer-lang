# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# semver_test.j - white-box tests for semver.j. Run with:
#
#     jennifer test modules/semver_test.j
#
# The overlay splices semver.j in first, so these tests reach its private
# helpers (isNum, sign, compareStr, compareIdent) by bare identifier as well
# as its exported surface.
use testing;
use meta;

# cmp compares two version strings, for terse ordering assertions.
func cmp(a as string, b as string) {
    return compare(parse($a), parse($b));
}

func testParseFields() {
    def v as Version init parse("1.2.3-rc.1+build.5");
    testing.assertEqual($v.major, 1);
    testing.assertEqual($v.minor, 2);
    testing.assertEqual($v.patch, 3);
    testing.assertEqual($v.prerelease, "rc.1");
    testing.assertEqual($v.build, "build.5");
}

func testRoundTrip() {
    testing.assertEqual(toString(parse("1.2.3")), "1.2.3");
    testing.assertEqual(toString(parse("1.0.0-alpha.1")), "1.0.0-alpha.1");
    testing.assertEqual(toString(parse("2.0.0+build.7")), "2.0.0+build.7");
    testing.assertEqual(toString(parse("1.2.3-rc.1+b.2")), "1.2.3-rc.1+b.2");
}

func testValidRejects() {
    testing.assertFalse(isValid("1.2.3.4"));   # four segments
    testing.assertFalse(isValid("1.2"));        # too few
    testing.assertFalse(isValid("01.0.0"));     # leading zero in core
    testing.assertFalse(isValid("1.0.0-"));     # empty prerelease
    testing.assertFalse(isValid("1.0.0+"));     # empty build
    testing.assertFalse(isValid("1.0.0-01"));   # leading zero in numeric prerelease
    testing.assertFalse(isValid("1.0.0-a..b")); # empty prerelease field
    testing.assertFalse(isValid(""));
    testing.assertTrue(isValid("0.0.0"));
    testing.assertTrue(isValid("1.0.0-alpha-beta"));  # hyphen inside an identifier
}

func testParseThrowsOnInvalid() {
    testing.assertThrows("parseBad", "value");
}
func parseBad() {
    return parse("not-a-version");
}

func testCoreOrdering() {
    testing.assertEqual(cmp("1.0.0-alpha", "1.0.0"), -1);
    testing.assertEqual(cmp("1.0.0", "1.0.1"), -1);
    testing.assertEqual(cmp("1.0.1", "1.1.0"), -1);
    testing.assertEqual(cmp("1.1.0", "2.0.0"), -1);
    testing.assertEqual(cmp("2.0.0", "1.1.0"), 1);
    testing.assertEqual(cmp("1.0.0", "1.0.0"), 0);
}

# The precedence example from semver.org, item 11.
func testPrereleasePrecedence() {
    testing.assertEqual(cmp("1.0.0-alpha", "1.0.0-alpha.1"), -1);
    testing.assertEqual(cmp("1.0.0-alpha.1", "1.0.0-alpha.beta"), -1);
    testing.assertEqual(cmp("1.0.0-alpha.beta", "1.0.0-beta"), -1);
    testing.assertEqual(cmp("1.0.0-beta", "1.0.0-beta.2"), -1);
    testing.assertEqual(cmp("1.0.0-beta.2", "1.0.0-beta.11"), -1);   # numeric, not lexical
    testing.assertEqual(cmp("1.0.0-beta.11", "1.0.0-rc.1"), -1);
    testing.assertEqual(cmp("1.0.0-rc.1", "1.0.0"), -1);
}

func testBuildIgnored() {
    testing.assertTrue(eq(parse("1.0.0+a"), parse("1.0.0+b")));
    testing.assertEqual(cmp("1.0.0+build.1", "1.0.0+build.2"), 0);
}

func testClassify() {
    testing.assertTrue(isStable(parse("1.0.0")));
    testing.assertFalse(isStable(parse("0.5.0")));       # 0.y.z is unstable
    testing.assertFalse(isStable(parse("1.0.0-rc")));
    testing.assertTrue(isPrerelease(parse("1.0.0-rc")));
    testing.assertFalse(isPrerelease(parse("1.0.0")));
}

func testIncrement() {
    testing.assertEqual(toString(incMajor(parse("1.2.3-rc+b"))), "2.0.0");
    testing.assertEqual(toString(incMinor(parse("1.2.3"))), "1.3.0");
    testing.assertEqual(toString(incPatch(parse("1.2.3"))), "1.2.4");
}

func testSort() {
    def vs as list of Version init [];
    $vs[] = parse("2.0.0");
    $vs[] = parse("1.0.0-alpha");
    $vs[] = parse("1.0.1");
    $vs[] = parse("1.0.0");
    $vs[] = parse("1.1.0");
    def out as list of Version init sort($vs);
    testing.assertEqual(toString($out[0]), "1.0.0-alpha");
    testing.assertEqual(toString($out[1]), "1.0.0");
    testing.assertEqual(toString($out[2]), "1.0.1");
    testing.assertEqual(toString($out[3]), "1.1.0");
    testing.assertEqual(toString($out[4]), "2.0.0");
}

# White-box: private helpers reached by bare identifier.
func testPrivateHelpers() {
    testing.assertTrue(isNum("42"));
    testing.assertFalse(isNum("4a"));
    testing.assertEqual(sign(-5), -1);
    testing.assertEqual(sign(0), 0);
    testing.assertEqual(sign(9), 1);
    testing.assertEqual(compareStr("alpha", "beta"), -1);
    testing.assertEqual(compareIdent("1", "beta"), -1);   # numeric ranks below alphanumeric
    testing.assertEqual(compareIdent("2", "11"), -1);     # numeric compared numerically
}

func testParsesInterpreterVersion() {
    testing.assertTrue(isValid(meta.VERSION));
}
