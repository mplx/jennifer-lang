# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# smtp_test.j - white-box tests for smtp.j's pure protocol helpers. Run with:
#
#     jennifer test modules/smtp_test.j
#
# The overlay splices smtp.j in front of this file, so the tests reach its
# private helpers (parseCode, replyFinalCode, authPlain, dotStuff, crlf,
# clientName) by bare identifier. The networked `send` path is not covered
# here (it needs a live server); it is verified end to end against a local
# SMTP daemon, outside the offline suite.
use testing;

func testParseCode() {
    testing.assertEqual(parseCode("250 OK"), 250);
    testing.assertEqual(parseCode("354 Go ahead"), 354);
    testing.assertEqual(parseCode("ab"), 0);   # too short
}

func testReplyFinalCodeSingleLine() {
    testing.assertEqual(replyFinalCode("220 mail.example.com ready\r\n"), 220);
    testing.assertEqual(replyFinalCode("354 End data with <CR><LF>.<CR><LF>\r\n"), 354);
}

func testReplyFinalCodeMultiLine() {
    # Continuation lines use "250-"; only the final "250 " completes the reply.
    def r as string init "250-mail.example.com\r\n250-PIPELINING\r\n250 STARTTLS\r\n";
    testing.assertEqual(replyFinalCode($r), 250);
}

func testReplyFinalCodeIncomplete() {
    # Only a continuation line so far.
    testing.assertEqual(replyFinalCode("250-mail.example.com\r\n"), -1);
    testing.assertEqual(replyFinalCode(""), -1);
    testing.assertEqual(replyFinalCode("220 partial with no newline yet"), -1);
}

func testAuthPlain() {
    # base64 of "\0user\0pass"
    testing.assertEqual(authPlain("user", "pass"), "AHVzZXIAcGFzcw==");
}

func testDotStuff() {
    testing.assertEqual(dotStuff(".leading"), "..leading");
    testing.assertEqual(dotStuff("normal\r\n.dot line\r\nend"), "normal\r\n..dot line\r\nend");
    testing.assertEqual(dotStuff("no dots here"), "no dots here");
}

func testCrlf() {
    testing.assertEqual(crlf("a\nb"), "a\r\nb");
    testing.assertEqual(crlf("a\r\nb"), "a\r\nb");
}

func testRequireAsciiThrowsOnIdn() {
    testing.assertThrows("idnRecipient", "smtp");
}
func idnRecipient() {
    requireAscii("user@münchen.de", "recipient address");
}

func testRequireAsciiPassesAscii() {
    requireAscii("ok@example.com", "recipient");   # no throw
    testing.assertTrue(true);
}

func testClientNameDefault() {
    def bare as Options init Options{host: "h", port: 25, security: "none",
        clientName: "", user: "", pass: ""};
    testing.assertEqual(clientName($bare), "localhost");
    def named as Options init Options{host: "h", port: 25, security: "none",
        clientName: "me.example", user: "", pass: ""};
    testing.assertEqual(clientName($named), "me.example");
}
