// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package cryptolib

import (
	"encoding/hex"
	"testing"

	"jennifer-lang.dev/jennifer/internal/interpreter"
)

var noCtx = interpreter.BuiltinCtx{}

func bytesArg(b []byte) interpreter.Value { return interpreter.BytesVal(b) }

func TestRandBytesLengthAndBounds(t *testing.T) {
	for _, n := range []int64{0, 1, 16, 512, 513} {
		v, err := randBytesFn(noCtx, []interpreter.Value{interpreter.IntVal(n)})
		if err != nil {
			t.Fatalf("randBytes(%d): %v", n, err)
		}
		if v.Kind != interpreter.KindBytes || int64(len(v.Bytes)) != n {
			t.Errorf("randBytes(%d): got %d bytes", n, len(v.Bytes))
		}
	}
	// Two draws of a decent width almost never collide.
	a, _ := randBytesFn(noCtx, []interpreter.Value{interpreter.IntVal(32)})
	b, _ := randBytesFn(noCtx, []interpreter.Value{interpreter.IntVal(32)})
	if string(a.Bytes) == string(b.Bytes) {
		t.Error("two 32-byte draws were identical")
	}
	if _, err := randBytesFn(noCtx, []interpreter.Value{interpreter.IntVal(-1)}); err == nil {
		t.Error("negative n should error")
	}
	if _, err := randBytesFn(noCtx, []interpreter.Value{interpreter.StringVal("x")}); err == nil {
		t.Error("non-int n should error")
	}
	// An over-limit n must be a catchable error, never an unbounded make (the
	// interpreter has no recover(), so make([]byte, huge) is a fatal crash).
	if _, err := randBytesFn(noCtx, []interpreter.Value{interpreter.IntVal(maxRandBytes + 1)}); err == nil {
		t.Error("over-limit n should error, not allocate")
	}
	// The cap itself is still allowed.
	if _, err := randBytesFn(noCtx, []interpreter.Value{interpreter.IntVal(maxRandBytes)}); err != nil {
		t.Errorf("n == maxRandBytes should be allowed: %v", err)
	}
}

// PBKDF2's keyLen and iterations must be bounded so an untrusted value (e.g. a
// hostile SCRAM server's i=) cannot allocate gigabytes or spin a core forever;
// both should surface as a catchable error, and sane values still derive a key.
func TestPBKDFBounds(t *testing.T) {
	pw := interpreter.BytesVal([]byte("password"))
	salt := interpreter.BytesVal([]byte("saltsalt"))
	algo := interpreter.StringVal("sha256")
	call := func(iter, keyLen int64) error {
		_, err := pbkdf2Fn(noCtx, []interpreter.Value{pw, salt, interpreter.IntVal(iter), interpreter.IntVal(keyLen), algo})
		return err
	}
	if call(1000, maxKeyLen+1) == nil {
		t.Error("over-limit keyLen should error")
	}
	// The work bound is on blocks*iterations, not either factor alone: a large
	// keyLen paired with a huge iteration count (individually plausible) is the
	// days-of-CPU case and must be rejected before it runs.
	if call(maxPBKDFWork, maxKeyLen) == nil {
		t.Error("keyLen*iterations over the work limit should error")
	}
	// sha256 = 32-byte blocks: 32768 blocks at 1MiB, so >maxPBKDFWork/32768 iters must fail.
	if call(maxPBKDFWork/32768+1, maxKeyLen) == nil {
		t.Error("work just over the limit should error")
	}
	// A modest single-block derivation (well under the work budget) is allowed
	// and cheap; the full-budget allowed case is ~20s of real work, so it is not
	// exercised here.
	if err := call(100000, 32); err != nil {
		t.Errorf("sane pbkdf params should derive a key: %v", err)
	}
}

// crypto.hkdf must reject an over-large length before the int() cast so a
// 32-bit target cannot truncate a huge length to a small accepted value and
// return a shorter-than-requested key.
func TestHKDFLengthBound(t *testing.T) {
	secret := interpreter.BytesVal([]byte("secret"))
	empty := interpreter.BytesVal(nil)
	algo := interpreter.StringVal("sha256")
	if _, err := hkdfFn(noCtx, []interpreter.Value{secret, empty, empty, interpreter.IntVal(maxKeyLen + 1), algo}); err == nil {
		t.Error("over-limit hkdf length should error")
	}
	if _, err := hkdfFn(noCtx, []interpreter.Value{secret, empty, empty, interpreter.IntVal(32), algo}); err != nil {
		t.Errorf("sane hkdf length should derive a key: %v", err)
	}
}

// RandFill fills a whole slice in one lock acquisition and yields distinct
// output across calls; the buffer refills correctly across its 512-byte edge.
func TestRandFill(t *testing.T) {
	var a, b [40]byte
	RandFill(a[:])
	RandFill(b[:])
	if a == b {
		t.Error("two RandFill draws were identical")
	}
	// A draw larger than the internal buffer must still fill completely.
	big := make([]byte, len(randBuf)*2+7)
	RandFill(big)
	allZero := true
	for _, x := range big {
		if x != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("RandFill left an over-buffer-sized slice all zero")
	}
}

func TestRandIntUniformAndBounds(t *testing.T) {
	// Every value in a small inclusive range appears over enough draws, and
	// none falls outside it.
	seen := map[int64]bool{}
	for i := 0; i < 4000; i++ {
		v, err := randIntFn(noCtx, []interpreter.Value{interpreter.IntVal(3), interpreter.IntVal(9)})
		if err != nil {
			t.Fatal(err)
		}
		if v.Int < 3 || v.Int > 9 {
			t.Fatalf("randInt out of range: %d", v.Int)
		}
		seen[v.Int] = true
	}
	for want := int64(3); want <= 9; want++ {
		if !seen[want] {
			t.Errorf("value %d never drawn over 4000 tries", want)
		}
	}
	// lo == hi is a fixed point; lo > hi errors.
	v, _ := randIntFn(noCtx, []interpreter.Value{interpreter.IntVal(5), interpreter.IntVal(5)})
	if v.Int != 5 {
		t.Errorf("randInt(5,5) = %d, want 5", v.Int)
	}
	if _, err := randIntFn(noCtx, []interpreter.Value{interpreter.IntVal(9), interpreter.IntVal(3)}); err == nil {
		t.Error("lo > hi should error")
	}
}

// The full int64 span must not overflow or spin forever (mod == full-range
// fast path).
func TestRandIntFullRangeDoesNotHang(t *testing.T) {
	for i := 0; i < 100; i++ {
		if _, err := randIntFn(noCtx, []interpreter.Value{interpreter.IntVal(-9223372036854775808), interpreter.IntVal(9223372036854775807)}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHmacEqual(t *testing.T) {
	cases := []struct {
		a, b []byte
		want bool
	}{
		{[]byte("abcdef"), []byte("abcdef"), true},
		{[]byte("abcdef"), []byte("abcdeg"), false},
		{[]byte("abc"), []byte("abcdef"), false},
		{[]byte{}, []byte{}, true},
		{[]byte{}, []byte("x"), false},
	}
	for _, c := range cases {
		v, err := hmacEqualFn(noCtx, []interpreter.Value{bytesArg(c.a), bytesArg(c.b)})
		if err != nil {
			t.Fatal(err)
		}
		if v.Bool != c.want {
			t.Errorf("hmacEqual(%q,%q) = %t, want %t", c.a, c.b, v.Bool, c.want)
		}
	}
	if _, err := hmacEqualFn(noCtx, []interpreter.Value{interpreter.StringVal("a"), bytesArg([]byte("a"))}); err == nil {
		t.Error("non-bytes argument should error")
	}
}

// RFC 5869 Appendix A.1 (HKDF-SHA256, Test Case 1).
func TestHkdfKnownAnswer(t *testing.T) {
	ikm := bytesRepeat(0x0b, 22)
	salt, _ := hex.DecodeString("000102030405060708090a0b0c")
	info, _ := hex.DecodeString("f0f1f2f3f4f5f6f7f8f9")
	wantHex := "3cb25f25faacd57a90434f64d0362f2a2d2d0a90cf1a5a4c5db02d56ecc4c5bf34007208d5b887185865"
	v, err := hkdfFn(noCtx, []interpreter.Value{bytesArg(ikm), bytesArg(salt), bytesArg(info), interpreter.IntVal(42), interpreter.StringVal("sha256")})
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(v.Bytes); got != wantHex {
		t.Errorf("hkdf =\n %s\nwant\n %s", got, wantHex)
	}
	if _, err := hkdfFn(noCtx, []interpreter.Value{bytesArg(ikm), bytesArg(salt), bytesArg(info), interpreter.IntVal(0), interpreter.StringVal("sha256")}); err == nil {
		t.Error("length 0 should error")
	}
	if _, err := hkdfFn(noCtx, []interpreter.Value{bytesArg(ikm), bytesArg(salt), bytesArg(info), interpreter.IntVal(42), interpreter.StringVal("md5")}); err == nil {
		t.Error("md5 is not a KDF algorithm and should error")
	}
}

// Standard PBKDF2 vectors: SHA-256 (P="password", S="salt", c=1, dkLen=32) and
// SHA-1 (RFC 6070 case 1, dkLen=20) - the latter is what SCRAM-SHA-1 needs.
func TestPbkdf2KnownAnswer(t *testing.T) {
	cases := []struct {
		algo         string
		iter, keyLen int64
		wantHex      string
	}{
		{"sha256", 1, 32, "120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b"},
		{"sha1", 1, 20, "0c60c80f961f0e71f3a9b524af6012062fe037a6"},
	}
	for _, c := range cases {
		v, err := pbkdf2Fn(noCtx, []interpreter.Value{bytesArg([]byte("password")), bytesArg([]byte("salt")), interpreter.IntVal(c.iter), interpreter.IntVal(c.keyLen), interpreter.StringVal(c.algo)})
		if err != nil {
			t.Fatal(err)
		}
		if got := hex.EncodeToString(v.Bytes); got != c.wantHex {
			t.Errorf("pbkdf(%s) =\n %s\nwant\n %s", c.algo, got, c.wantHex)
		}
	}
	if _, err := pbkdf2Fn(noCtx, []interpreter.Value{bytesArg([]byte("p")), bytesArg([]byte("s")), interpreter.IntVal(0), interpreter.IntVal(32), interpreter.StringVal("sha256")}); err == nil {
		t.Error("iterations 0 should error")
	}
}

// RandByte is the source uuid draws on, and a spawned task can generate UUIDs
// concurrently with the main goroutine, so the shared refill buffer must be
// race-free. Run under -race.
func TestRandByteConcurrent(t *testing.T) {
	const goroutines, draws = 16, 2000
	done := make(chan struct{}, goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			var acc byte
			for i := 0; i < draws; i++ {
				acc ^= RandByte()
			}
			_ = acc
			done <- struct{}{}
		}()
	}
	for g := 0; g < goroutines; g++ {
		<-done
	}
}

func bytesRepeat(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}
