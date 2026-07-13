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
| `web.body($ctx)` | `bytes` | The request body. |
| `web.setHeader($ctx, name, value)` | `null` | Set a response header (before responding). |
| `web.respond($ctx, status, body)` | `null` | Send a response (body is a string). |
| `web.text($ctx, status, body)` | `null` | Respond with `text/plain`. |
| `web.sendJson($ctx, status, doc)` | `null` | Respond with `application/json` from a `json.Value`. |
| `web.serveFile($ctx, path)` | `null` | Respond with a file from disk. |

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

Patterns are `/`-separated. A segment beginning with `:` captures a parameter:
`/users/:id/posts/:pid` matches `/users/7/posts/9` and captures `id` = `7`,
`pid` = `9`. The first matching route wins.

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
