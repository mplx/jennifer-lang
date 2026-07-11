# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# sasl_test.j - white-box tests for sasl.j. Run with:
#
#     jennifer test modules/sasl_test.j
#
# The overlay splices sasl.j in front of this file, so the tests reach its
# private helpers (baseEncode, ctrlA) by bare identifier alongside the exported
# encoders. Reference base64 values were computed independently.
use testing;

func testBaseEncode() {
    testing.assertEqual(baseEncode("hello"), "aGVsbG8=");
    testing.assertEqual(baseEncode(""), "");
}

func testCtrlAIsOneByte() {
    testing.assertEqual(len(ctrlA()), 1);
}

func testPlain() {
    # base64("\0user\0pass")
    testing.assertEqual(plain("user", "pass"), "AHVzZXIAcGFzcw==");
}

func testLoginSteps() {
    testing.assertEqual(loginUser("user"), "dXNlcg==");
    testing.assertEqual(loginPass("pass"), "cGFzcw==");
}

func testBearer() {
    # SASL XOAUTH2: base64("user=...\x01auth=Bearer ...\x01\x01")
    testing.assertEqual(bearer("me@gmail.com", "ya29.TOKEN"),
        "dXNlcj1tZUBnbWFpbC5jb20BYXV0aD1CZWFyZXIgeWEyOS5UT0tFTgEB");
}
