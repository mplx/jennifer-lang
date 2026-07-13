// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// A .j program builds a small app with the web framework, serves it on an
// ephemeral port in a spawned task, and acts as its own client. It asserts the
// full round-trip: an exact-route text response, a `:param` route returning
// JSON with the captured parameter, a middleware-set response header, and a 404
// for an unmatched path. This exercises the whole chain - httpd engine, route
// matching, and cross-module handler dispatch via meta.callMain (a web.Context
// built inside the module reaching a handler defined in this entry program).
func TestWebFrameworkRoundTrip(t *testing.T) {
	webMod, err := filepath.Abs(filepath.Join("..", "..", "modules", "web.j"))
	if err != nil {
		t.Fatal(err)
	}
	httpMod, err := filepath.Abs(filepath.Join("..", "..", "modules", "http.j"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	prog := fmt.Sprintf(`use testing;
use httpd;
use task;
use json;
import %q as web;
import %q as http;

func showHome(ctx as web.Context) { web.text($ctx, 200, "home"); }
func showUser(ctx as web.Context) {
    def out as json.Value init json.map();
    $out = json.set($out, "/id", web.param($ctx, "id"));
    $out = json.set($out, "/method", web.method($ctx));
    web.sendJson($ctx, 200, $out);
}
func tagHeader(ctx as web.Context) { web.setHeader($ctx, "X-Powered-By", "jennifer/web"); return true; }

def app as web.App init web.new();
$app = web.before($app, "tagHeader");
$app = web.get($app, "/", "showHome");
$app = web.get($app, "/users/:id", "showUser");

def srv as httpd.Server init httpd.listen("127.0.0.1:0");
def addr as string init httpd.address($srv);
def server as task of null init spawn { web.serveOn($app, $srv); };

def h as map of string to string init {};
def home as http.Response init http.get("http://" + $addr + "/", $h);
testing.assertEqual($home.status, 200);
testing.assertEqual($home.body, "home");
testing.assertEqual(http.header($home, "X-Powered-By"), "jennifer/web");

def user as http.Response init http.get("http://" + $addr + "/users/42", $h);
testing.assertEqual($user.status, 200);
def doc as json.Value init json.decode($user.body);
testing.assertEqual(json.asString($doc, "/id"), "42");
testing.assertEqual(json.asString($doc, "/method"), "GET");

def miss as http.Response init http.get("http://" + $addr + "/nope", $h);
testing.assertEqual($miss.status, 404);

httpd.shutdown($srv);
task.wait($server);`, webMod, httpMod)

	progPath := filepath.Join(dir, "app.j")
	if err := os.WriteFile(progPath, []byte(prog), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, code := loadForTest(progPath); code != testExitPass {
		t.Fatalf("web framework program failed with code %d", code)
	}
}

// TestWebCookiesAndSession drives the cookie + session-id surface end to end: a
// handler resolves the session id via web.sessionId (minting + Set-Cookie on
// first use), counting hits in an app-owned store keyed by the id. The .j
// program acts as its own client: a first cookieless GET mints a session
// (counter 1, HttpOnly Set-Cookie), a second GET replaying that cookie hits the
// same session (counter 2), and a third cookieless GET starts a fresh one
// (counter 1). Proves web owns only the id cookie while the store stays the
// app's.
func TestWebCookiesAndSession(t *testing.T) {
	webMod, err := filepath.Abs(filepath.Join("..", "..", "modules", "web.j"))
	if err != nil {
		t.Fatal(err)
	}
	httpMod, err := filepath.Abs(filepath.Join("..", "..", "modules", "http.j"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	prog := fmt.Sprintf(`use testing;
use httpd;
use task;
use convert;
use maps;
use strings;
import %q as web;
import %q as http;

# The app owns its session store (here a simple in-program map keyed by id);
# web only manages the id cookie.
def store as map of string to int init {};

func hit(ctx as web.Context) {
    def id as string init web.sessionId($ctx, "sid");
    def n as int init 0;
    if (maps.has($store, $id)) { $n = $store[$id]; }
    $n = $n + 1;
    $store[$id] = $n;
    web.text($ctx, 200, convert.toString($n));
}

def app as web.App init web.new();
$app = web.get($app, "/hit", "hit");
def srv as httpd.Server init httpd.listen("127.0.0.1:0");
def addr as string init httpd.address($srv);
def server as task of null init spawn { web.serveOn($app, $srv); };

def h as map of string to string init {};
def first as http.Response init http.get("http://" + $addr + "/hit", $h);
testing.assertEqual($first.status, 200);
testing.assertEqual($first.body, "1");
def sc as string init http.header($first, "Set-Cookie");
testing.assertTrue(strings.startsWith($sc, "sid="));
testing.assertTrue(strings.contains($sc, "HttpOnly"));

# Replay the cookie: same session, counter advances.
def pair as string init strings.substring($sc, 0, strings.indexOf($sc, ";"));
def hc as map of string to string init {};
$hc["Cookie"] = $pair;
def second as http.Response init http.get("http://" + $addr + "/hit", $hc);
testing.assertEqual($second.body, "2");

# No cookie: a fresh session starts over.
def third as http.Response init http.get("http://" + $addr + "/hit", $h);
testing.assertEqual($third.body, "1");

httpd.shutdown($srv);
task.wait($server);`, webMod, httpMod)

	progPath := filepath.Join(dir, "sess.j")
	if err := os.WriteFile(progPath, []byte(prog), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, code := loadForTest(progPath); code != testExitPass {
		t.Fatalf("web cookie/session program failed with code %d", code)
	}
}
