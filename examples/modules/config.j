# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# config.j - a leaf module. Its top-level `def const` runs once, the first
# time any module imports it, and never again (run-once caching).
use io;

def const LOADED as int init announce();

func announce() {
    io.printf("config: initialised\n");
    return 0;
}
