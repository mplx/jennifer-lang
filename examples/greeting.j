# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# greeting.j - strings, escape sequences, multiple printf calls.

use io;

def name as string init "Jennifer";
io.printf("hello, ");
io.printf($name);
io.printf("!\n");
