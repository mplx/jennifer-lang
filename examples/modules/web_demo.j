#!/usr/bin/env -S jennifer run
# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>

/**
 * A tiny web app with the web framework module.
 * Registers routes against handler methods by name, serves them on an ephemeral local port in a spawned task, then acts as its own client to exercise the routes and prints the results. Self-contained, so it needs no external service - just the default jennifer binary (the httpd engine is net-backed).
 * @module web_demo
 */
use io;
use httpd;
use task;
use json;
import "../../modules/web.j" as web;
import "../../modules/http.j" as http;

# --- handlers: each takes a web.Context ------------------------------------

/**
 * Handle GET / - the welcome page.
 * @param ctx {web.Context} the request context
 */
func showHome(ctx as web.Context) {
    web.text($ctx, 200, "welcome to the jennifer web demo\n");
}

/**
 * Handle GET /users/:id - echo the captured id as JSON.
 * @param ctx {web.Context} the request context
 */
func showUser(ctx as web.Context) {
    def id as string init web.param($ctx, "id");
    def out as json.Value init json.map();
    $out = json.set($out, "/id", $id);
    $out = json.set($out, "/method", web.method($ctx));
    web.sendJson($ctx, 200, $out);
}

/**
 * Middleware: tag every response with an X-Powered-By header.
 * @param ctx {web.Context} the request context
 * @return {bool} true to continue to the route handler
 */
func addServerHeader(ctx as web.Context) {
    web.setHeader($ctx, "X-Powered-By", "jennifer/web");
    return true;
}

# --- build the app ----------------------------------------------------------

def app as web.App init web.new();
$app = web.before($app, "addServerHeader");
$app = web.get($app, "/", "showHome");
$app = web.get($app, "/users/:id", "showUser");

# --- serve on an ephemeral port, in the background --------------------------

def srv as httpd.Server init httpd.listen("127.0.0.1:0");
def addr as string init httpd.address($srv);
def server as task of null init spawn { web.serveOn($app, $srv); };

# --- act as our own client --------------------------------------------------

def noHeaders as map of string to string init {};

def home as http.Response init http.get("http://" + $addr + "/", $noHeaders);
io.printf("GET /          -> %d %s", $home.status, $home.body);

def user as http.Response init http.get("http://" + $addr + "/users/7", $noHeaders);
io.printf("GET /users/7   -> %d %s\n", $user.status, $user.body);

def missing as http.Response init http.get("http://" + $addr + "/missing", $noHeaders);
io.printf("GET /missing   -> %d %s", $missing.status, $missing.body);

# --- shut down cleanly ------------------------------------------------------

httpd.shutdown($srv);
task.wait($server);
io.printf("server stopped cleanly\n");
