# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# web.j - an ergonomic HTTP framework over the `httpd` server engine. Register
# routes against handler methods by name, then `web.run` owns the accept loop,
# matches each request, and dispatches to the handler. Since Jennifer has no
# first-class functions, dispatch is by method name via `meta.callMain` (which
# reaches the entry program's handlers across the module boundary); a handler
# is `func name(ctx as web.Context) { ... }`. Routing supports `:param`
# segments, a middleware chain, and a custom not-found handler. Response and
# request helpers hang off the `web.Context` so a handler rarely reaches for
# `httpd` directly.
#
# Needs the default `jennifer` binary (the `httpd` engine is net-backed; the
# constrained `jennifer-tiny` has no network stack).

use httpd;
use meta;
use json;
use lists;
use maps;
use strings;

# A Route is one registered (method, pattern, handler-name) triple. Exported
# only to satisfy the referential-closure rule (App exposes a list of Route);
# programs never build one directly - they call web.get / web.route.
export def struct Route {
    method as string,
    pattern as string,
    handler as string
};

# App is the value-semantic router state: the routes, the middleware chain
# (handler names run before each route), and an optional not-found handler
# name (empty = the built-in 404). Every registrar returns a fresh App.
export def struct App {
    routes as list of Route,
    middleware as list of string,
    notFound as string
};

# Context is what a handler receives: the underlying request handle plus the
# path parameters captured from the matched route (`:id` -> params["id"]).
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

# new returns an empty App.
export func new() {
    def routes as list of Route init [];
    def mw as list of string init [];
    return App{ routes: $routes, middleware: $mw, notFound: "" };
}

# route registers a handler for a method + pattern, returning a new App. It
# errors early if the handler method is not defined.
export func route(app as App, method as string, pattern as string, handler as string) {
    if (not meta.definedMain($handler)) {
        throw Error{ kind: "web", message: "web.route: handler not defined: " + $handler, file: "", line: 0, col: 0 };
    }
    def r as Route init Route{ method: $method, pattern: $pattern, handler: $handler };
    def out as App init $app;
    $out.routes = lists.push($out.routes, $r);
    return $out;
}

export func get(app as App, pattern as string, handler as string) {
    return route($app, "GET", $pattern, $handler);
}

export func post(app as App, pattern as string, handler as string) {
    return route($app, "POST", $pattern, $handler);
}

export func put(app as App, pattern as string, handler as string) {
    return route($app, "PUT", $pattern, $handler);
}

export func patch(app as App, pattern as string, handler as string) {
    return route($app, "PATCH", $pattern, $handler);
}

export func delete(app as App, pattern as string, handler as string) {
    return route($app, "DELETE", $pattern, $handler);
}

# before registers a middleware handler, run before each route handler. A
# middleware is `func name(ctx as web.Context) { ...; return true; }`: return
# true to continue to the route handler, or respond and return false to halt.
export func before(app as App, handler as string) {
    if (not meta.definedMain($handler)) {
        throw Error{ kind: "web", message: "web.before: middleware not defined: " + $handler, file: "", line: 0, col: 0 };
    }
    def out as App init $app;
    $out.middleware = lists.push($out.middleware, $handler);
    return $out;
}

# notFound sets a custom handler for unmatched requests (default: a plain 404).
export func notFound(app as App, handler as string) {
    if (not meta.definedMain($handler)) {
        throw Error{ kind: "web", message: "web.notFound: handler not defined: " + $handler, file: "", line: 0, col: 0 };
    }
    def out as App init $app;
    $out.notFound = $handler;
    return $out;
}

# --- request / response helpers (on a Context) ------------------------------

export func method(ctx as Context) { return httpd.method($ctx.req); }

export func path(ctx as Context) { return httpd.path($ctx.req); }

export func query(ctx as Context, name as string) { return httpd.query($ctx.req, $name); }

export func header(ctx as Context, name as string) { return httpd.header($ctx.req, $name); }

export func body(ctx as Context) { return httpd.body($ctx.req); }

# param returns a captured path parameter, or "" if the route had none by that
# name.
export func param(ctx as Context, name as string) {
    if (maps.has($ctx.params, $name)) {
        return $ctx.params[$name];
    }
    return "";
}

export func setHeader(ctx as Context, name as string, value as string) {
    httpd.setHeader($ctx.req, $name, $value);
}

export func respond(ctx as Context, status as int, body as string) {
    httpd.respond($ctx.req, $status, $body);
}

# text answers with a text/plain body.
export func text(ctx as Context, status as int, body as string) {
    httpd.setHeader($ctx.req, "Content-Type", "text/plain; charset=utf-8");
    httpd.respond($ctx.req, $status, $body);
}

# sendJson answers with an application/json body encoded from a json.Value.
# (Named sendJson, not json, because a method may not shadow the `json`
# namespace this module imports.)
export func sendJson(ctx as Context, status as int, doc as json.Value) {
    httpd.setHeader($ctx.req, "Content-Type", "application/json");
    httpd.respond($ctx.req, $status, json.encode($doc));
}

# serveFile answers with a file from disk.
export func serveFile(ctx as Context, path as string) {
    httpd.serveFile($ctx.req, $path);
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

# serveOn serves on an already-listening httpd.Server, dispatching each request
# to its matched handler. Blocks until the server is shut down (httpd.accept
# then errors and the loop exits). Use this when you want to hold the server
# handle yourself - e.g. to shut it down from another task, or to serve from a
# `spawn`. `web.run` is the listen-and-serve convenience over it.
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

# run listens on addr and serves forever, dispatching each request to its
# matched handler. Blocks; interrupt to stop.
export func run(app as App, addr as string) {
    def srv as httpd.Server init httpd.listen($addr);
    serveOn($app, $srv);
}
