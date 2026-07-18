# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>

/**
 * Loads `intl` message catalogs from TOML files on disk. Writes an en.toml and a
 * de.toml into a scratch directory, reads each back with `fs`, decodes it with
 * `toml`, and bridges the decoded `toml.Value` to the `map of string to string`
 * that `intl.load` wants (`toml.keys` walks the keys, `toml.asString` pulls each
 * value). The scratch dir is created fresh and removed on the way out, and every
 * output is deterministic, so it doubles as a golden test.
 * @module intl
 */

use io;
use fs;
use toml;
use intl;

/** The scratch directory this example writes the catalog files into. */
def const ROOT as string init "intl-toml-tmp";

/** Read a flat "key = value" TOML catalog into a map of string to string. */
func loadCatalog(path as string) {
    def doc as toml.Value init toml.decode(fs.readString($path));
    def out as map of string to string init {};
    for (def key in toml.keys($doc)) {
        $out[$key] = toml.asString($doc, "/" + $key);
    }
    return $out;
}

# Fresh slate, then write the two catalog files (the "assets" a real app ships).
if (fs.exists(ROOT)) {
    fs.removeAll(ROOT);
}
fs.mkdirAll(ROOT);
fs.writeString(ROOT + "/en.toml",
    "greeting = \"Hello, {name}!\"\n"
    + "cart = \"You have {n} items in your cart\"\n"
    + "bye = \"Goodbye\"\n");
fs.writeString(ROOT + "/de.toml",
    "greeting = \"Hallo, {name}!\"\n"
    + "bye = \"Auf Wiedersehen\"\n");

# Load them: the first language loaded (en) is the default / fallback.
intl.load("en", loadCatalog(ROOT + "/en.toml"));
intl.load("de", loadCatalog(ROOT + "/de.toml"));

intl.setLocale("de");
io.printf("locale: %s\n", intl.locale());
io.printf("%s\n", intl.tr("greeting", {"name": "Welt"}));
io.printf("%s\n", intl.tr("bye"));
io.printf("%s\n", intl.tr("cart", {"n": 3}));   # not in de -> falls back to en

# Clean up the scratch directory.
fs.removeAll(ROOT);
io.printf("cleanup: exists = %t\n", fs.exists(ROOT));
