#!/usr/bin/env -S jennifer run
# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# gotify_demo.j - push a notification to a Gotify server with the gotify module.
#
# Needs a Gotify server and the default `jennifer` binary (the module uses
# `net` via `http`). Supply the target through the environment so nothing is
# committed:
#
#     GOTIFY_URL=https://push.example.com GOTIFY_TOKEN=AqB3cD... \
#         jennifer run examples/modules/gotify_demo.j
#
# With the variables unset (or no server reachable) it prints a hint / the error
# rather than failing. Not a golden test (it needs a live server).
use io;
use os;
import "../../modules/gotify.j" as gotify;
import "../../modules/http.j" as http;

def url as string init os.getEnv("GOTIFY_URL");
def token as string init os.getEnv("GOTIFY_TOKEN");

if (len($url) == 0 or len($token) == 0) {
    io.printf("set GOTIFY_URL and GOTIFY_TOKEN to push a real notification\n");
} else {
    def g as gotify.Config init gotify.Config{url: $url, token: $token};
    try {
        def r as http.Response init gotify.push($g, "Jennifer",
            "Hello from the gotify demo", 5);
        io.printf("push -> %d %s\n", $r.status, $r.statusText);
    } catch (e) {
        io.printf("no Gotify server at %s (%s)\n", $url, $e.message);
    }
}
