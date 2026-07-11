# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# session.j - server-side sessions on memcached, the canonical memcached use.
# A session is a `map of string to string` held under a `sess:ID` key with a
# sliding TTL, so it expires on its own when idle. The pieces: `memcache`
# (store + TTL), `uuid` (the session ID), and `json` (encode the map). Built on
# the `memcache` module, so it needs the default `jennifer` binary.
#
#     import "session.j" as session;
#     import "memcache.j" as memcache;
#
#     def mc as memcache.Session init memcache.connect(memcache.Options{
#         host: "127.0.0.1", port: 11211});
#     def id as string init session.create($mc, 1800);      # 30-minute session
#     def data as map of string to string init session.load($mc, $id);
#     $data["user"] = "ada";
#     session.save($mc, $id, $data, 1800);
#
# Sessions are volatile (memcached evicts under memory pressure), so treat them
# as a cache of soft state, not a store of record. The JSON is stored base64-
# wrapped, so session values are binary-safe (any UTF-8), at the cost of not
# being human-readable in the cache; a PHP-session-compatible layout is a
# separate follow-on. Session IDs are UUID v4 from `math`'s non-crypto RNG (see
# the `uuid` module) - fine for a cache key, not a security token on its own.
use strings;
use convert;
use json;
use uuid;
use encoding;
import "./memcache.j" as memcache;

# The key namespace for sessions in the cache.
def const PREFIX as string init "sess:";

# --- helpers (private) ---------------------------------------------

func cacheKey(id as string) {
    return PREFIX + $id;
}

# pointer builds the RFC 6901 JSON Pointer for a session-data key, escaping
# `~` and `/` so keys carrying them still resolve.
func pointer(k as string) {
    def e as string init strings.replace($k, "~", "~0");
    return "/" + strings.replace($e, "/", "~1");
}

# encodeData serializes a session map to a storable ASCII blob (base64 of the
# JSON), so the cache only ever holds ASCII and any UTF-8 value round-trips.
func encodeData(data as map of string to string) {
    def js as string init json.encode($data);
    return encoding.toText(convert.bytesFromString($js, "utf-8"), "base64");
}

# decodeData parses a stored blob back into a session map (empty when the blob
# is empty, i.e. the session was absent or expired).
func decodeData(blob as string) {
    def out as map of string to string init {};
    if (len($blob) == 0) {
        return $out;
    }
    def js as string init convert.stringFromBytes(encoding.fromText($blob, "base64"), "utf-8");
    def node as json.Value init json.decode($js);
    for (def k in json.keys($node)) {
        $out[$k] = json.asString($node, pointer($k));
    }
    return $out;
}

# --- session lifecycle (exported) ----------------------------------

# create mints a new session ID and stores an empty session with a `ttl`-second
# expiry; returns the ID.
export func create(mc as memcache.Session, ttl as int) {
    def id as string init uuid.generate("v4");
    def empty as map of string to string init {};
    memcache.set($mc, cacheKey($id), encodeData($empty), $ttl);
    return $id;
}

# load returns the session's data map, or an empty map when the session is
# absent or expired.
export func load(mc as memcache.Session, id as string) {
    return decodeData(memcache.get($mc, cacheKey($id)));
}

# save writes the session's data map and re-arms its expiry to `ttl` seconds.
export func save(mc as memcache.Session, id as string,
    data as map of string to string, ttl as int) {
    memcache.set($mc, cacheKey($id), encodeData($data), $ttl);
}

# touch re-arms the session's expiry without rewriting its data; returns whether
# the session still existed.
export func touch(mc as memcache.Session, id as string, ttl as int) {
    return memcache.touch($mc, cacheKey($id), $ttl);
}

# destroy removes the session; returns whether it existed.
export func destroy(mc as memcache.Session, id as string) {
    return memcache.delete($mc, cacheKey($id));
}
