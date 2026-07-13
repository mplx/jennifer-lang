# `web` - a small HTTP framework

`import "web.j" as web;`

An ergonomic HTTP framework over the [`httpd`](../libraries/httpd.md) server
engine. Register routes against handler methods by name, then `web.run` owns
the accept loop, matches each request, and dispatches to the handler - so a web
app reads as a set of handlers plus a route table, not a hand-written server
loop.

Needs the default `jennifer` binary (the `httpd` engine is net-backed; the
constrained `jennifer-tiny` has no network stack).

## Handlers

A handler is a top-level method taking a `web.Context`:

```jennifer
import "web.j" as web;

func showUser(ctx as web.Context) {
    web.text($ctx, 200, "user " + web.param($ctx, "id"));
}

def app as web.App init web.new();
$app = web.get($app, "/users/:id", "showUser");
web.run($app, ":8080");
```

Because Jennifer has no first-class functions, a route stores the handler's
**name** and dispatch happens by name via `meta.callMain` - the primitive that
lets the framework module reach a handler defined in the entry program across
the module boundary. You never register a function value; you register a name,
and `web.get`/`web.route` check at registration time that the method exists.

## The `web.Context`

Everything a handler needs hangs off the `web.Context` it receives, so it rarely
touches `httpd` directly:

| Call | Returns | |
| ---- | ------- | - |
| `web.method($ctx)` | `string` | Request method. |
| `web.path($ctx)` | `string` | Request path. |
| `web.param($ctx, name)` | `string` | A captured `:param` (`""` if none). |
| `web.query($ctx, name)` | `string` | A query-string parameter. |
| `web.header($ctx, name)` | `string` | A request header. |
| `web.body($ctx)` | `bytes` | The raw request body (binary-safe). |
| `web.bodyJson($ctx)` | `json.Value` | The request body decoded as JSON. |
| `web.form($ctx)` | `map of string to string` | The `x-www-form-urlencoded` body, decoded. |
| `web.formValue($ctx, name)` | `string` | One form field (`""` if absent). |
| `web.remoteAddr($ctx)` | `string` | The client's `host:port`. |
| `web.setHeader($ctx, name, value)` | `null` | Set a response header (before responding). |
| `web.respond($ctx, status, body)` | `null` | Send a response (body is a string). |
| `web.text($ctx, status, body)` | `null` | Respond with `text/plain`. |
| `web.html($ctx, status, body)` | `null` | Respond with `text/html`. |
| `web.sendJson($ctx, status, doc)` | `null` | Respond with `application/json` from a `json.Value`. |
| `web.redirect($ctx, status, location)` | `null` | Redirect (301/302/303/307/308) with a `Location` header. |
| `web.serveFile($ctx, path)` | `null` | Respond with a file from disk. |
| `web.serveDir($ctx, root)` | `null` | Serve static files from a directory root. |

## Cookies

Pure HTTP-header work over `httpd`, no extra dependency:

| Call | Returns | |
| ---- | ------- | - |
| `web.cookie($ctx, name)` | `string` | The request cookie's value (`""` if absent). |
| `web.setCookie($ctx, name, value, opts)` | `null` | Emit a `Set-Cookie` response header. |

`opts` is a `web.CookieOptions` - `path`, `domain`, `maxAge` (int seconds; `0`
omits the attribute, negative expires the cookie now), `httpOnly`, `secure`,
`sameSite` (`"Lax"` / `"Strict"` / `"None"` / `""`). A zero-value struct is a
plain session cookie. Several `setCookie` calls emit several `Set-Cookie`
headers (they are not collapsed).

## Sessions

`web` owns only the session **id cookie**; the session **data** lives in a store
the **app** owns, so `web` forces no store or network dependency (it never
imports `session` / `memcache`). That keeps a `web` app that uses no sessions at
`httpd` + `meta` + `json` + collections.

| Call | Returns | |
| ---- | ------- | - |
| `web.sessionId($ctx, cookieName)` | `string` | The request's session id, minting a new UUID + `HttpOnly` / `SameSite=Lax` / path-`/` cookie on first use. |

Call `web.sessionId` once per request and pair the id with your own store - for
multi-process serving the [`session`](session.md) module over
[`memcache`](memcache.md); for a single process anything the app holds:

```jennifer
# The app owns the store; web just resolves the id cookie.
func profile(ctx as web.Context) {
    def id as string init web.sessionId($ctx, "sid");
    def data as map of string to string init session.load($store, $id);   # app's store
    # ... read / write $data ...
    session.save($store, $id, $data, 3600);
    web.text($ctx, 200, "ok\n");
}
```

Stateless signed-cookie sessions (data in the cookie, no store) wait on a
`crypto` library for real HMAC - see [horizon.md](../horizon.md).

## Registering routes

Each registrar returns a **new** `App` (value semantics), so build the app by
threading it through:

| Call | |
| ---- | - |
| `web.new()` | An empty app. |
| `web.get($app, pattern, handler)` | Register a `GET` route. |
| `web.post` / `web.put` / `web.patch` / `web.delete` | The other verbs. |
| `web.route($app, method, pattern, handler)` | The general form. |
| `web.before($app, handler)` | Add a middleware (runs before each route handler). |
| `web.notFound($app, handler)` | A custom handler for unmatched requests (default: a plain 404). |

Patterns are `/`-separated. A segment beginning with `:` captures a single
parameter: `/users/:id/posts/:pid` matches `/users/7/posts/9` and captures `id`
= `7`, `pid` = `9`. A trailing segment beginning with `*` is a **wildcard** that
captures the entire remainder (joined by `/`, possibly empty): `/files/*path`
matches `/files/css/app.css` with `path` = `css/app.css`, and `/*path` is a
catch-all - a natural **SPA fallback**. Read either with `web.param($ctx, name)`.

The first matching route wins, so register specific routes **before** a wildcard
(a greedy `/*path` registered first would swallow everything). A wildcard is only
special as the last pattern segment.

```jennifer
$app = web.get($app, "/static/*path", "serveStatic");   # nested static files
$app = web.get($app, "/*page", "spaIndex");              # fallback, registered last
```

## Middleware

A middleware is a handler that runs before the route handler and returns a
`bool`: `true` to continue, or **respond and return `false`** to halt (e.g. an
auth gate):

```jennifer
func requireKey(ctx as web.Context) {
    if (web.header($ctx, "X-Api-Key") == "secret") {
        return true;
    }
    web.text($ctx, 401, "unauthorized\n");
    return false;
}

$app = web.before($app, "requireKey");
```

Every request is answered exactly once: if a handler throws or forgets to
respond, the framework sends a `500` so the connection never hangs.

## Authentication

`web` parses the incoming `Authorization` header; **checking** the credentials
(against your user store) and sending the `401` challenge stay app code.

| Call | Returns | |
| ---- | ------- | - |
| `web.basicAuth($ctx)` | `BasicCredentials` | Decode `Authorization: Basic base64(user:pass)`. |
| `web.bearerToken($ctx)` | `string` | The token from `Authorization: Bearer <token>` (`""` if absent). |

`BasicCredentials` is `{ user, password, present }`; `present` is false when the
header was missing or malformed. A Basic-auth gate is a middleware:

```jennifer
func requireLogin(ctx as web.Context) {
    def cred as web.BasicCredentials init web.basicAuth($ctx);
    if ($cred.present and checkUser($cred.user, $cred.password)) {
        return true;
    }
    web.setHeader($ctx, "WWW-Authenticate", "Basic realm=\"app\"");   # the 401 challenge is a two-liner
    web.text($ctx, 401, "unauthorized\n");
    return false;
}
$app = web.before($app, "requireLogin");
```

For **bearer** tokens, `web.bearerToken($ctx)` extracts the token; validate it
yourself (an opaque lookup, or `jwt.verify` once the `jwt` module lands). Client
-side auth (sending `Authorization`) lives in the [`rest`](rest.md) module
(`rest.basic` / `rest.bearer`). **Digest** auth is not supported (a legacy,
challenge/nonce scheme; use Basic over TLS or a bearer token).

## CSRF

Stateless, HMAC-signed double-submit tokens. `web` holds no secret or session
state - the **app supplies a secret** (a stable per-deployment string) and opts
in with a middleware. A token is `<random>.<hmac(secret, random)>`, minted into
the `csrf` cookie and echoed by the client; a request is accepted only when the
submitted token equals the cookie *and* its signature verifies, so a forger
without the secret cannot mint one.

| Call | Returns | |
| ---- | ------- | - |
| `web.csrfToken($ctx, secret)` | `string` | Mint a token, set the `csrf` cookie, return it for the form / an `X-CSRF-Token` header. |
| `web.csrfCheck($ctx, secret)` | `bool` | True when the request carries a valid token. |

Mint the token in the GET handler that renders a form; guard the unsafe methods
with a middleware:

```jennifer
def const CSRF as string init os.getEnv("CSRF_SECRET");   # a stable secret

func showForm(ctx as web.Context) {
    def token as string init web.csrfToken($ctx, CSRF);
    web.html($ctx, 200, "<form method=post><input type=hidden name=csrf value=" + $token + ">...</form>");
}

func guardCsrf(ctx as web.Context) {
    def m as string init web.method($ctx);
    if ($m == "GET" or $m == "HEAD") {
        return true;                       # safe methods
    }
    if (web.csrfCheck($ctx, CSRF)) {
        return true;
    }
    web.text($ctx, 403, "CSRF check failed\n");
    return false;
}
$app = web.before($app, "guardCsrf");
```

The submitted token is read from the `X-CSRF-Token` header (for JSON / fetch
clients) or the `csrf` form field (for HTML forms).

## CORS

`web.cors($app, opts) -> App` sets a cross-origin policy for the whole app.
When it is set, the serve loop adds the `Access-Control-*` headers to **every**
response and answers a preflight `OPTIONS` request with a `204` before routing -
so CORS is a one-line, app-wide policy, not something each handler repeats.

```jennifer
def opts as web.CorsOptions;
$opts.allowOrigin = "*";
$opts.allowMethods = "GET, POST, PUT, DELETE, OPTIONS";
$opts.allowHeaders = "Content-Type, Authorization";
$app = web.cors($app, $opts);
```

`opts` is a `web.CorsOptions` - `allowOrigin` (`"*"` or a specific origin; `""`
leaves CORS off), `allowMethods`, `allowHeaders`, `allowCredentials` (bool), and
`maxAge` (int seconds, `0` omits it). A zero-value struct is CORS off, so
`web.new()` starts with no policy.

## Caching

**Static files are already cached:** `web.serveFile` rides Go's file server,
which sets `ETag` / `Last-Modified` and answers `If-None-Match` /
`If-Modified-Since` (and `Range`) on its own. **Plain cache headers** are a
one-liner - `web.setHeader($ctx, "Cache-Control", "max-age=3600")` - so there is
no wrapper for them.

For a **dynamic** response, `web.etag($ctx, tag) -> bool` handles the conditional
GET: it sets the `ETag` header and, if the request's `If-None-Match` matches,
answers `304 Not Modified` and returns `true` so the handler stops before
sending the body. `tag` is your choice of validator - a content hash (via
`hash`), a database row version, an mtime - so `web` needs no hashing of its own.

```jennifer
func page(ctx as web.Context) {
    def body init render();
    def tag init hashOf($body);           # your validator
    if (web.etag($ctx, $tag)) {
        return;                            # 304 sent; skip the body
    }
    web.text($ctx, 200, $body);
}
```

The match is a simple exact / `*` comparison; the RFC 7232 comma-list and weak
(`W/`) tag forms are not parsed.

## Serving

| Call | |
| ---- | - |
| `web.run($app, addr)` | Listen on `addr` and serve forever (blocks; interrupt to stop). |
| `web.serveOn($app, srv)` | Serve on an already-listening `httpd.Server` you hold - so you can shut it down (or serve from a `spawn`). |

`web.run` is the common path. Use `web.serveOn` when you want the server handle,
e.g. to shut down from another task:

```jennifer
def srv as httpd.Server init httpd.listen(":8080");
def worker as task of null init spawn { web.serveOn($app, $srv); };
# ... later ...
httpd.shutdown($srv);
task.wait($worker);
```

## Running with `jennifer serve`

`jennifer serve app.j` runs a web app; `--watch` restarts it whenever the
entry file changes - a Hugo-style edit / reload loop:

```sh
jennifer serve app.j            # run the app
jennifer serve app.j --watch    # reload on change
```

`--watch` is not web-specific: it re-runs *any* program on every change to the
entry file, so it doubles as an autorun / edit-and-rerun loop for plain
scripts too. See [the `serve` command](../technical/cli.md#the-serve-command).

## See also

- [`httpd`](../libraries/httpd.md) - the server engine `web` is built on.
- [`http`](http.md) - the HTTP client module (talk to other servers).
- [`json`](../libraries/json.md) - encode response bodies for `web.sendJson`.
- [`rest`](rest.md), [`session`](session.md), [`ratelimit`](ratelimit.md) -
  companions for a fuller serving stack.
