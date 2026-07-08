// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package archivelib

import (
	"testing"

	"github.com/mplx/jennifer-lang/internal/interpreter"
	"github.com/mplx/jennifer-lang/internal/parser"
)

func str(s string) interpreter.Value { return interpreter.StringVal(s) }

func entriesVal(es []entry) interpreter.Value {
	vs := make([]interpreter.Value, len(es))
	for i, e := range es {
		vs[i] = makeEntry(e)
	}
	return interpreter.ListVal(parser.NamespacedStructType(LibraryName, "Entry"), vs)
}

func roundtrip(t *testing.T, format string, in []entry) []entry {
	t.Helper()
	packed, err := packFn(interpreter.BuiltinCtx{}, []interpreter.Value{entriesVal(in), str(format)})
	if err != nil {
		t.Fatalf("%s pack: %v", format, err)
	}
	if packed.Kind != interpreter.KindBytes {
		t.Fatalf("%s: pack returned %s, want bytes", format, packed.Kind)
	}
	back, err := unpackFn(interpreter.BuiltinCtx{}, []interpreter.Value{packed, str(format)})
	if err != nil {
		t.Fatalf("%s unpack: %v", format, err)
	}
	out := make([]entry, len(back.List))
	for i, v := range back.List {
		e, err := extractEntry(i, v)
		if err != nil {
			t.Fatalf("%s extract %d: %v", format, i, err)
		}
		out[i] = e
	}
	return out
}

func TestRoundTrip(t *testing.T) {
	in := []entry{
		{name: "a.txt", data: []byte("alpha"), mode: 0o644, mtime: 1700000000},
		{name: "dir/b.txt", data: []byte("bravo bravo bravo"), mode: 0o600, mtime: 1700000000},
	}
	for _, format := range []string{"tar", "zip", "tar.gz", "tgz"} {
		out := roundtrip(t, format, in)
		if len(out) != len(in) {
			t.Fatalf("%s: got %d entries, want %d", format, len(out), len(in))
		}
		for i := range in {
			if out[i].name != in[i].name {
				t.Errorf("%s[%d] name = %q, want %q", format, i, out[i].name, in[i].name)
			}
			if string(out[i].data) != string(in[i].data) {
				t.Errorf("%s[%d] data = %q, want %q", format, i, out[i].data, in[i].data)
			}
		}
	}
}

func TestModeAndMtimePreserved(t *testing.T) {
	in := []entry{{name: "x", data: []byte("d"), mode: 0o600, mtime: 1700000000}}
	// tar preserves both exactly.
	tarOut := roundtrip(t, "tar", in)
	if tarOut[0].mode&0o777 != 0o600 {
		t.Errorf("tar mode = %o, want 600", tarOut[0].mode&0o777)
	}
	if tarOut[0].mtime != 1700000000 {
		t.Errorf("tar mtime = %d, want 1700000000", tarOut[0].mtime)
	}
	// zip preserves the permission bits (mtime uses the DOS field + an
	// extended timestamp; the permission round-trip is the stable assertion).
	zipOut := roundtrip(t, "zip", in)
	if zipOut[0].mode&0o777 != 0o600 {
		t.Errorf("zip mode = %o, want 600", zipOut[0].mode&0o777)
	}
}

func TestDefaultMode(t *testing.T) {
	out := roundtrip(t, "tar", []entry{{name: "x", data: []byte("d"), mode: 0}})
	if out[0].mode&0o777 != defaultMode {
		t.Errorf("default mode = %o, want %o", out[0].mode&0o777, defaultMode)
	}
}

func TestTarGzIsGzip(t *testing.T) {
	packed, err := packFn(interpreter.BuiltinCtx{}, []interpreter.Value{
		entriesVal([]entry{{name: "x", data: []byte("d")}}), str("tar.gz")})
	if err != nil {
		t.Fatal(err)
	}
	if len(packed.Bytes) < 2 || packed.Bytes[0] != 0x1f || packed.Bytes[1] != 0x8b {
		t.Errorf("tar.gz output is not gzip-framed: %x", packed.Bytes[:min(4, len(packed.Bytes))])
	}
}

func TestEmptyArchive(t *testing.T) {
	if out := roundtrip(t, "tar", nil); len(out) != 0 {
		t.Errorf("empty tar produced %d entries", len(out))
	}
	if out := roundtrip(t, "zip", nil); len(out) != 0 {
		t.Errorf("empty zip produced %d entries", len(out))
	}
}

func TestErrors(t *testing.T) {
	entries := entriesVal([]entry{{name: "x", data: []byte("d")}})
	ctx := interpreter.BuiltinCtx{}
	if _, err := packFn(ctx, []interpreter.Value{entries, str("rar")}); err == nil {
		t.Error("pack: unknown format should error")
	}
	if _, err := unpackFn(ctx, []interpreter.Value{interpreter.BytesVal([]byte("x")), str("rar")}); err == nil {
		t.Error("unpack: unknown format should error")
	}
	if _, err := packFn(ctx, []interpreter.Value{str("not a list"), str("tar")}); err == nil {
		t.Error("pack: non-list first argument should error")
	}
	// a list whose element is not an archive.Entry
	badList := interpreter.ListVal(parser.NamespacedStructType(LibraryName, "Entry"), []interpreter.Value{interpreter.IntVal(3)})
	if _, err := packFn(ctx, []interpreter.Value{badList, str("tar")}); err == nil {
		t.Error("pack: non-Entry list element should error")
	}
	if _, err := unpackFn(ctx, []interpreter.Value{interpreter.BytesVal([]byte("not a zip file")), str("zip")}); err == nil {
		t.Error("unpack: corrupt zip should error")
	}
	if _, err := unpackFn(ctx, []interpreter.Value{str("x"), str("tar")}); err == nil {
		t.Error("unpack: non-bytes first argument should error")
	}
}
