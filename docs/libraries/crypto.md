# `crypto` - security primitives

Enable with `use crypto;`. The home for the security-sensitive
primitives that need a cryptographically secure random source or a
timing-safe operation: crypto-grade randomness, constant-time
comparison, and the two standard key-derivation functions.

Digests (MD5 / SHA-*) and the keyed-hash MAC (`hash.hmac`) live in
[`hash`](hash.md); non-cryptographic checksums live in [`crc`](crc.md).
`crypto` is the layer above them - the operations where predictability
or a timing side channel is a vulnerability.

Everything here is Go standard library only (`crypto/rand`,
`crypto/subtle`, `crypto/hkdf`, `crypto/pbkdf2`), so the library adds no
dependency and works on **both binaries** (`jennifer` and
`jennifer-tiny`).

```jennifer
use io;
use crypto;
use encoding;

def token as bytes init crypto.randBytes(32);
io.printf("session token: %s\n", encoding.toText($token, "base64-url"));
```

## Crypto-grade randomness

Unlike [`math`](math.md)'s `rand` family (fast, seedable, and therefore
**predictable** - a PRNG for simulations and sampling), these draw from
the operating system's cryptographically secure source. They are
**not** seedable: there is no reproducible sequence, by design.

| Call                   | Returns | Notes                                                        |
| ---------------------- | ------- | ------------------------------------------------------------ |
| `crypto.randBytes(n)`  | `bytes` | `n` secure random bytes (`0 <= n <= 64 MiB`; `0` yields empty bytes). |
| `crypto.randInt(lo, hi)` | `int` | A uniform int in the **inclusive** range `[lo, hi]`.        |

`crypto.randInt` has the same shape as [`math.randInt`](math.md) but is
unpredictable and unseedable - the drop-in when the value guards
something (a token, a nonce, a shuffle of secret material). It uses
rejection sampling, so the distribution is exactly uniform with no
modulo bias, across the full `int` range.

```jennifer
use crypto;

def dieRoll as int init crypto.randInt(1, 6);        # secure, unbiased
def key as bytes init crypto.randBytes(16);          # a 128-bit key
```

Since this library landed, [`uuid`](uuid.md) draws its v4 / v7
randomness here, so `uuid.generate("v4")` is unguessable and safe to
hand a client as a session id or bearer token.

## Constant-time comparison

`crypto.hmacEqual(a, b)` compares two `bytes` values for equality
**without an early-out**, so an attacker cannot recover a secret byte by
byte from response-timing differences. Use it to check a computed MAC
against a supplied one, rather than `==` (which short-circuits on the
first differing byte). Unequal-length inputs return `false`.

```jennifer
use crypto;
use hash;
use convert;

def key as bytes init convert.bytesFromString("secret", "utf-8");
def body as bytes init convert.bytesFromString("payload", "utf-8");
def expected as bytes init hash.hmac($key, $body, "sha256");
def supplied as bytes init requestSignature();      # from the caller

if (crypto.hmacEqual($expected, $supplied)) {
    # signature verified
}
```

## Key derivation

Two functions turn a secret into keying material. Both output `bytes`.

| Call                                          | Returns | Notes                                            |
| --------------------------------------------- | ------- | ------------------------------------------------ |
| `crypto.hkdf(secret, salt, info, length, algo)`     | `bytes` | HKDF (RFC 5869): expand a **high-entropy** secret into `length` bytes. `salt` / `info` may be empty bytes. |
| `crypto.pbkdf(password, salt, iterations, keyLen, algo)` | `bytes` | PBKDF2 (RFC 8018): stretch a **low-entropy** password over `iterations` rounds against `salt`. |

`algo` names the PRF hash: `"sha1"`, `"sha256"`, or `"sha512"` (the
`hash` library's names, minus `md5` - too weak to derive a key with). An
unknown algorithm is a positioned error. Use `"sha256"` unless you are
matching an external protocol: SCRAM-SHA-1 (MongoDB, XMPP) requires
`"sha1"`.

- **HKDF** derives one or more subkeys from material that is already
  strong (a shared secret, a master key). It is fast; do not use it to
  hash passwords.
- **PBKDF2** is for passwords: the `iterations` count makes each guess
  expensive, so pick the largest value the deployment can afford. The
  total work — `ceil(keyLen / hashLen) × iterations` — is capped so an
  untrusted parameter (e.g. a hostile SCRAM server's `i=`) cannot pin a
  core for days; a single-block key allows up to ~10⁸ iterations (worst
  case ~20 s), and a larger `keyLen` proportionally fewer. `keyLen` is
  also capped at 1 MiB, and `hkdf`'s `length` likewise — real key
  material is tens of bytes. It unblocks SASL SCRAM (see the
  [`sasl`](../modules/sasl.md) module) and password-based key wrapping.

```jennifer
use crypto;
use convert;
use encoding;

def password as bytes init convert.bytesFromString("correct horse", "utf-8");
def salt as bytes init crypto.randBytes(16);
def derived as bytes init crypto.pbkdf($password, $salt, 600000, 32, "sha256");
io.printf("derived key: %s\n", encoding.toText($derived, "hex"));
```

> **Why `pbkdf`, not `pbkdf2`?** A Jennifer method name is letters-only,
> so the "2" cannot appear - the same rule that makes uuid's version a
> string argument (`uuid.generate("v4")`). The scheme is still PBKDF2;
> only the name drops the digit.

### Password *hashing* is not here

Storing passwords for later verification wants a memory-hard function
(Argon2id, bcrypt, scrypt), which needs a dependency outside the
standard library. That is deliberately out of scope for `crypto`; use
PBKDF2 only where an interoperable KDF is required (e.g. SCRAM), not as
a general password store.

## Errors

Every function validates argument kinds and counts and raises a
positioned runtime error on misuse (wrong type, negative `randBytes`
length, `lo > hi`, non-positive `iterations` / `keyLen` / `length`). The
secure random source is assumed always available on the supported
platform; an impossible failure aborts rather than returning weak bytes.
