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
