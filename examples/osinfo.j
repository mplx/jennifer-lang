# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# osinfo.j - exercise the M8 namespaced `os` library: a zero-arg
# qualified call (`os.platform()`), a qualified constant (`os.JENNIFER_OS`),
# and `use os as o;` aliasing.
use io;
use os;

printf("platform: %s\n", os.platform());
printf("OS tag:   %s\n", os.JENNIFER_OS);
