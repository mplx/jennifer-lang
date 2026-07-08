# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>

# examples/json.j - encode and decode JSON with the `json` library.

use io;
use json;
use convert;

# Encode a nested value (structs/maps -> object, list -> array).
io.printf("%s\n", json.encode({"name": "jen", "nums": [1, 2, 3], "ok": true}));

# Pretty-print with 2-space indent.
io.printf("%s\n", json.encodePretty({"a": 1, "b": [true, null]}));

# Decode into a generic map and read fields back.
def m as map of string to int init json.decode("{\"x\": 7, \"y\": 8}");
io.printf("x+y = %d\n", $m["x"] + $m["y"]);

# Numbers decode to int when integral, else float.
io.printf("%s %s\n", convert.typeOf(json.decode("42")), convert.typeOf(json.decode("4.2")));

# No map-to-struct coercion: rebuild a typed struct from the decoded map.
def struct Point { x as int, y as int };
def d as map of string to int init json.decode("{\"x\": 1, \"y\": 2}");
def p as Point init Point{ x: $d["x"], y: $d["y"] };
io.printf("point = %s\n", json.encode($p));
