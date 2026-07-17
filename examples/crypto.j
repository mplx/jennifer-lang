# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>

/**
 * Exercises the `crypto` library: crypto-grade random, constant-time
 * comparison, and the two key-derivation functions. The random draws are
 * unpredictable, so the golden prints only their shape (a length, an in-range
 * check); the comparison and KDF outputs are deterministic.
 * @module crypto
 */

use io;
use convert;
use encoding;
use crypto;

# Crypto-grade random: unguessable and unseedable, so assert only the shape.
def token as bytes init crypto.randBytes(16);
io.printf("randBytes(16) len = %d\n", len($token));
def roll as int init crypto.randInt(1, 6);
io.printf("randInt(1,6) in range = %t\n", $roll >= 1 and $roll <= 6);

# Constant-time equality of two byte strings (MAC comparison).
def a as bytes init convert.bytesFromString("same-value", "utf-8");
def b as bytes init convert.bytesFromString("same-value", "utf-8");
def c as bytes init convert.bytesFromString("diff-value", "utf-8");
io.printf("hmacEqual(equal)   = %t\n", crypto.hmacEqual($a, $b));
io.printf("hmacEqual(unequal) = %t\n", crypto.hmacEqual($a, $c));

# Key derivation (deterministic, RFC test vectors).
def password as bytes init convert.bytesFromString("password", "utf-8");
def salt as bytes init convert.bytesFromString("salt", "utf-8");
def empty as bytes;
io.printf("pbkdf(password,salt,1,32) = %s\n",
    encoding.toText(crypto.pbkdf($password, $salt, 1, 32, "sha256"), "hex"));
io.printf("hkdf derived length = %d\n",
    len(crypto.hkdf($password, $salt, $empty, 42, "sha256")));
