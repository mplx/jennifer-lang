// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

//go:build !tinygo

// Standard-Go implementation of the `term` library, over golang.org/x/term
// (the same package the REPL's line editor uses). Raw mode and terminal size
// operate on the real terminal device (os.Stdin / os.Stdout / os.Stderr by fd);
// raw byte reads come from the interpreter's input reader, so they compose with
// the raw mode set on stdin and stay testable.
package termlib

import (
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/term"

	"jennifer-lang.dev/jennifer/internal/interpreter"
)

// rawEntry holds one active raw-mode session: the fd it was set on and the
// termios state to restore.
type rawEntry struct {
	fd   int
	prev *term.State
}

var (
	statesMu sync.Mutex
	states   = map[int]*rawEntry{}
	nextID   int
)

// ResetForTest clears the raw-mode registry between tests.
func ResetForTest() {
	statesMu.Lock()
	states = map[int]*rawEntry{}
	nextID = 0
	statesMu.Unlock()
}

// streamFile maps a stream name to its *os.File. Raw mode / size are properties
// of the terminal device, so they use the real standard streams by fd (like
// os.isTerminal), not the interpreter's possibly-redirected reader.
func streamFile(fnName, name string) (*os.File, error) {
	switch name {
	case "stdin":
		return os.Stdin, nil
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	}
	return nil, fmt.Errorf("%s: unknown stream %q; known: \"stdin\", \"stdout\", \"stderr\"", fnName, name)
}

func streamArg(fnName string, args []Value) (*os.File, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s expects 1 argument (stream), got %d", fnName, len(args))
	}
	if args[0].Kind != interpreter.KindString {
		return nil, fmt.Errorf("%s: stream must be string, got %s", fnName, args[0].Kind)
	}
	return streamFile(fnName, args[0].Str)
}

// makeRawFn implements term.makeRaw(stream) -> term.State. Raw mode disables line
// buffering and echo, so each keypress is delivered to term.readByte immediately.
// Refused inside the REPL, which owns the terminal for its own line editor.
func makeRawFn(ctx interpreter.BuiltinCtx, args []Value) (Value, error) {
	if ctx.InREPL {
		return interpreter.Null(), fmt.Errorf("term.makeRaw: not available inside the REPL (it owns the terminal)")
	}
	f, err := streamArg("term.makeRaw", args)
	if err != nil {
		return interpreter.Null(), err
	}
	fd := int(f.Fd())
	prev, rerr := term.MakeRaw(fd)
	if rerr != nil {
		return interpreter.Null(), fmt.Errorf("term.makeRaw: %s is not a terminal or cannot enter raw mode: %v", args[0].Str, rerr)
	}
	statesMu.Lock()
	id := nextID
	nextID++
	states[id] = &rawEntry{fd: fd, prev: prev}
	statesMu.Unlock()
	return interpreter.NamespacedStructVal(LibraryName, "State", []interpreter.StructField{
		{Name: "id", Value: interpreter.IntVal(int64(id))},
	}), nil
}

// restoreFn implements term.restore(state) -> null. The handle is consumed: a
// second restore of the same state is an error, so a cooked terminal is never
// clobbered by a stale handle.
func restoreFn(_ interpreter.BuiltinCtx, args []Value) (Value, error) {
	if len(args) != 1 {
		return interpreter.Null(), fmt.Errorf("term.restore expects 1 argument (state), got %d", len(args))
	}
	id, err := stateID("term.restore", args[0])
	if err != nil {
		return interpreter.Null(), err
	}
	statesMu.Lock()
	entry, ok := states[id]
	if ok {
		delete(states, id)
	}
	statesMu.Unlock()
	if !ok {
		return interpreter.Null(), fmt.Errorf("term.restore: state already restored or not a live handle")
	}
	if rerr := term.Restore(entry.fd, entry.prev); rerr != nil {
		return interpreter.Null(), fmt.Errorf("term.restore: %v", rerr)
	}
	return interpreter.Null(), nil
}

// sizeFn implements term.size(stream) -> term.Size{rows, cols}.
func sizeFn(_ interpreter.BuiltinCtx, args []Value) (Value, error) {
	f, err := streamArg("term.size", args)
	if err != nil {
		return interpreter.Null(), err
	}
	cols, rows, gerr := term.GetSize(int(f.Fd()))
	if gerr != nil {
		return interpreter.Null(), fmt.Errorf("term.size: %s is not a terminal: %v", args[0].Str, gerr)
	}
	return interpreter.NamespacedStructVal(LibraryName, "Size", []interpreter.StructField{
		{Name: "rows", Value: interpreter.IntVal(int64(rows))},
		{Name: "cols", Value: interpreter.IntVal(int64(cols))},
	}), nil
}

// readByteFn implements term.readByte() -> int: the next raw byte from stdin
// (0-255), or -1 at end of input. In raw mode this returns as soon as a key is
// pressed. Refused inside the REPL.
func readByteFn(ctx interpreter.BuiltinCtx, args []Value) (Value, error) {
	if ctx.InREPL {
		return interpreter.Null(), fmt.Errorf("term.readByte: not available inside the REPL")
	}
	if len(args) != 0 {
		return interpreter.Null(), fmt.Errorf("term.readByte expects 0 arguments, got %d", len(args))
	}
	if ctx.In == nil {
		return interpreter.IntVal(-1), nil
	}
	var buf [1]byte
	n, err := ctx.In.Read(buf[:])
	if n > 0 {
		return interpreter.IntVal(int64(buf[0])), nil
	}
	if err == nil || err == io.EOF {
		return interpreter.IntVal(-1), nil
	}
	return interpreter.Null(), fmt.Errorf("term.readByte: %v", err)
}

// stateID extracts the registry id from a term.State handle.
func stateID(fnName string, v Value) (int, error) {
	if v.Kind != interpreter.KindStruct || v.StructNS != LibraryName || v.StructName != "State" {
		return 0, fmt.Errorf("%s: argument must be a term.State, got %s", fnName, v.Kind)
	}
	for _, f := range v.Fields {
		if f.Name == "id" {
			return int(f.Value.Int), nil
		}
	}
	return 0, fmt.Errorf("%s: invalid term.State", fnName)
}
