# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>

/**
 * An ergonomic HTTP framework over the `httpd` server engine. Register routes
 * against handler methods by name, then `web.run` owns the accept loop, matches
 * each request, and dispatches to the handler. Since Jennifer has no
 * first-class functions, dispatch is by method name via `meta.callMain` (which
 * reaches the entry program's handlers across the module boundary); a handler
 * is `func name(ctx as web.Context) { ... }`. Routing supports `:param`
 * segments, a middleware chain, and a custom not-found handler. Response and
 * request helpers hang off the `web.Context` so a handler rarely reaches for
 * `httpd` directly. Needs the default `jennifer` binary (the `httpd` engine is
 * net-backed; the constrained `jennifer-tiny` has no network stack).
 * @module web
 * @example
 * def app as web.App init web.new();
 * $app = web.get($app, "/hello/:name", "hello");
 * web.run($app, ":8080");
 */

use httpd;
use meta;
use json;
use lists;
use maps;
use strings;
use convert;
use uuid;

/**
 * One registered (method, pattern, handler-name) triple. Exported only to
 * satisfy the referential-closure rule (App exposes a list of Route); programs
 * never build one directly - they call web.get / web.route.
 * @field method {string} the HTTP method (e.g. "GET")
 * @field pattern {string} the route pattern, with optional `:param` segments
 * @field handler {string} the name of the handler method to dispatch to
 */
export def struct Route {
    method as string,
    pattern as string,
    handler as string
};

/**
 * The value-semantic router state: the routes, the middleware chain (handler
 * names run before each route), and an optional not-found handler name (empty =
 * the built-in 404). Every registrar returns a fresh App.
 * @field routes {list of Route} the registered routes
 * @field middleware {list of string} handler names run before each route
 * @field notFound {string} the not-found handler name (empty = built-in 404)
 */
export def struct App {
    routes as list of Route,
    middleware as list of string,
    notFound as string
};

/**
 * What a handler receives: the underlying request handle plus the path
 * parameters captured from the matched route (`:id` -> params["id"]).
 * @field req {httpd.Request} the underlying request handle
 * @field params {map of string to string} the captured path parameters
 */
export def struct Context {
    req as httpd.Request,
    params as map of string to string
};

# Match is the internal result of routing a request. Private: it never appears
# in an exported signature.
def struct Match {
    found as bool,
    handler as string,
    params as map of string to string
};

# --- registration ----------------------------------------------------------

/**
 * Return an empty App with no routes or middleware.
 * @return {App} a fresh, empty router
 */
export func new() {
    def routes as list of Route init [];
    def mw as list of string init [];
    return App{ routes: $routes, middleware: $mw, notFound: "" };
}

/**
 * Register a handler for a method + pattern, returning a new App.
 * @param app {App} the router to extend
 * @param method {string} the HTTP method to match
 * @param pattern {string} the route pattern, with optional `:param` segments
 * @param handler {string} the name of the handler method
 * @return {App} a new App with the route added
 * @throws {Error} kind "web" when the handler method is not defined
 */
export func route(app as App, method as string, pattern as string, handler as string) {
    if (not meta.definedMain($handler)) {
        throw Error{ kind: "web", message: "web.route: handler not defined: " + $handler, file: "", line: 0, col: 0 };
    }
    def r as Route init Route{ method: $method, pattern: $pattern, handler: $handler };
    def out as App init $app;
    $out.routes = lists.push($out.routes, $r);
    return $out;
}

/**
 * Register a GET handler for a pattern, returning a new App.
 * @param app {App} the router to extend
 * @param pattern {string} the route pattern, with optional `:param` segments
 * @param handler {string} the name of the handler method
 * @return {App} a new App with the route added
 */
export func get(app as App, pattern as string, handler as string) {
    return route($app, "GET", $pattern, $handler);
}

/**
 * Register a POST handler for a pattern, returning a new App.
 * @param app {App} the router to extend
 * @param pattern {string} the route pattern, with optional `:param` segments
 * @param handler {string} the name of the handler method
 * @return {App} a new App with the route added
 */
export func post(app as App, pattern as string, handler as string) {
    return route($app, "POST", $pattern, $handler);
}

/**
 * Register a PUT handler for a pattern, returning a new App.
 * @param app {App} the router to extend
 * @param pattern {string} the route pattern, with optional `:param` segments
 * @param handler {string} the name of the handler method
 * @return {App} a new App with the route added
 */
export func put(app as App, pattern as string, handler as string) {
    return route($app, "PUT", $pattern, $handler);
}

/**
 * Register a PATCH handler for a pattern, returning a new App.
 * @param app {App} the router to extend
 * @param pattern {string} the route pattern, with optional `:param` segments
 * @param handler {string} the name of the handler method
 * @return {App} a new App with the route added
 */
export func patch(app as App, pattern as string, handler as string) {
    return route($app, "PATCH", $pattern, $handler);
}

/**
 * Register a DELETE handler for a pattern, returning a new App.
 * @param app {App} the router to extend
 * @param pattern {string} the route pattern, with optional `:param` segments
 * @param handler {string} the name of the handler method
 * @return {App} a new App with the route added
 */
export func delete(app as App, pattern as string, handler as string) {
    return route($app, "DELETE", $pattern, $handler);
}

/**
 * Register a middleware handler, run before each route handler. A middleware is
 * `func name(ctx as web.Context) { ...; return true; }`: return true to continue
 * to the route handler, or respond and return false to halt.
 * @param app {App} the router to extend
 * @param handler {string} the name of the middleware method
 * @return {App} a new App with the middleware appended to the chain
 * @throws {Error} kind "web" when the middleware method is not defined
 */
export func before(app as App, handler as string) {
    if (not meta.definedMain($handler)) {
        throw Error{ kind: "web", message: "web.before: middleware not defined: " + $handler, file: "", line: 0, col: 0 };
    }
    def out as App init $app;
    $out.middleware = lists.push($out.middleware, $handler);
    return $out;
}

/**
 * Set a custom handler for unmatched requests (default: a plain 404).
 * @param app {App} the router to extend
 * @param handler {string} the name of the not-found handler method
 * @return {App} a new App with the not-found handler set
 * @throws {Error} kind "web" when the handler method is not defined
 */
export func notFound(app as App, handler as string) {
    if (not meta.definedMain($handler)) {
        throw Error{ kind: "web", message: "web.notFound: handler not defined: " + $handler, file: "", line: 0, col: 0 };
    }
    def out as App init $app;
    $out.notFound = $handler;
    return $out;
}

# --- request / response helpers (on a Context) ------------------------------

/**
 * Return the request's HTTP method.
 * @param ctx {Context} the request context
 * @return {string} the HTTP method (e.g. "GET")
 */
export func method(ctx as Context) { return httpd.method($ctx.req); }

/**
 * Return the request's URL path.
 * @param ctx {Context} the request context
 * @return {string} the request path
 */
export func path(ctx as Context) { return httpd.path($ctx.req); }

/**
 * Return a query-string parameter value.
 * @param ctx {Context} the request context
 * @param name {string} the query parameter name
 * @return {string} the parameter value, or "" if absent
 */
export func query(ctx as Context, name as string) { return httpd.query($ctx.req, $name); }

/**
 * Return a request header value.
 * @param ctx {Context} the request context
 * @param name {string} the header name
 * @return {string} the header value, or "" if absent
 */
export func header(ctx as Context, name as string) { return httpd.header($ctx.req, $name); }

/**
 * Return the request body.
 * @param ctx {Context} the request context
 * @return {string} the request body
 */
export func body(ctx as Context) { return httpd.body($ctx.req); }

/**
 * Return a captured path parameter, or "" if the route had none by that name.
 * @param ctx {Context} the request context
 * @param name {string} the path parameter name
 * @return {string} the captured value, or "" if absent
 */
export func param(ctx as Context, name as string) {
    if (maps.has($ctx.params, $name)) {
        return $ctx.params[$name];
    }
    return "";
}

/**
 * Set a response header.
 * @param ctx {Context} the request context
 * @param name {string} the header name
 * @param value {string} the header value
 */
export func setHeader(ctx as Context, name as string, value as string) {
    httpd.setHeader($ctx.req, $name, $value);
}

/**
 * Answer the request with a status code and body.
 * @param ctx {Context} the request context
 * @param status {int} the HTTP status code
 * @param body {string} the response body
 */
export func respond(ctx as Context, status as int, body as string) {
    httpd.respond($ctx.req, $status, $body);
}

/**
 * Answer with a text/plain body.
 * @param ctx {Context} the request context
 * @param status {int} the HTTP status code
 * @param body {string} the response body
 */
export func text(ctx as Context, status as int, body as string) {
    httpd.setHeader($ctx.req, "Content-Type", "text/plain; charset=utf-8");
    httpd.respond($ctx.req, $status, $body);
}

/**
 * Answer with an application/json body encoded from a json.Value. (Named
 * sendJson, not json, because a method may not shadow the `json` namespace this
 * module imports.)
 * @param ctx {Context} the request context
 * @param status {int} the HTTP status code
 * @param doc {json.Value} the JSON document to encode
 */
export func sendJson(ctx as Context, status as int, doc as json.Value) {
    httpd.setHeader($ctx.req, "Content-Type", "application/json");
    httpd.respond($ctx.req, $status, json.encode($doc));
}

/**
 * Answer with a file from disk.
 * @param ctx {Context} the request context
 * @param path {string} the filesystem path to serve
 */
export func serveFile(ctx as Context, path as string) {
    httpd.serveFile($ctx.req, $path);
}

# --- cookies ----------------------------------------------------------------

/**
 * Attributes for a Set-Cookie header. A zero-value struct is a session cookie
 * (no attributes); set the fields you need.
 * @field path {string} the Path attribute ("" omits it)
 * @field domain {string} the Domain attribute ("" omits it)
 * @field maxAge {int} the Max-Age in seconds; 0 omits it, a negative value expires the cookie now
 * @field httpOnly {bool} add HttpOnly (hide the cookie from JavaScript)
 * @field secure {bool} add Secure (send only over HTTPS)
 * @field sameSite {string} the SameSite attribute: "Lax", "Strict", "None", or "" to omit
 */
export def struct CookieOptions {
    path as string,
    domain as string,
    maxAge as int,
    httpOnly as bool,
    secure as bool,
    sameSite as string
};

# parseCookie finds `name` in a raw `Cookie` header ("k1=v1; k2=v2"), returning
# its value or "" when absent.
func parseCookie(header as string, name as string) {
    def parts as list of string init strings.split($header, ";");
    for (def p in $parts) {
        def kv as string init strings.trim($p);
        def eq as int init strings.indexOf($kv, "=");
        if ($eq > 0) {
            if (strings.substring($kv, 0, $eq) == $name) {
                return strings.substring($kv, $eq + 1, len($kv));
            }
        }
    }
    return "";
}

# formatSetCookie renders a Set-Cookie header value from name, value, and options.
func formatSetCookie(name as string, value as string, opts as CookieOptions) {
    def out as string init $name + "=" + $value;
    if (len($opts.path) > 0) {
        $out = $out + "; Path=" + $opts.path;
    }
    if (len($opts.domain) > 0) {
        $out = $out + "; Domain=" + $opts.domain;
    }
    if (not ($opts.maxAge == 0)) {
        $out = $out + "; Max-Age=" + convert.toString($opts.maxAge);
    }
    if ($opts.httpOnly) {
        $out = $out + "; HttpOnly";
    }
    if ($opts.secure) {
        $out = $out + "; Secure";
    }
    if (len($opts.sameSite) > 0) {
        $out = $out + "; SameSite=" + $opts.sameSite;
    }
    return $out;
}

/**
 * Return the value of the request cookie named `name`, or "" if absent.
 * @param ctx {Context} the request context
 * @param name {string} the cookie name
 * @return {string} the cookie value, or "" when absent
 */
export func cookie(ctx as Context, name as string) {
    return parseCookie(httpd.header($ctx.req, "Cookie"), $name);
}

/**
 * Set a response cookie with the given attributes (a `Set-Cookie` header).
 * @param ctx {Context} the request context
 * @param name {string} the cookie name
 * @param value {string} the cookie value
 * @param opts {CookieOptions} the cookie attributes (zero-value = a session cookie)
 */
export func setCookie(ctx as Context, name as string, value as string, opts as CookieOptions) {
    httpd.setHeader($ctx.req, "Set-Cookie", formatSetCookie($name, $value, $opts));
    return null;
}

# --- sessions ---------------------------------------------------------------

/**
 * Return the request's session id, minting a new one on first use. If the
 * `cookieName` cookie is present its value is returned; otherwise a fresh UUID
 * is generated and set as a `HttpOnly`, `SameSite=Lax`, path-`/` cookie. `web`
 * manages only the id cookie - the session data itself lives in a store the app
 * owns (e.g. the `session` module over `memcache`), so `web` forces no store or
 * network dependency. Call once per request and capture the returned id.
 * @param ctx {Context} the request context
 * @param cookieName {string} the session-id cookie name (e.g. "sid")
 * @return {string} the session id (existing or newly minted)
 */
export func sessionId(ctx as Context, cookieName as string) {
    def id as string init cookie($ctx, $cookieName);
    if (len($id) > 0) {
        return $id;
    }
    $id = uuid.generate("v4");
    def opts as CookieOptions;
    $opts.path = "/";
    $opts.httpOnly = true;
    $opts.sameSite = "Lax";
    setCookie($ctx, $cookieName, $id, $opts);
    return $id;
}

# --- routing internals ------------------------------------------------------

# splitPath breaks a URL path into non-empty segments ("/a/b/" -> ["a", "b"]).
func splitPath(p as string) {
    def raw as list of string init strings.split($p, "/");
    def out as list of string init [];
    for (def seg in $raw) {
        if (not ($seg == "")) {
            $out = lists.push($out, $seg);
        }
    }
    return $out;
}

# matchRoute finds the first route matching method + path, capturing any
# `:param` segments. Returns a Match with found=false when nothing matches.
func matchRoute(app as App, method as string, path as string) {
    def segs as list of string init splitPath($path);
    for (def r in $app.routes) {
        if ($r.method == $method) {
            def pat as list of string init splitPath($r.pattern);
            if (len($pat) == len($segs)) {
                def params as map of string to string init {};
                def ok as bool init true;
                def idx as int init 0;
                for (def ps in $pat) {
                    def actual as string init $segs[$idx];
                    if (strings.startsWith($ps, ":")) {
                        def key as string init strings.substring($ps, 1, len($ps));
                        $params[$key] = $actual;
                    } elseif (not ($ps == $actual)) {
                        $ok = false;
                    }
                    $idx = $idx + 1;
                }
                if ($ok) {
                    return Match{ found: true, handler: $r.handler, params: $params };
                }
            }
        }
    }
    def empty as map of string to string init {};
    return Match{ found: false, handler: "", params: $empty };
}

# runMiddleware runs the chain; returns false as soon as one middleware halts.
func runMiddleware(app as App, ctx as Context) {
    for (def mw in $app.middleware) {
        def keep as bool init meta.callMain($mw, $ctx);
        if (not $keep) {
            return false;
        }
    }
    return true;
}

# ensureAnswered guarantees a request is answered exactly once: httpd.respond
# errors on an already-answered request, which we swallow, so a handler that
# forgot to respond still gets a response instead of hanging the connection.
func ensureAnswered(req as httpd.Request, status as int, body as string) {
    try {
        httpd.respond($req, $status, $body);
    } catch (err) {
        # already answered - nothing to do.
    }
}

# dispatch runs the middleware chain then the handler, guaranteeing a response.
func dispatch(app as App, handler as string, ctx as Context, req as httpd.Request) {
    def cont as bool init runMiddleware($app, $ctx);
    if ($cont) {
        try {
            meta.callMain($handler, $ctx);
        } catch (err) {
            # handler failed - the safety net below turns it into a 500.
        }
    }
    ensureAnswered($req, 500, "500 internal server error\n");
}

# --- serving ----------------------------------------------------------------

/**
 * Serve on an already-listening httpd.Server, dispatching each request to its
 * matched handler. Blocks until the server is shut down (httpd.accept then
 * errors and the loop exits). Use this when you want to hold the server handle
 * yourself - e.g. to shut it down from another task, or to serve from a
 * `spawn`. `web.run` is the listen-and-serve convenience over it.
 * @param app {App} the router
 * @param srv {httpd.Server} the already-listening server handle
 */
export func serveOn(app as App, srv as httpd.Server) {
    def running as bool init true;
    while ($running) {
        try {
            def req as httpd.Request init httpd.accept($srv);
            def m as string init httpd.method($req);
            def p as string init httpd.path($req);
            def matched as Match init matchRoute($app, $m, $p);
            def ctx as Context init Context{ req: $req, params: $matched.params };
            if ($matched.found) {
                dispatch($app, $matched.handler, $ctx, $req);
            } elseif (not ($app.notFound == "")) {
                dispatch($app, $app.notFound, $ctx, $req);
            } else {
                httpd.respond($req, 404, "404 not found\n");
            }
        } catch (err) {
            # httpd.accept errors once the server is shut down; end the loop.
            $running = false;
        }
    }
}

/**
 * Listen on addr and serve forever, dispatching each request to its matched
 * handler. Blocks; interrupt to stop.
 * @param app {App} the router
 * @param addr {string} the listen address (e.g. ":8080")
 */
export func run(app as App, addr as string) {
    def srv as httpd.Server init httpd.listen($addr);
    serveOn($app, $srv);
}
