# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# smtp.j - an SMTP send client: the line-oriented command/response dialogue of
# RFC 5321, over the `net` system library (plaintext, implicit TLS, or
# STARTTLS) with SASL AUTH PLAIN (LOGIN / XOAUTH2 to follow). The message body is any
# string, typically built by the `mime` module. Because it uses `net`, this
# module runs on the default `jennifer` binary only (`jennifer-tiny` stubs the
# network stack).
#
#     import "smtp.j" as smtp;
#     import "mime.j" as mime;
#     def opts as smtp.Options init smtp.Options{host: "mail.example.com",
#         port: 587, security: "starttls", clientName: "me.example.com",
#         user: "me@example.com", pass: "secret"};
#     def msg as mime.Part init mime.text("text/plain", "Hello!");
#     $msg = mime.withHeader($msg, "Subject", "Hi");
#     def rcpts as list of string init ["you@example.com"];
#     smtp.send($opts, "me@example.com", $rcpts, mime.encode($msg));
#
# `send` throws a catchable `Error` (kind "smtp") when the server rejects a
# command. TLS certificate verification is the `net` default. Addresses must
# be ASCII: a non-ASCII host or envelope address (an IDN domain) throws rather
# than misrouting, until an IDNA module lands - punycode the domain yourself.
use net;
use strings;
use convert;
use encoding;

# Connection settings. `security` is "none" (plaintext), "tls" (implicit TLS
# on connect), or "starttls" (upgrade after EHLO). `user` "" skips AUTH.
# `clientName` is the EHLO identity (defaults to "localhost" when empty).
export def struct Options {
    host as string,
    port as int,
    security as string,
    clientName as string,
    user as string,
    pass as string
};

# One parsed SMTP reply: its (final) status code and the raw reply text.
def struct Reply {
    code as int,
    text as string
};

# --- pure protocol helpers (private, unit-tested) ------------------

# parseCode reads the leading 3-digit status of a reply line, or 0.
func parseCode(line as string) {
    if (len($line) < 3) {
        return 0;
    }
    return convert.toInt(strings.substring($line, 0, 3));
}

# stripCR drops a single trailing CR.
func stripCR(line as string) {
    if (strings.endsWith($line, "\r")) {
        return strings.substring($line, 0, len($line) - 1);
    }
    return $line;
}

# replyFinalCode returns the status code once `text` holds a complete reply
# (a terminated final line: "NNN" or "NNN " rather than a "NNN-" continuation),
# else -1. Multi-line replies use "-" after the code on every line but the last.
func replyFinalCode(text as string) {
    def lines as list of string init strings.split($text, "\n");
    def n as int init len($lines);
    def i as int init 0;
    while ($i < $n - 1) {
        def line as string init stripCR($lines[$i]);
        def spaced as bool init len($line) >= 4 and strings.substring($line, 3, 4) == " ";
        def final as bool init len($line) == 3 or $spaced;
        if ($final) {
            return parseCode($line);
        }
        $i = $i + 1;
    }
    return -1;
}

# authPlain builds the SASL PLAIN token: base64 of "\0user\0pass".
func authPlain(user as string, pass as string) {
    def raw as string init "\0" + $user + "\0" + $pass;
    return encoding.toText(convert.bytesFromString($raw, "utf-8"), "base64");
}

# dotStuff prefixes an extra "." to any body line that begins with one, per the
# RFC 5321 transparency rule, so the "." end-of-data terminator is unambiguous.
func dotStuff(body as string) {
    def out as string init "";
    def first as bool init true;
    for (def line in strings.split($body, "\n")) {
        if (not $first) {
            $out = $out + "\n";
        }
        $first = false;
        def l as string init $line;
        if (strings.startsWith($l, ".")) {
            $l = "." + $l;
        }
        $out = $out + $l;
    }
    return $out;
}

# crlf normalises any line endings to CRLF.
func crlf(s as string) {
    def out as string init strings.replace($s, "\r\n", "\n");
    return strings.replace($out, "\n", "\r\n");
}

# --- net dialogue (private) ----------------------------------------

func fail(reply as Reply, ctx as string) {
    def msg as string init $ctx + ": " + convert.toString($reply.code);
    $msg = $msg + " " + strings.trim($reply.text);
    throw Error{kind: "smtp", message: $msg, file: "", line: 0, col: 0};
}

# expect throws unless the reply code falls in [lo, hi].
func expect(reply as Reply, lo as int, hi as int, ctx as string) {
    if ($reply.code < $lo or $reply.code > $hi) {
        fail($reply, $ctx);
    }
}

# requireAscii throws when `s` is not pure ASCII. Until an IDNA / SMTPUTF8 path
# exists, a non-ASCII host or envelope address (an IDN domain, or a non-ASCII
# local part) cannot be put on the wire correctly, so this fails loudly rather
# than sending a misrouted address.
func requireAscii(s as string, what as string) {
    if (encoding.isAscii(convert.bytesFromString($s, "utf-8"))) {
        return;
    }
    def msg as string init $what + " is not ASCII: an IDN domain needs punycode";
    $msg = $msg + " or SMTPUTF8, not yet supported (" + $s + ")";
    throw Error{kind: "smtp", message: $msg, file: "", line: 0, col: 0};
}

# readReply reads from the connection until a complete SMTP reply arrives.
func readReply(conn as net.Conn) {
    def buf as string init "";
    while (true) {
        def code as int init replyFinalCode($buf);
        if ($code >= 0) {
            return Reply{code: $code, text: $buf};
        }
        def chunk as bytes init net.readBytes($conn, 512);
        if (len($chunk) == 0) {
            return Reply{code: replyFinalCode($buf + "\n"), text: $buf};
        }
        $buf = $buf + convert.stringFromBytes($chunk, "utf-8");
    }
    return Reply{code: -1, text: $buf};
}

# command sends one line and returns the server's reply.
func command(conn as net.Conn, line as string) {
    net.writeBytes($conn, convert.bytesFromString($line + "\r\n", "utf-8"));
    return readReply($conn);
}

func clientName(opts as Options) {
    if (len($opts.clientName) == 0) {
        return "localhost";
    }
    return $opts.clientName;
}

# greet sends EHLO and expects a 2xx; used after connect and after STARTTLS.
func greet(conn as net.Conn, opts as Options) {
    expect(command($conn, "EHLO " + clientName($opts)), 250, 259, "EHLO");
}

# authenticate runs SASL PLAIN when credentials are set.
func authenticate(conn as net.Conn, opts as Options) {
    if (len($opts.user) == 0) {
        return;
    }
    expect(command($conn, "AUTH PLAIN " + authPlain($opts.user, $opts.pass)), 235, 235, "AUTH");
}

func dial(opts as Options) {
    def addr as string init $opts.host + ":" + convert.toString($opts.port);
    if ($opts.security == "tls") {
        return net.connectTLS($addr);
    }
    return net.connect($addr);
}

# --- send (exported) -----------------------------------------------

# send delivers `message` (a full RFC 5322 message, e.g. from mime.encode) to
# every recipient, with `from` as the envelope sender. It opens the connection
# per `opts.security`, greets, upgrades with STARTTLS when asked, authenticates
# when credentials are present, then runs MAIL FROM / RCPT TO / DATA / QUIT. A
# server rejection throws a catchable `Error` (kind "smtp").
export func send(opts as Options, from as string, recipients as list of string,
        message as string) {
    # Fail fast on IDN / non-ASCII before opening a connection.
    requireAscii($opts.host, "host");
    requireAscii($from, "sender address");
    for (def rcpt in $recipients) {
        requireAscii($rcpt, "recipient address");
    }
    def conn as net.Conn init dial($opts);
    expect(readReply($conn), 220, 220, "greeting");
    greet($conn, $opts);
    if ($opts.security == "starttls") {
        expect(command($conn, "STARTTLS"), 220, 220, "STARTTLS");
        $conn = net.startTLS($conn);
        greet($conn, $opts);
    }
    authenticate($conn, $opts);
    expect(command($conn, "MAIL FROM:<" + $from + ">"), 250, 259, "MAIL FROM");
    for (def rcpt in $recipients) {
        expect(command($conn, "RCPT TO:<" + $rcpt + ">"), 250, 259, "RCPT TO");
    }
    expect(command($conn, "DATA"), 354, 354, "DATA");
    def payload as string init dotStuff(crlf($message)) + "\r\n.\r\n";
    net.writeBytes($conn, convert.bytesFromString($payload, "utf-8"));
    expect(readReply($conn), 250, 259, "end of DATA");
    command($conn, "QUIT");
    net.close($conn);
}
