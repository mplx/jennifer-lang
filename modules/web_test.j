# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# web_test.j - white-box tests for web.j. Run with:
#
#     jennifer test modules/web_test.j
#
# The overlay splices web.j in first, so these tests reach its private helpers
# (splitPath, matchRoute) and the private Match struct by bare identifier, as
# well as the exported registration surface. The serving loop itself is covered
# by the Go integration test (cmd/jennifer/web_test.go) and the demo, since it
# needs a live listener. web.j already `use`s httpd / meta / json / lists /
# maps / strings / convert / uuid, so the overlay only adds testing.
use testing;

# Dummy handlers so route registration's meta.defined check passes.
func hHome(ctx as Context) { return; }
func hUser(ctx as Context) { return; }

func testSplitPath() {
    testing.assertEqual(len(splitPath("/a/b/")), 2);
    testing.assertEqual(splitPath("/a/b")[0], "a");
    testing.assertEqual(splitPath("/a/b")[1], "b");
    testing.assertEqual(len(splitPath("/")), 0);
    testing.assertEqual(len(splitPath("")), 0);
}

func testMatchExact() {
    def app as App init new();
    $app = get($app, "/home", "hHome");
    def m as Match init matchRoute($app, "GET", "/home");
    testing.assertTrue($m.found);
    testing.assertEqual($m.handler, "hHome");
}

func testMatchParam() {
    def app as App init new();
    $app = get($app, "/users/:id", "hUser");
    def m as Match init matchRoute($app, "GET", "/users/42");
    testing.assertTrue($m.found);
    testing.assertEqual($m.params["id"], "42");
}

func testMultiParam() {
    def app as App init new();
    $app = get($app, "/users/:uid/posts/:pid", "hUser");
    def m as Match init matchRoute($app, "GET", "/users/7/posts/99");
    testing.assertEqual($m.params["uid"], "7");
    testing.assertEqual($m.params["pid"], "99");
}

func testNoMatchPath() {
    def app as App init new();
    $app = get($app, "/home", "hHome");
    def m as Match init matchRoute($app, "GET", "/nope");
    testing.assertFalse($m.found);
}

func testMethodMismatch() {
    def app as App init new();
    $app = get($app, "/home", "hHome");
    def m as Match init matchRoute($app, "POST", "/home");
    testing.assertFalse($m.found);
}

func testRegistrationIsImmutable() {
    def app as App init new();
    def grown as App init get($app, "/home", "hHome");
    testing.assertEqual(len($app.routes), 0);
    testing.assertEqual(len($grown.routes), 1);
}

func testVerbsSetMethod() {
    def app as App init new();
    $app = post($app, "/a", "hHome");
    $app = delete($app, "/b", "hHome");
    testing.assertEqual($app.routes[0].method, "POST");
    testing.assertEqual($app.routes[1].method, "DELETE");
}

func testBeforeRegisters() {
    def app as App init new();
    $app = before($app, "hHome");
    testing.assertEqual(len($app.middleware), 1);
}

func testNotFoundSets() {
    def app as App init new();
    $app = notFound($app, "hHome");
    testing.assertEqual($app.notFound, "hHome");
}

# badRoute registers an undefined handler, which web.route must reject.
func badRoute() {
    def app as App init new();
    def bad as App init route($app, "GET", "/x", "definitelyNotDefined");
}

func testRouteValidatesHandler() {
    testing.assertThrows("badRoute", "web");
}

func testParseCookie() {
    testing.assertEqual(parseCookie("sid=abc; theme=dark", "theme"), "dark");
    testing.assertEqual(parseCookie("sid=abc; theme=dark", "sid"), "abc");
    testing.assertEqual(parseCookie("sid=abc", "sid"), "abc");
    testing.assertEqual(parseCookie("", "sid"), "");
    testing.assertEqual(parseCookie("a=1; b=2", "c"), "");
}

func testFormatSetCookieMinimal() {
    def o as CookieOptions;
    testing.assertEqual(formatSetCookie("sid", "abc", $o), "sid=abc");
}

func testFormatSetCookieFull() {
    def o as CookieOptions;
    $o.path = "/";
    $o.maxAge = 3600;
    $o.httpOnly = true;
    $o.sameSite = "Lax";
    testing.assertEqual(formatSetCookie("sid", "abc", $o),
        "sid=abc; Path=/; Max-Age=3600; HttpOnly; SameSite=Lax");
}

func testFormatSetCookieExpire() {
    def o as CookieOptions;
    $o.maxAge = -1;
    $o.secure = true;
    testing.assertEqual(formatSetCookie("sid", "", $o), "sid=; Max-Age=-1; Secure");
}
