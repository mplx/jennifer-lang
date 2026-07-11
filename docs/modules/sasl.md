# `sasl` - SASL authentication encoders

Import with `import "sasl.j" as sasl;`. The crypto-free SASL mechanisms as
pure base64 encoders, shared by the mail clients (`smtp` / `pop` / `imap`).
These format the client tokens; the protocol clients run the
mechanism-specific wire dialogue around them. **No networking and no crypto**,
so this module is TinyGo-clean and runs on either binary.

```jennifer
import "sasl.j" as sasl;

def p as string init sasl.plain("me@example.com", "secret");     # SASL PLAIN
def b as string init sasl.bearer("me@gmail.com", accessToken);   # SASL XOAUTH2
```

Runnable: [`examples/modules/sasl_demo.j`](https://github.com/mplx/jennifer-lang/blob/main/examples/modules/sasl_demo.j)
(and it is exercised end to end by the mail clients' demos, e.g.
[`smtp_demo.j`](https://github.com/mplx/jennifer-lang/blob/main/examples/modules/smtp_demo.j)).

## Surface

| Call                        | Returns  | Notes                                                             |
| --------------------------- | -------- | ----------------------------------------------------------------- |
| `sasl.plain(user, pass)`    | `string` | SASL PLAIN: base64 of `"\0user\0pass"`.                           |
| `sasl.loginUser(user)`      | `string` | SASL LOGIN step 1: base64 of the username.                        |
| `sasl.loginPass(pass)`      | `string` | SASL LOGIN step 2: base64 of the password.                        |
| `sasl.bearer(user, token)`  | `string` | SASL XOAUTH2: an OAuth2 bearer-token response.                     |

## XOAUTH2 (the "use a token" half of OAuth2)

`sasl.bearer(user, token)` builds the SASL **XOAUTH2** initial response -
`base64("user=" user 0x01 "auth=Bearer " token 0x01 0x01)` - which is how
**Google** and **Microsoft 365** authenticate mail now that both have retired
password auth. The mail clients accept it via `Options.auth = "xoauth2"` (with
the access token in `pass`):

```jennifer
def opts as smtp.Options init smtp.Options{host: "smtp.gmail.com", port: 587,
    security: "starttls", clientName: "me.example", user: "me@gmail.com",
    pass: accessToken, auth: "xoauth2"};
smtp.send($opts, "me@gmail.com", ["you@example.com"], $message);
```

The function is named `bearer`, not `xoauth2`, because a Jennifer method name
is letters-only (no digit); the wire mechanism name `"XOAUTH2"` is a string the
client sends. Getting the **token itself** is the other half of OAuth2 - the
job of the generic `oauth` client (planned, M18.7.3).

## Out of scope

- **Crypto-free mechanisms only.** The challenge-response mechanisms
  (`SCRAM-SHA-256`, `CRAM-MD5`) need HMAC / PBKDF2 and join this module when
  the `crypto` library (M20.1) lands.
- **Encoders, not transport.** `sasl` formats tokens; the SMTP `AUTH`, IMAP
  `AUTHENTICATE`, and POP3 `AUTH` dialogue lives in the respective clients.

## See also

- [smtp.md](smtp.md) / [pop.md](pop.md) / [imap.md](imap.md) - the mail
  clients that consume these encoders.
- [encoding.md](../libraries/encoding.md) - the base64 codec `sasl` builds on.
- [modules/index.md](index.md) - the module catalog and import rules.
