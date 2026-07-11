# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# smtp_demo.j - the smtp module (modules/smtp.j) with mime (modules/mime.j):
# build a message and send it. Run:
#
#     jennifer run examples/modules/smtp_demo.j
#
# By default it targets a local SMTP server on 127.0.0.1:2525 (e.g. a dev
# MailHog / aiosmtpd); with none running it prints the message and the
# connection error rather than failing. Point `host` / `port` at a real server
# to actually deliver. Needs the default `jennifer` binary (`jennifer-tiny`
# has no network stack).
use io;
import "../../modules/mime.j" as mime;
import "../../modules/smtp.j" as smtp;

# Build the message with mime.
def msg as mime.Part init mime.text("text/plain", "Hello from Jennifer! Accents: café.");
$msg = mime.withHeader($msg, "From", mime.address("Claude", "claude@example.com"));
$msg = mime.withHeader($msg, "To", "you@example.com");
$msg = mime.withHeader($msg, "Subject", "Jennifer SMTP demo");
def wire as string init mime.encode($msg);

io.printf("=== message ===\n%s\n\n", $wire);

# Send it. security "none" | "starttls" | "tls"; set user / pass for AUTH.
def opts as smtp.Options init smtp.Options{host: "127.0.0.1", port: 2525,
    security: "none", clientName: "jennifer.demo", user: "", pass: ""};
def rcpts as list of string init ["you@example.com"];

try {
    smtp.send($opts, "claude@example.com", $rcpts, $wire);
    io.printf("delivered to %s:%d\n", $opts.host, $opts.port);
} catch (e) {
    io.printf("no SMTP server at %s:%d (%s)\n", $opts.host, $opts.port, $e.message);
}
