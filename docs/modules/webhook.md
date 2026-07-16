# `webhook` - HMAC-signed webhooks

Import with `import "webhook.j" as webhook;`. Sign and verify webhook deliveries
the GitHub / Stripe way - the **`X-Hub-Signature-256`** convention. A sender
computes `sha256=<hex>`, the hex HMAC-SHA256 of the exact request body keyed by a
shared secret, and puts it in a header; the receiver recomputes it over the body
it got and compares, confirming the delivery is authentic and untampered.

`sign` / `verify` are pure and run on **both** binaries; `send` POSTs a signed
payload and needs the default **`jennifer`** binary (`net` via `http`).

```jennifer
import "webhook.j" as webhook;

def sig as string init webhook.sign("{\"event\":\"push\"}", "topsecret");
# -> "sha256=..."  (put this in the X-Hub-Signature-256 header)

def ok as bool init webhook.verify("{\"event\":\"push\"}", $sig, "topsecret");
```

Runnable: [`examples/modules/webhook_demo.j`](https://github.com/jennifer-language/jennifer/blob/main/examples/modules/webhook_demo.j).

## Functions

| Call | Returns | Notes |
| ---- | ------- | ----- |
| `webhook.sign(payload, secret)` | `string` | `sha256=` + hex HMAC-SHA256 of `payload` keyed by `secret`. |
| `webhook.verify(payload, signature, secret)` | `bool` | True if `signature` matches (constant-time compare). |
| `webhook.send(url, payload, secret)` | `http.Response` | POST `payload` to `url` with the signature header set. Needs the default binary. |

The signature covers the **raw body bytes** - sign and verify the exact string
you send / receive, before any parsing. A receiver that re-serializes the body
first can compute a different signature and reject a valid delivery.

`verify` uses a constant-time comparison, so a check does not leak via timing
how many leading characters of the signature were correct. It returns `false`
(never throws) for a wrong secret, a tampered payload, or a malformed signature.

## Sending

`webhook.send` posts the payload as `application/json` with the
`X-Hub-Signature-256` header, and returns the receiver's `http.Response`
(status / headers / body). Reading the result needs `import "http.j"` for the
type:

```jennifer
import "webhook.j" as webhook;
import "http.j" as http;

def r as http.Response init webhook.send("https://example.com/hook",
    "{\"event\":\"push\"}", "topsecret");
io.printf("delivered: %d\n", $r.status);
```

A non-2xx status comes back as a value to branch on; a network failure throws a
positioned `http` / `net` error (wrap in `try` / `catch`).

## Notes and scope

- **SHA-256, `X-Hub-Signature-256`.** This is the modern GitHub convention. The
  legacy `X-Hub-Signature` (SHA-1) header is not emitted; sign with SHA-256.
- **The secret is the shared key** - store it like a password and distribute it
  over a secure channel. Randomness for a new secret is out of scope (use a
  cryptographic source; `math.rand*` is not crypto-grade).
- **Content type is `application/json`** for `send`. For a different body type,
  `sign` the payload yourself and post it with your own headers via `http`.

## See also

- [hash.md](../libraries/hash.md) - the `hmac` primitive the signature is built on.
- [http.md](http.md) - the client `send` posts through.
- [totp.md](totp.md) - the other `hash.hmac`-based module (2FA codes).
- [modules/index.md](index.md) - the module catalog and import rules.
