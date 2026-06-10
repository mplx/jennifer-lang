# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# osinfo.j - exercise the M8 namespaced `os` library in both its
# bare-prefix and aliased-prefix forms. In real programs you pick
# one form per program (the style guide and user-guide/imports.md
# document why). This example deliberately uses both so the two
# spellings sit side-by-side; the runtime accepts the combination
# because each `use` activates one prefix without removing the
# other.
#
# Golden output is pinned to "linux" because Jennifer is Linux-only
# today. When Windows / macOS support lands, this example's golden
# file (examples/expected/osinfo.txt) will need a per-OS strategy -
# until then, this example is the deliberate home for printing the
# actual host-OS values; showcase.j only typeOf()s them so it stays
# portable.

use io;
use os;
use os as o;

# Bare prefix: the library name is the prefix.
printf("bare prefix:\n");
printf("  os.platform()       = %s\n", os.platform());
printf("  os.JENNIFER_OS      = %s\n", os.JENNIFER_OS);

# Aliased prefix: `o.` resolves to the same library via the alias.
printf("aliased prefix (`use os as o;`):\n");
printf("  o.platform()        = %s\n", o.platform());
printf("  o.JENNIFER_OS       = %s\n", o.JENNIFER_OS);
