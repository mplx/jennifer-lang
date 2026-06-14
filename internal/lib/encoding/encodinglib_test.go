// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package encodinglib

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mplx/jennifer-lang/internal/interpreter"
)

// callFn invokes a builtin with the given Values, returning the
// result Value. Failure shape: t.Fatal so individual cases stay tidy.
func callFn(t *testing.T, fn interpreter.Builtin, args ...interpreter.Value) interpreter.Value {
	t.Helper()
	v, err := fn(interpreter.BuiltinCtx{}, args)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	return v
}

// expectErr invokes a builtin expecting an error and returns the
// error message for substring checks.
func expectErr(t *testing.T, fn interpreter.Builtin, args ...interpreter.Value) string {
	t.Helper()
	_, err := fn(interpreter.BuiltinCtx{}, args)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	return err.Error()
}

// ----- introspection ------------------------------------------------

func TestIsAscii(t *testing.T) {
	if v := callFn(t, isAsciiFn, interpreter.BytesVal([]byte("hello"))); v.Bool != true {
		t.Errorf(`isAscii("hello") = %v, want true`, v.Bool)
	}
	if v := callFn(t, isAsciiFn, interpreter.BytesVal([]byte{0x80})); v.Bool != false {
		t.Errorf(`isAscii(0x80) = %v, want false`, v.Bool)
	}
	if v := callFn(t, isAsciiFn, interpreter.BytesVal(nil)); v.Bool != true {
		t.Errorf(`isAscii(empty) = %v, want true`, v.Bool)
	}
}

func TestLenBytesVsLenRunes(t *testing.T) {
	// "café" is 5 UTF-8 bytes (c, a, f, 0xC3 0xA9) but 4 runes.
	bytesVal := callFn(t, lenBytesFn, interpreter.StringVal("café"))
	if bytesVal.Int != 5 {
		t.Errorf(`lenBytes("café") = %d, want 5`, bytesVal.Int)
	}
	runesVal := callFn(t, lenRunesFn, interpreter.BytesVal([]byte("café")))
	if runesVal.Int != 4 {
		t.Errorf(`lenRunes("café".bytes) = %d, want 4`, runesVal.Int)
	}
}

func TestLenRunesRejectsInvalidUTF8(t *testing.T) {
	msg := expectErr(t, lenRunesFn, interpreter.BytesVal([]byte{0xFF, 0xFE}))
	if !strings.Contains(msg, "valid UTF-8") {
		t.Errorf("err does not mention UTF-8 validity: %v", msg)
	}
}

// ----- toText / fromText (hex, base64, base64-url) ------------------

func TestToTextHexRoundTrip(t *testing.T) {
	enc := callFn(t, toTextFn, interpreter.BytesVal([]byte{0xDE, 0xAD, 0xBE, 0xEF}), interpreter.StringVal("hex"))
	if enc.Str != "deadbeef" {
		t.Errorf("hex got %q, want %q", enc.Str, "deadbeef")
	}
	dec := callFn(t, fromTextFn, interpreter.StringVal("DEADBEEF"), interpreter.StringVal("hex"))
	if !bytes.Equal(dec.Bytes, []byte{0xDE, 0xAD, 0xBE, 0xEF}) {
		t.Errorf("fromText hex (uppercase) = %x, want deadbeef", dec.Bytes)
	}
}

func TestToTextEmptyRoundTrip(t *testing.T) {
	for _, fmtName := range []string{"hex", "base64", "base64-url"} {
		enc := callFn(t, toTextFn, interpreter.BytesVal(nil), interpreter.StringVal(fmtName))
		if enc.Str != "" {
			t.Errorf("toText(empty, %q) = %q, want empty", fmtName, enc.Str)
		}
		dec := callFn(t, fromTextFn, interpreter.StringVal(""), interpreter.StringVal(fmtName))
		if len(dec.Bytes) != 0 {
			t.Errorf("fromText(empty, %q) = %x, want empty", fmtName, dec.Bytes)
		}
	}
}

func TestFromTextHexRejectsOddLength(t *testing.T) {
	msg := expectErr(t, fromTextFn, interpreter.StringVal("abc"), interpreter.StringVal("hex"))
	if !strings.Contains(msg, "encoding.fromText") {
		t.Errorf("err lacks fn name: %v", msg)
	}
}

func TestToTextBase64StandardRoundTrip(t *testing.T) {
	// "Hello" -> "SGVsbG8=" (RFC 4648 standard).
	enc := callFn(t, toTextFn, interpreter.BytesVal([]byte("Hello")), interpreter.StringVal("base64"))
	if enc.Str != "SGVsbG8=" {
		t.Errorf("base64 got %q, want SGVsbG8=", enc.Str)
	}
	dec := callFn(t, fromTextFn, interpreter.StringVal("SGVsbG8="), interpreter.StringVal("base64"))
	if string(dec.Bytes) != "Hello" {
		t.Errorf("fromText base64 got %q, want Hello", string(dec.Bytes))
	}
}

// TestBase64UrlSafeDistinguishes confirms the format string routes to
// the right alphabet. Input bytes 0xFF 0xFE produce different strings
// under the two variants because the standard alphabet uses `+` and
// `/` while url-safe uses `-` and `_`.
func TestBase64UrlSafeDistinguishes(t *testing.T) {
	input := []byte{0xFF, 0xFE}
	std := callFn(t, toTextFn, interpreter.BytesVal(input), interpreter.StringVal("base64")).Str
	url := callFn(t, toTextFn, interpreter.BytesVal(input), interpreter.StringVal("base64-url")).Str
	if std == url {
		t.Fatalf("standard and url-safe produced same output: %q", std)
	}
	if !strings.ContainsAny(std, "+/") {
		t.Errorf("standard output %q missing +/ chars", std)
	}
	if strings.ContainsAny(url, "+/") {
		t.Errorf("url-safe output %q contains +/", url)
	}
	dec := callFn(t, fromTextFn, interpreter.StringVal(url), interpreter.StringVal("base64-url"))
	if !bytes.Equal(dec.Bytes, input) {
		t.Errorf("url-safe round-trip mismatch: %x vs %x", dec.Bytes, input)
	}
}

func TestToTextUnknownFormat(t *testing.T) {
	msg := expectErr(t, toTextFn, interpreter.BytesVal(nil), interpreter.StringVal("morse"))
	if !strings.Contains(msg, `unknown text format "morse"`) {
		t.Errorf("err does not name the bad format: %v", msg)
	}
}

// ----- codec table --------------------------------------------------

func TestCodecsLists(t *testing.T) {
	v := callFn(t, codecsFn)
	want := []string{"ascii", "latin-1", "windows-1252", "ebcdic"}
	if v.Kind != interpreter.KindList || len(v.List) != len(want) {
		t.Fatalf("codecs() = %v", v)
	}
	for i, s := range want {
		if v.List[i].Str != s {
			t.Errorf("codecs()[%d] = %q, want %q", i, v.List[i].Str, s)
		}
	}
}

func TestCodecAliases(t *testing.T) {
	// Normalisation: case, hyphens, underscores, spaces.
	cases := []struct {
		alias string
		want  string
	}{
		{"ascii", "ascii"},
		{"ASCII", "ascii"},
		{"us-ascii", "ascii"},
		{"latin-1", "latin-1"},
		{"latin_1", "latin-1"},
		{"ISO-8859-1", "latin-1"},
		{"iso88591", "latin-1"},
		{"windows-1252", "windows-1252"},
		{"cp1252", "windows-1252"},
		{"MS 1252", "windows-1252"},
		{"ebcdic", "ebcdic"},
		{"IBM-1047", "ebcdic"},
	}
	for _, tc := range cases {
		c, ok := lookupCodec(tc.alias)
		if !ok {
			t.Errorf("%q: not found", tc.alias)
			continue
		}
		if c.canonical != tc.want {
			t.Errorf("%q -> %q, want %q", tc.alias, c.canonical, tc.want)
		}
	}
}

func TestEncodeDecodeAsciiRoundTrip(t *testing.T) {
	enc := callFn(t, encodeFn, interpreter.StringVal("hello"), interpreter.StringVal("ascii"))
	if !bytes.Equal(enc.Bytes, []byte("hello")) {
		t.Errorf("encode ascii: %x", enc.Bytes)
	}
	dec := callFn(t, decodeFn, interpreter.BytesVal([]byte("hello")), interpreter.StringVal("ascii"))
	if dec.Str != "hello" {
		t.Errorf("decode ascii: %q", dec.Str)
	}
}

func TestAsciiRejectsHighRune(t *testing.T) {
	msg := expectErr(t, encodeFn, interpreter.StringVal("café"), interpreter.StringVal("ascii"))
	if !strings.Contains(msg, "ASCII") {
		t.Errorf("err lacks ASCII mention: %v", msg)
	}
}

func TestAsciiRejectsHighByteOnDecode(t *testing.T) {
	msg := expectErr(t, decodeFn, interpreter.BytesVal([]byte{'h', 'i', 0xE9}), interpreter.StringVal("ascii"))
	if !strings.Contains(msg, "ASCII") {
		t.Errorf("err lacks ASCII mention: %v", msg)
	}
}

func TestLatin1RoundTrip(t *testing.T) {
	// "café" -> 0x63 0x61 0x66 0xE9 in Latin-1.
	enc := callFn(t, encodeFn, interpreter.StringVal("café"), interpreter.StringVal("latin-1"))
	if !bytes.Equal(enc.Bytes, []byte{0x63, 0x61, 0x66, 0xE9}) {
		t.Errorf("encode latin-1: %x", enc.Bytes)
	}
	dec := callFn(t, decodeFn, interpreter.BytesVal([]byte{0x63, 0x61, 0x66, 0xE9}), interpreter.StringVal("latin-1"))
	if dec.Str != "café" {
		t.Errorf("decode latin-1: %q", dec.Str)
	}
}

func TestLatin1RejectsBmpRune(t *testing.T) {
	msg := expectErr(t, encodeFn, interpreter.StringVal("€100"), interpreter.StringVal("latin-1"))
	if !strings.Contains(msg, "Latin-1") {
		t.Errorf("err lacks Latin-1 mention: %v", msg)
	}
}

func TestWindows1252EuroDecode(t *testing.T) {
	// 0x80 in Windows-1252 is EURO SIGN; same byte in Latin-1 is a
	// C1 control char. This is the textbook reason the two codecs
	// are distinct.
	dec := callFn(t, decodeFn, interpreter.BytesVal([]byte{0x80, '1', '0', '0'}), interpreter.StringVal("windows-1252"))
	if dec.Str != "€100" {
		t.Errorf("decode 0x80 = %q, want €100", dec.Str)
	}
	enc := callFn(t, encodeFn, interpreter.StringVal("€100"), interpreter.StringVal("windows-1252"))
	if !bytes.Equal(enc.Bytes, []byte{0x80, '1', '0', '0'}) {
		t.Errorf("encode €100 = %x", enc.Bytes)
	}
}

func TestWindows1252UndefinedByte(t *testing.T) {
	// 0x81 is one of the five canonical-undefined positions.
	msg := expectErr(t, decodeFn, interpreter.BytesVal([]byte{0x81}), interpreter.StringVal("windows-1252"))
	if !strings.Contains(msg, "no mapping") {
		t.Errorf("err lacks mapping language: %v", msg)
	}
}

func TestEbcdicHelloRoundTrip(t *testing.T) {
	// EBCDIC IBM-1047: "Hello" = 0xC8 0x85 0x93 0x93 0x96
	enc := callFn(t, encodeFn, interpreter.StringVal("Hello"), interpreter.StringVal("ebcdic"))
	want := []byte{0xC8, 0x85, 0x93, 0x93, 0x96}
	if !bytes.Equal(enc.Bytes, want) {
		t.Errorf("encode Hello -> %x, want %x", enc.Bytes, want)
	}
	dec := callFn(t, decodeFn, interpreter.BytesVal(want), interpreter.StringVal("ebcdic"))
	if dec.Str != "Hello" {
		t.Errorf("decode = %q", dec.Str)
	}
}

func TestEbcdicLatin1Range(t *testing.T) {
	// EBCDIC IBM-1047 covers Latin-1 so "café" must round-trip.
	enc := callFn(t, encodeFn, interpreter.StringVal("café"), interpreter.StringVal("ebcdic"))
	dec := callFn(t, decodeFn, interpreter.BytesVal(enc.Bytes), interpreter.StringVal("ebcdic"))
	if dec.Str != "café" {
		t.Errorf("ebcdic café round-trip: enc=%x dec=%q", enc.Bytes, dec.Str)
	}
}

func TestUnknownCodec(t *testing.T) {
	msg := expectErr(t, encodeFn, interpreter.StringVal("hi"), interpreter.StringVal("klingon"))
	if !strings.Contains(msg, `unknown codec "klingon"`) {
		t.Errorf("err lacks unknown-codec phrasing: %v", msg)
	}
	if !strings.Contains(msg, "ascii") {
		t.Errorf("err does not list known codecs: %v", msg)
	}
}

func TestEncodeWrongScalarType(t *testing.T) {
	msg := expectErr(t, encodeFn, interpreter.BytesVal(nil), interpreter.StringVal("ascii"))
	if !strings.Contains(msg, "must be string") {
		t.Errorf("err lacks type complaint: %v", msg)
	}
}

func TestDecodeWrongScalarType(t *testing.T) {
	msg := expectErr(t, decodeFn, interpreter.StringVal("hi"), interpreter.StringVal("ascii"))
	if !strings.Contains(msg, "must be bytes") {
		t.Errorf("err lacks type complaint: %v", msg)
	}
}
