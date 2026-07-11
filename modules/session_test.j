# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# session_test.j - white-box tests for session.j's pure helpers. Run with:
#
#     jennifer test modules/session_test.j
#
# The overlay splices session.j in front of this file, so the tests reach its
# private key builder, JSON-Pointer escaper, and base64+JSON encode / decode by
# bare identifier. The networked lifecycle (create / load / save / touch /
# destroy) is verified against an in-process memcached server in the Go suite
# (TestSessionLifecycle).
use testing;

func testCacheKey() {
    testing.assertEqual(cacheKey("abc"), "sess:abc");
}

func testPointer() {
    testing.assertEqual(pointer("user"), "/user");
    testing.assertEqual(pointer("a/b"), "/a~1b");     # "/" escapes to ~1
    testing.assertEqual(pointer("x~y"), "/x~0y");     # "~" escapes to ~0
    testing.assertEqual(pointer("~/"), "/~0~1");      # both, in the right order
}

func testRoundTrip() {
    def src as map of string to string init {"user": "ada", "role": "admin"};
    def back as map of string to string init decodeData(encodeData($src));
    testing.assertEqual(len($back), 2);
    testing.assertEqual($back["user"], "ada");
    testing.assertEqual($back["role"], "admin");
}

func testRoundTripUnicode() {
    # base64-wrapping is what makes a non-ASCII value survive the cache round-trip.
    def src as map of string to string init {"name": "José", "city": "München"};
    def back as map of string to string init decodeData(encodeData($src));
    testing.assertEqual($back["name"], "José");
    testing.assertEqual($back["city"], "München");
}

func testEncodedBlobIsAscii() {
    # the stored blob (base64) has equal byte and rune length: pure ASCII.
    def src as map of string to string init {"name": "José"};
    def blob as string init encodeData($src);
    testing.assertEqual(len(convert.bytesFromString($blob, "utf-8")), len($blob));
}

func testDecodeEmptyBlob() {
    # an absent / expired session reads back as an empty map.
    testing.assertEqual(len(decodeData("")), 0);
}

func testRoundTripEmptyMap() {
    def empty as map of string to string init {};
    testing.assertEqual(len(decodeData(encodeData($empty))), 0);
}
