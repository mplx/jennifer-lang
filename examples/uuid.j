# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>

# examples/uuid.j - generate and inspect UUIDs. The UUID values themselves are
# random / time-based, so this prints only the deterministic facts about them
# (version, validity, byte count) plus the fixed NIL constant.

use io;
use uuid;

io.printf("v4 version:  %d\n", uuid.version(uuid.generate("v4")));
io.printf("v7 version:  %d\n", uuid.version(uuid.generate("v7")));
io.printf("v4 valid:    %t\n", uuid.isValid(uuid.generate("v4")));
io.printf("v4 bytes:    %d\n", len(uuid.parse(uuid.generate("v4"))));
io.printf("NIL:         %s\n", uuid.NIL);
io.printf("NIL version: %d\n", uuid.version(uuid.NIL));
io.printf("bad valid:   %t\n", uuid.isValid("not-a-uuid"));
