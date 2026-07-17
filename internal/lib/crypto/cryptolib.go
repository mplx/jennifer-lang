// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

// Package cryptolib implements Jennifer's `crypto` library: crypto-grade
// randomness, constant-time comparison, and the two standard-library key
// derivation functions (HKDF and PBKDF2-HMAC-SHA256). Digests live in `hash`
// (MD5 / SHA-*) and the keyed-hash MAC in `hash.hmac`; this library is the
// place for the security primitives that need a cryptographically secure
// source or a timing-safe operation.
//
// Everything here is standard library only (`crypto/rand`, `crypto/subtle`,
// `crypto/hkdf`, `crypto/pbkdf2`, added to the Go stdlib in 1.24), so the
// library adds no dependency and stays TinyGo-clean.
//
// The random source is a package-level buffer refilled from `crypto/rand` so
// the common per-byte draw (uuid's `randByte`) does not pay a syscall each
// time; `crypto/rand.Read` never returns an error on the supported platform,
// and an impossible failure panics rather than silently yielding weak bytes.
package cryptolib

import (
	"crypto/hkdf"
	"crypto/pbkdf2"
	crand "crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	gohash "hash"
	"sync"

	"jennifer-lang.dev/jennifer/internal/interpreter"
)

// LibraryName is the namespace prefix (`crypto.`) and the `use` name.
const LibraryName = "crypto"

// kdfHash maps a Jennifer-side algorithm string to the hash constructor the
// KDFs use as their PRF. The same names as `hash.compute` / `hash.hmac`, minus
// md5 (too weak to derive a key with). SHA-1 stays available because SCRAM-SHA-1
// (MongoDB, XMPP) mandates PBKDF2-HMAC-SHA1.
var kdfHash = map[string]func() gohash.Hash{
	"sha1":   sha1.New,
	"sha256": sha256.New,
	"sha512": sha512.New,
}

// kdfAlgoList is the rendered known-algorithm set for error messages.
const kdfAlgoList = `"sha1", "sha256", "sha512"`

// Install registers the crypto surface.
func Install(in *interpreter.Interpreter) {
	in.RegisterNamespaced(LibraryName, "randBytes", randBytesFn)
	in.RegisterNamespaced(LibraryName, "randInt", randIntFn)
	in.RegisterNamespaced(LibraryName, "hmacEqual", hmacEqualFn)
	in.RegisterNamespaced(LibraryName, "hkdf", hkdfFn)
	// Registered as `pbkdf` (not `pbkdf2`): a Jennifer method name is
	// letters-only, so the "2" can't appear - the same rule that makes uuid's
	// version a string arg (`uuid.generate("v4")`). The scheme is still
	// PBKDF2-HMAC-SHA256; the name just drops the digit.
	in.RegisterNamespaced(LibraryName, "pbkdf", pbkdf2Fn)
}

// ----- random source -------------------------------------------------------

var (
	randMu  sync.Mutex
	randBuf [512]byte
	randPos = len(randBuf) // past the end: force a refill on the first draw
)

// RandByte returns one cryptographically secure random byte. It is the source
// `uuidlib.randByte` draws on, so v4 / v7 UUIDs are unguessable. Buffered so a
// per-byte caller does not hit `crypto/rand` on every call.
func RandByte() byte {
	randMu.Lock()
	defer randMu.Unlock()
	if randPos >= len(randBuf) {
		fill(randBuf[:])
		randPos = 0
	}
	b := randBuf[randPos]
	randPos++
	return b
}

// RandFill fills b with cryptographically secure random bytes from the shared
// buffer, taking the lock once for the whole slice. uuid fills all 16 (or 10)
// of a UUID's random bytes in one call, so it pays one lock acquisition instead
// of one per byte. Refills the buffer mid-copy as needed.
func RandFill(b []byte) {
	randMu.Lock()
	defer randMu.Unlock()
	for i := range b {
		if randPos >= len(randBuf) {
			fill(randBuf[:])
			randPos = 0
		}
		b[i] = randBuf[randPos]
		randPos++
	}
}

// fill reads len(b) crypto-grade random bytes into b. crypto/rand.Read fills
// completely and does not return an error on the supported platform; a failure
// is unrecoverable (no weak fallback), so it panics.
func fill(b []byte) {
	if _, err := crand.Read(b); err != nil {
		panic("crypto: secure random source unavailable: " + err.Error())
	}
}

// randUint64 draws a uniform 64-bit value directly from crypto/rand (not the
// per-byte buffer): callers here need 8 bytes at a time, not one.
func randUint64() uint64 {
	var b [8]byte
	fill(b[:])
	return binary.BigEndian.Uint64(b[:])
}

// maxRandBytes bounds a single crypto.randBytes request. Keys, tokens, nonces,
// and salts are tens of bytes to a few KB; 64 MiB is far beyond any legitimate
// draw and keeps an untrusted length (e.g. a request field) from triggering an
// unbounded `make` - which, since the interpreter has no recover(), is a fatal
// process crash (uncatchable OOM), not a Jennifer error.
const maxRandBytes = 1 << 26 // 64 MiB

// randBytesFn implements `crypto.randBytes(n) -> bytes`: n crypto-grade random
// bytes. n must be in [0, maxRandBytes]; n == 0 yields empty bytes.
func randBytesFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if len(args) != 1 {
		return interpreter.Null(), fmt.Errorf("crypto.randBytes expects 1 argument (n), got %d", len(args))
	}
	if args[0].Kind != interpreter.KindInt {
		return interpreter.Null(), fmt.Errorf("crypto.randBytes: n must be int, got %s", args[0].Kind)
	}
	n := args[0].Int
	if n < 0 {
		return interpreter.Null(), fmt.Errorf("crypto.randBytes: n must be >= 0, got %d", n)
	}
	if n > maxRandBytes {
		return interpreter.Null(), fmt.Errorf("crypto.randBytes: n (%d) exceeds the %d-byte limit", n, maxRandBytes)
	}
	out := make([]byte, n)
	fill(out)
	return interpreter.BytesVal(out), nil
}

// randIntFn implements `crypto.randInt(lo, hi) -> int`: a uniform int in the
// inclusive range [lo, hi], drawn crypto-grade. Same shape as math.randInt,
// but unpredictable and unseedable - the drop-in for a security context.
// Rejection sampling makes the distribution exactly uniform (no modulo bias).
func randIntFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if len(args) != 2 {
		return interpreter.Null(), fmt.Errorf("crypto.randInt expects 2 arguments (lo, hi), got %d", len(args))
	}
	if args[0].Kind != interpreter.KindInt || args[1].Kind != interpreter.KindInt {
		return interpreter.Null(), fmt.Errorf("crypto.randInt: lo and hi must be int")
	}
	lo, hi := args[0].Int, args[1].Int
	if lo > hi {
		return interpreter.Null(), fmt.Errorf("crypto.randInt: lo (%d) must be <= hi (%d)", lo, hi)
	}
	return interpreter.IntVal(randInRange(lo, hi)), nil
}

// randInRange returns a uniform int64 in [lo, hi]. Width is computed in uint64
// so the full int64 span (MinInt64..MaxInt64) does not overflow; the top
// partial block is rejected so every value is equally likely.
func randInRange(lo, hi int64) int64 {
	width := uint64(hi) - uint64(lo) // exact for hi >= lo, even across the sign boundary
	if width == ^uint64(0) {
		// The interval is the entire int64 range: every 64-bit draw is valid.
		return int64(uint64(lo) + randUint64())
	}
	n := width + 1
	mod := ((^uint64(0) % n) + 1) % n // 2^64 mod n, computed without overflowing
	for {
		r := randUint64()
		if mod != 0 && r >= -mod { // r in the top [2^64-mod, 2^64) block: biased, reject
			continue
		}
		return int64(uint64(lo) + r%n)
	}
}

// ----- constant-time comparison --------------------------------------------

// hmacEqualFn implements `crypto.hmacEqual(a, b) -> bool`: a constant-time
// equality test over two `bytes`, for comparing a computed MAC against a
// supplied one without leaking, through response timing, how many leading
// bytes matched. Unequal lengths return false (also in constant time relative
// to the inputs' contents).
func hmacEqualFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if len(args) != 2 {
		return interpreter.Null(), fmt.Errorf("crypto.hmacEqual expects 2 arguments (a, b), got %d", len(args))
	}
	if args[0].Kind != interpreter.KindBytes || args[1].Kind != interpreter.KindBytes {
		return interpreter.Null(), fmt.Errorf("crypto.hmacEqual: both arguments must be bytes")
	}
	eq := subtle.ConstantTimeCompare(args[0].Bytes, args[1].Bytes) == 1
	return interpreter.BoolVal(eq), nil
}

// ----- key derivation ------------------------------------------------------

// hkdfFn implements `crypto.hkdf(secret, salt, info, length, algo) -> bytes`:
// the HMAC-based Extract-and-Expand KDF (RFC 5869), deriving `length` bytes of
// keying material from a high-entropy `secret`. `salt` and `info` may be empty
// bytes; `algo` picks the PRF hash ("sha1" / "sha256" / "sha512").
func hkdfFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if len(args) != 5 {
		return interpreter.Null(), fmt.Errorf("crypto.hkdf expects 5 arguments (secret, salt, info, length, algo), got %d", len(args))
	}
	if args[0].Kind != interpreter.KindBytes || args[1].Kind != interpreter.KindBytes || args[2].Kind != interpreter.KindBytes {
		return interpreter.Null(), fmt.Errorf("crypto.hkdf: secret, salt, and info must be bytes")
	}
	if args[3].Kind != interpreter.KindInt {
		return interpreter.Null(), fmt.Errorf("crypto.hkdf: length must be int, got %s", args[3].Kind)
	}
	h, err := kdfHashOf("crypto.hkdf", args[4])
	if err != nil {
		return interpreter.Null(), err
	}
	length := args[3].Int
	if length <= 0 {
		return interpreter.Null(), fmt.Errorf("crypto.hkdf: length must be > 0, got %d", length)
	}
	key, kerr := hkdf.Key(h, args[0].Bytes, args[1].Bytes, string(args[2].Bytes), int(length))
	if kerr != nil {
		return interpreter.Null(), fmt.Errorf("crypto.hkdf: %v", kerr)
	}
	return interpreter.BytesVal(key), nil
}

// maxKeyLen bounds a derived-key length. Real keys are tens of bytes; 1 MiB is
// generous and keeps a large keyLen from allocating gigabytes and pinning a
// core (PBKDF2 output work is proportional to keyLen). maxPBKDFIterations caps
// the stretch factor: it is far above any legitimate value (OWASP suggests
// hundreds of thousands) but stops an untrusted iteration count - e.g. a
// hostile SCRAM server's `i=` - from running effectively forever with no
// interrupt.
const (
	maxKeyLen          = 1 << 20     // 1 MiB
	maxPBKDFIterations = 100_000_000 // 1e8
)

// pbkdf2Fn implements `crypto.pbkdf(password, salt, iterations, keyLen, algo)
// -> bytes`: PBKDF2 (RFC 8018) deriving a `keyLen`-byte key from a low-entropy
// `password` stretched by `iterations` rounds against `salt`, with `algo` the
// HMAC hash ("sha1" / "sha256" / "sha512"). A higher iteration count is slower
// to brute-force; pick the largest the deployment can afford. (SCRAM-SHA-1
// needs "sha1"; new password stores want "sha256" or better.)
func pbkdf2Fn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if len(args) != 5 {
		return interpreter.Null(), fmt.Errorf("crypto.pbkdf expects 5 arguments (password, salt, iterations, keyLen, algo), got %d", len(args))
	}
	if args[0].Kind != interpreter.KindBytes || args[1].Kind != interpreter.KindBytes {
		return interpreter.Null(), fmt.Errorf("crypto.pbkdf: password and salt must be bytes")
	}
	if args[2].Kind != interpreter.KindInt || args[3].Kind != interpreter.KindInt {
		return interpreter.Null(), fmt.Errorf("crypto.pbkdf: iterations and keyLen must be int")
	}
	h, err := kdfHashOf("crypto.pbkdf", args[4])
	if err != nil {
		return interpreter.Null(), err
	}
	iterations, keyLen := args[2].Int, args[3].Int
	if iterations <= 0 {
		return interpreter.Null(), fmt.Errorf("crypto.pbkdf: iterations must be > 0, got %d", iterations)
	}
	if iterations > maxPBKDFIterations {
		return interpreter.Null(), fmt.Errorf("crypto.pbkdf: iterations (%d) exceeds the %d limit", iterations, maxPBKDFIterations)
	}
	if keyLen <= 0 {
		return interpreter.Null(), fmt.Errorf("crypto.pbkdf: keyLen must be > 0, got %d", keyLen)
	}
	if keyLen > maxKeyLen {
		return interpreter.Null(), fmt.Errorf("crypto.pbkdf: keyLen (%d) exceeds the %d-byte limit", keyLen, maxKeyLen)
	}
	key, kerr := pbkdf2.Key(h, string(args[0].Bytes), args[1].Bytes, int(iterations), int(keyLen))
	if kerr != nil {
		return interpreter.Null(), fmt.Errorf("crypto.pbkdf: %v", kerr)
	}
	return interpreter.BytesVal(key), nil
}

// kdfHashOf resolves the algo argument shared by hkdf and pbkdf.
func kdfHashOf(fn string, v interpreter.Value) (func() gohash.Hash, error) {
	if v.Kind != interpreter.KindString {
		return nil, fmt.Errorf("%s: algo must be string, got %s", fn, v.Kind)
	}
	h, ok := kdfHash[v.Str]
	if !ok {
		return nil, fmt.Errorf("%s: unknown algorithm %q; known: %s", fn, v.Str, kdfAlgoList)
	}
	return h, nil
}
