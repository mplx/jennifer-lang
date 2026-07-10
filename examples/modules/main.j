# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# main.j - the entry program. It imports db (which imports config) and also
# imports config directly. Run it with:
#
#     jennifer run examples/modules/main.j
#
# Expected output shows the post-order init and the run-once guarantee:
#
#     config: initialised     <- config initialises first (deepest dependency)
#     db: initialised         <- then db, which imported config
#     app: running            <- finally main's body
#
# config appears exactly once even though both db and main import it.
#
# (Addressing a module's exported members as `alias.member` arrives in a
# later milestone; today an import runs a module for its initialisation.)
use io;
import "./db.j";
import "./config.j";

io.printf("app: running\n");
