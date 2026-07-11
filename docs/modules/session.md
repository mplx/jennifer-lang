# `session` - server-side sessions on memcached

Import with `import "session.j" as session;`. A server-side **session store** on
the [`memcache`](memcache.md) module - the canonical memcached use. A session is
a `map of string to string` held under a `sess:ID` key with a sliding TTL, so it
expires on its own when idle. It threads three pieces together: `memcache`
(store + TTL), [`uuid`](../libraries/uuid.md) (the session ID), and
[`json`](../libraries/json.md) (encode the map). Because it builds on `memcache`
(which uses `net`), this module needs the default **`jennifer`** binary.

```jennifer
import "session.j" as session;
import "memcache.j" as memcache;

def mc as memcache.Session init memcache.connect(memcache.Options{
    host: "127.0.0.1", port: 11211});

def id as string init session.create($mc, 1800);        # 30-minute session
def data as map of string to string init session.load($mc, $id);
$data["user"] = "ada";
session.save($mc, $id, $data, 1800);
```

Runnable: [`examples/modules/session_demo.j`](https://github.com/mplx/jennifer-lang/blob/main/examples/modules/session_demo.j).

## Surface

Each call takes a live `memcache.Session` and a session `id`; `ttl` is the
expiry in **seconds**.

| Call                                       | Notes                                                             |
| ------------------------------------------ | ----------------------------------------------------------------- |
| `session.create(mc, ttl)`                  | Mint a new ID, store an empty session; returns the `id` (string). |
| `session.load(mc, id)`                     | The session's `map of string to string`, or an empty map when absent / expired. |
| `session.save(mc, id, data, ttl)`          | Write the data map and re-arm the expiry to `ttl` seconds.        |
| `session.touch(mc, id, ttl)`               | Re-arm the expiry without rewriting the data; returns whether it still existed. |
| `session.destroy(mc, id)`                  | Remove the session; returns whether it existed.                   |

The typical request cycle: `load` the session at the start, read / write the
map, `save` it at the end (which also slides the expiry). A quiet request that
only needs to keep the session alive can `touch` instead of `save`.

## Storage format

The data map is stored as **base64-wrapped JSON** under `sess:ID`. The base64
wrap keeps the cached value pure ASCII, so a session value with any UTF-8 text
(`"name": "José"`) round-trips exactly - memcached's value read is byte-exact
for ASCII, and base64 makes every value ASCII on the wire. The trade-off is that
the cached blob is not human-readable and not interchangeable with another
framework's session format; a PHP-session-compatible layout is a separate
follow-on.

## Caveats

- **Volatile.** memcached evicts under memory pressure and loses everything on
  restart, so a session can vanish before its TTL. Treat sessions as a cache of
  soft state (a shopping cart, a wizard step), not a store of record. Anything
  that must survive belongs in a database.
- **Session IDs are UUID v4 from a non-crypto RNG** (see the
  [`uuid`](../libraries/uuid.md) module - randomness draws from `math`'s
  seedable source). That is fine for a cache key; a session ID used as an
  authentication token wants a cryptographic source, which lands with the
  planned `crypto` library.
- **String values only.** A session is a `map of string to string`; encode
  richer values (numbers, nested data) yourself, e.g. via `json` or
  `convert.toString`.

## See also

- [memcache.md](memcache.md) - the cache client this builds on.
- [uuid.md](../libraries/uuid.md) - the session-ID source.
- [json.md](../libraries/json.md) - the map serialization.
- [modules/index.md](index.md) - the module catalog and import rules.
