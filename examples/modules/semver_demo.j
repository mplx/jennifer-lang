# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>

/**
 * The semver module (modules/semver.j): parse, compare, and sort Semantic Versioning strings.
 * @module semver_demo
 */
use io;
import "../../modules/semver.j" as semver;

def v as semver.Version init semver.parse("1.4.2-rc.1+build.9");
io.printf("parsed %d.%d.%d pre=%s build=%s\n", $v.major, $v.minor, $v.patch, $v.prerelease, $v.build);
io.printf("1.4.2-rc.1 < 1.4.2 : %t\n", semver.lt($v, semver.parse("1.4.2")));
io.printf("next minor of 1.4.2 : %s\n", semver.toString(semver.incMinor(semver.parse("1.4.2"))));

def vs as list of semver.Version init [];
$vs[] = semver.parse("2.0.0");
$vs[] = semver.parse("1.0.0-alpha");
$vs[] = semver.parse("1.0.0");
$vs[] = semver.parse("1.10.0");
$vs[] = semver.parse("1.2.0");
io.printf("sorted:");
for (def s in semver.sort($vs)) {
    io.printf(" %s", semver.toString($s));
}
io.printf("\n");
