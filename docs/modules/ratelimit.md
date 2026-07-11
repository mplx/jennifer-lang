# `ratelimit` - a fixed-window rate limiter on memcached

Import with `import "ratelimit.j" as ratelimit;`. A **fixed-window rate
limiter** on the [`memcache`](memcache.md) module - the sharpest use of
memcached's distinctive strength: **atomic `incr` plus a per-key TTL**. Each key
(a client IP, a user, an API token) counts hits in a time window; the counter is
armed with the window's expiry when it is first created, so it resets on its own
when the window ends - there is nothing to reap. Because it builds on `memcache`
(which uses `net`), this module needs the default **`jennifer`** binary.

> **On `jennifer-tiny`:** "needs the default `jennifer` binary" refers to the
> **stock** tiny build, which ships without a network driver - not a TinyGo
> limitation. A `jennifer-tiny` rebuilt with a network stack runs this module
> too; see the
> [note on `net` and TinyGo](../technical/tinygo.md#net-on-tinygo-is-a-build-choice-not-a-hard-limit).

```jennifer
import "ratelimit.j" as ratelimit;
import "memcache.j" as memcache;

def mc as memcache.Session init memcache.connect(memcache.Options{
    host: "127.0.0.1", port: 11211});

# 100 requests per 60 seconds, per client IP
if (ratelimit.allow($mc, "ip:203.0.113.7", 100, 60)) {
    # ... serve the request ...
} else {
    # ... reject (e.g. HTTP 429) ...
}
```

Runnable: [`examples/modules/ratelimit_demo.j`](https://github.com/mplx/jennifer-lang/blob/main/examples/modules/ratelimit_demo.j).

## Surface

| Call                                    | Notes                                                              |
| --------------------------------------- | ------------------------------------------------------------------ |
| `ratelimit.allow(mc, key, limit, window)` | Record one hit; `true` if within `limit` for the current `window` (seconds). |
| `ratelimit.remaining(mc, key, limit)`   | Hits left in the current window (the full `limit` when untouched, `0` once exhausted). |

## How the window works

`allow` does an atomic `incr` on the key. The window starts at the **first**
hit: an absent counter is created (via `add`) carrying the window's TTL, and
because a later `incr` does not re-arm the expiry, the counter dies exactly
`window` seconds after that first hit - a clean fixed window with no background
cleanup. The `incr`-then-`add` pair also closes the create race: if two callers
both find the counter absent, only one `add` wins and the loser re-`incr`s, so
no hit is lost.

```jennifer
def ok as bool init ratelimit.allow($mc, "user:ada", 5, 60);   # 5 per minute
io.printf("%d left\n", ratelimit.remaining($mc, "user:ada", 5));
```

`allow` returns `true` for the first `limit` hits in a window and `false`
afterwards; the counter keeps rising while denied, and `remaining` reports `0`,
until the window expires and the budget refills.

## Out of scope

- **Fixed window only.** The count resets at the window boundary, so up to
  `2 * limit` hits can land across two adjacent windows in the worst case.
  A **sliding window** or **token bucket** (smoother, burst-tolerant) is a
  later refinement.
- **Not a distributed clock.** The window is per key in one memcached; it does
  not coordinate wall-clock alignment across instances.
- **Volatile.** memcached can evict a counter under memory pressure, which
  resets that key's window early - acceptable for throttling, not for billing.

## See also

- [memcache.md](memcache.md) - the `incr` / `add` / `get` primitives this uses.
- [modules/index.md](index.md) - the module catalog and import rules.
