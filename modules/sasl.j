# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>

/**
 * The crypto-free SASL authentication mechanisms as pure base64 encoders,
 * shared by the mail clients (`smtp` / `pop` / `imap`). These format the client
 * tokens; the protocol clients run the mechanism-specific wire dialogue (SMTP
 * `AUTH`, IMAP `AUTHENTICATE`, POP3 `AUTH`) around them. No networking and no
 * crypto, so this module is TinyGo-clean and runs on both binaries. `plain` and
 * `login*` are username / password mechanisms; `bearer` builds the SASL XOAUTH2
 * response from an OAuth2 bearer token (the "use a token" half of OAuth2 - the
 * token itself comes from an OAuth2 client). Named `bearer`, not `xoauth2`,
 * because a Jennifer method name is letters-only; the wire mechanism name
 * "XOAUTH2" is a string the mail client sends. The challenge-response
 * mechanisms (`SCRAM`, `CRAM-MD5`) join this module once `crypto` lands.
 * @module sasl
 * @example
 * import "sasl.j" as sasl;
 * def token as string init sasl.plain("me@example.com", "secret");
 * def xo as string init sasl.bearer("me@gmail.com", accessToken);
 */
use convert;
use encoding;

# baseEncode is base64 of a string's UTF-8 bytes.
func baseEncode(s as string) {
    return encoding.toText(convert.bytesFromString($s, "utf-8"), "base64");
}

# ctrlA returns the SASL control-A (0x01) separator. Jennifer has no `\x01`
# string escape, so it is built from a one-byte `bytes`.
func ctrlA() {
    def b as bytes;
    $b[] = 1;
    return convert.stringFromBytes($b, "utf-8");
}

/**
 * Build the SASL PLAIN response: base64 of "\0user\0pass".
 * @param user {string} the login username
 * @param pass {string} the login password
 * @return {string} the base64-encoded PLAIN token
 */
export func plain(user as string, pass as string) {
    return baseEncode("\0" + $user + "\0" + $pass);
}

/**
 * Build the username step of SASL LOGIN (a server sends the "Username:" prompt).
 * @param user {string} the login username
 * @return {string} the base64-encoded username
 */
export func loginUser(user as string) {
    return baseEncode($user);
}

/**
 * Build the password step of SASL LOGIN (a server sends the "Password:" prompt).
 * @param pass {string} the login password
 * @return {string} the base64-encoded password
 */
export func loginPass(pass as string) {
    return baseEncode($pass);
}

/**
 * Build the SASL XOAUTH2 response for an OAuth2 bearer token:
 * base64("user=" user 0x01 "auth=Bearer " token 0x01 0x01). This is how Google
 * and Microsoft 365 authenticate mail (both have retired password auth).
 * @param user {string} the account username
 * @param token {string} the OAuth2 bearer access token
 * @return {string} the base64-encoded XOAUTH2 response
 */
export func bearer(user as string, token as string) {
    def sep as string init ctrlA();
    def raw as string init "user=" + $user + $sep + "auth=Bearer " + $token + $sep + $sep;
    return baseEncode($raw);
}
