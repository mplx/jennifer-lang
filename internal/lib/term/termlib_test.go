// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

//go:build !tinygo

package termlib

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/term"

	"jennifer-lang.dev/jennifer/internal/interpreter"
)

func TestReadByte(t *testing.T) {
	ctx := interpreter.BuiltinCtx{In: strings.NewReader("AB")}
	for _, want := range []int64{'A', 'B', -1, -1} {
		v, err := readByteFn(ctx, nil)
		if err != nil {
			t.Fatalf("readByte: %v", err)
		}
		if v.Kind != interpreter.KindInt || v.Int != want {
			t.Errorf("readByte = %v, want %d", v, want)
		}
	}
}

// TestReadByteBulk drains a large stream one byte at a time. term.readByte reads
// into a fixed 1-byte buffer, so memory is flat regardless of input size; this
// confirms it returns every byte in order and terminates cleanly at -1 (no hang,
// no accumulation).
func TestReadByteBulk(t *testing.T) {
	const n = 100000
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = byte(i * 7)
	}
	ctx := interpreter.BuiltinCtx{In: strings.NewReader(string(buf))}
	for i := 0; i < n; i++ {
		v, err := readByteFn(ctx, nil)
		if err != nil {
			t.Fatalf("readByte[%d]: %v", i, err)
		}
		if v.Int != int64(buf[i]) {
			t.Fatalf("readByte[%d] = %d, want %d", i, v.Int, buf[i])
		}
	}
	// Exactly drained: the next read is -1, and stays -1.
	for k := 0; k < 3; k++ {
		v, _ := readByteFn(ctx, nil)
		if v.Int != -1 {
			t.Fatalf("post-drain read = %d, want -1", v.Int)
		}
	}
}

func TestReadByteRefusedInREPL(t *testing.T) {
	ctx := interpreter.BuiltinCtx{In: strings.NewReader("A"), InREPL: true}
	if _, err := readByteFn(ctx, nil); err == nil {
		t.Error("readByte in the REPL should error")
	}
}

func TestReadByteArity(t *testing.T) {
	ctx := interpreter.BuiltinCtx{In: strings.NewReader("A")}
	if _, err := readByteFn(ctx, []Value{interpreter.StringVal("x")}); err == nil {
		t.Error("readByte with an argument should error")
	}
}

func TestMakeRawRefusedInREPL(t *testing.T) {
	ctx := interpreter.BuiltinCtx{InREPL: true}
	if _, err := makeRawFn(ctx, []Value{interpreter.StringVal("stdin")}); err == nil {
		t.Error("makeRaw in the REPL should error")
	}
}

func TestUnknownStream(t *testing.T) {
	for _, fn := range []func(interpreter.BuiltinCtx, []Value) (Value, error){makeRawFn, sizeFn} {
		_, err := fn(interpreter.BuiltinCtx{}, []Value{interpreter.StringVal("nope")})
		if err == nil || !strings.Contains(err.Error(), "unknown stream") {
			t.Errorf("expected an unknown-stream error, got %v", err)
		}
	}
}

func TestStreamArity(t *testing.T) {
	if _, err := sizeFn(interpreter.BuiltinCtx{}, nil); err == nil {
		t.Error("size with no arguments should error")
	}
	if _, err := makeRawFn(interpreter.BuiltinCtx{}, []Value{interpreter.IntVal(1)}); err == nil {
		t.Error("makeRaw with a non-string stream should error")
	}
}

func TestRestoreInvalid(t *testing.T) {
	ResetForTest()
	// A non-State value is rejected.
	if _, err := restoreFn(interpreter.BuiltinCtx{}, []Value{interpreter.IntVal(1)}); err == nil {
		t.Error("restore of a non-State value should error")
	}
	// A State whose id is not in the registry (stale / already restored).
	stale := interpreter.NamespacedStructVal(LibraryName, "State", []interpreter.StructField{
		{Name: "id", Value: interpreter.IntVal(999)},
	})
	if _, err := restoreFn(interpreter.BuiltinCtx{}, []Value{stale}); err == nil {
		t.Error("restore of a stale State should error")
	}
}

// TestMakeRawNonTTY exercises the real MakeRaw failure path only when stdin is
// not an interactive terminal (CI, a pipe) - never on a developer's live
// terminal, which raw mode would disrupt.
func TestMakeRawNonTTY(t *testing.T) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		t.Skip("stdin is an interactive terminal; skipping to avoid disrupting it")
	}
	if _, err := makeRawFn(interpreter.BuiltinCtx{}, []Value{interpreter.StringVal("stdin")}); err == nil {
		t.Error("makeRaw on a non-terminal stdin should error")
	}
}

// TestSize accepts either outcome: on a real terminal it returns a well-formed
// term.Size (querying size does not change terminal state); off one it errors.
func TestSize(t *testing.T) {
	v, err := sizeFn(interpreter.BuiltinCtx{}, []Value{interpreter.StringVal("stdout")})
	if err != nil {
		if !strings.Contains(err.Error(), "not a terminal") {
			t.Errorf("size error = %v, want a not-a-terminal message", err)
		}
		return
	}
	if v.Kind != interpreter.KindStruct || v.StructName != "Size" {
		t.Fatalf("size = %v, want a term.Size struct", v)
	}
	var haveRows, haveCols bool
	for _, f := range v.Fields {
		switch f.Name {
		case "rows":
			haveRows = true
		case "cols":
			haveCols = true
		}
	}
	if !haveRows || !haveCols {
		t.Errorf("term.Size missing rows/cols: %v", v.Fields)
	}
}
