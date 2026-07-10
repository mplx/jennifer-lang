# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# db.j - depends on config. Because imports initialise depth-first
# post-order, config is fully initialised before this module's body runs.
use io;
import "./config.j";

def const LOADED as int init announce();

func announce() {
    io.printf("db: initialised\n");
    return 0;
}
