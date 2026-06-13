# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# osinfo.j - exercise the namespaced `os` library in both its
# bare-prefix and aliased-prefix forms. In real programs you pick
# one form per program (the style guide and user-guide/imports.md
# document why). This example deliberately uses both so the two
# spellings sit side-by-side; the runtime accepts the combination
# because each `use` activates one prefix without removing the
# other.
#
# Golden output is pinned to "linux"/"amd64" because Jennifer is
# Linux-only today and the test environment is amd64. When Windows /
# macOS support lands, this example's golden file
# (examples/expected/osinfo.txt) will need a per-OS strategy - until
# then, this example is the deliberate home for printing the actual
# host values; showcase.j only convert.typeOf()-s them so it stays
# portable.

use io;
use os;
use os as o;

# Bare prefix: the library name is the prefix.
io.printf("bare prefix:\n");
io.printf("  os.PLATFORM = %s\n", os.PLATFORM);
io.printf("  os.ARCH     = %s\n", os.ARCH);
io.printf("  os.DIRSEP   = %s\n", os.DIRSEP);
io.printf("  os.PATHSEP  = %s\n", os.PATHSEP);

# Aliased prefix: `o.` resolves to the same library via the alias.
io.printf("aliased prefix (`use os as o;`):\n");
io.printf("  o.PLATFORM  = %s\n", o.PLATFORM);
io.printf("  o.ARCH      = %s\n", o.ARCH);

# Alias is a rename operation: possible but should be avoided
use convert as realConvert;
use os as convert;
io.printf("aliased prefix (`use os as convert;`):\n");
io.printf("  convert.PLATFORM = %s\n", convert.PLATFORM);
io.printf("  convert.ARCH     = %s\n", convert.ARCH);
