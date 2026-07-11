# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# gotify.j - push a notification to a Gotify server (https://gotify.net), a
# tiny real-world module on top of the `http` client. Hold a value-semantic
# `Config` (server URL + application token) and call `push`; it POSTs the
# message form to `URL/message` with the `X-Gotify-Key` header, per Gotify's
# push-message contract (https://gotify.net/docs/pushmsg). Because it builds on
# `http` (which uses `net`), this module needs the default `jennifer` binary.
#
#     import "gotify.j" as gotify;
#
#     def g as gotify.Config init gotify.Config{url: "https://push.example.com",
#         token: "AqB3cD..."};
#     def r as http.Response init gotify.push($g, "Deploy", "build 1234 is live", 5);
#     # $r.status is 200 on success; a bad token comes back as a 4xx value.
#
# The URL and token belong to the caller - read them from the environment or a
# config file; never commit them.
use strings;
use convert;
import "./http.j" as http;

# A Gotify target: the server `url` (no trailing slash) and an application
# `token`. Value-semantic; pass it to each `push` (no mutable module state).
export def struct Config {
    url as string,
    token as string
};

# --- form encoding (private) ---------------------------------------

# isUnreserved reports whether a byte is an unreserved URL character
# (A-Z / a-z / 0-9 / - / _ / . / ~), which is left literal in a form value.
func isUnreserved(b as int) {
    if ($b >= 65 and $b <= 90) {
        return true;
    }
    if ($b >= 97 and $b <= 122) {
        return true;
    }
    if ($b >= 48 and $b <= 57) {
        return true;
    }
    return $b == 45 or $b == 95 or $b == 46 or $b == 126;
}

# hexByte renders one byte as two uppercase hex digits.
func hexByte(b as int) {
    def digits as string init "0123456789ABCDEF";
    def hi as int init $b // 16;
    def lo as int init $b % 16;
    return strings.substring($digits, $hi, $hi + 1) + strings.substring($digits, $lo, $lo + 1);
}

# urlEncode percent-encodes a string for an `application/x-www-form-urlencoded`
# value: unreserved bytes stay, a space becomes `+`, and every other byte
# becomes `%XX` (over the value's UTF-8 bytes).
func urlEncode(s as string) {
    def raw as bytes init convert.bytesFromString($s, "utf-8");
    def out as string init "";
    def i as int init 0;
    while ($i < len($raw)) {
        def b as int init $raw[$i];
        if (isUnreserved($b)) {
            $out = $out + convert.fromCodepoint($b);
        } elseif ($b == 32) {
            $out = $out + "+";
        } else {
            $out = $out + "%" + hexByte($b);
        }
        $i = $i + 1;
    }
    return $out;
}

# formBody builds the Gotify message form (title / message / priority).
func formBody(title as string, message as string, priority as int) {
    def body as string init "title=" + urlEncode($title);
    $body = $body + "&message=" + urlEncode($message);
    return $body + "&priority=" + convert.toString($priority);
}

# --- push (exported) -----------------------------------------------

# push sends a notification (title / message / priority) to the Gotify server
# and returns the `http.Response` (2xx on success; a bad token surfaces as a
# 4xx value, not a crash).
export func push(cfg as Config, title as string, message as string, priority as int) {
    def headers as map of string to string init {"X-Gotify-Key": $cfg.token};
    def body as string init formBody($title, $message, $priority);
    return http.post($cfg.url + "/message", "application/x-www-form-urlencoded", $body, $headers);
}
