// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package iolib

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/mplx/jennifer-lang/internal/interpreter"
)

// Package-level stdin state. The interpreter is single-instance per
// process today, so this is safe. If multi-interpreter ever lands, move
// these onto the Interpreter struct (or pass through BuiltinCtx).
//
// The buffered reader has to persist across calls because `eof()` may
// Peek a byte that a subsequent `readLine()` needs to see; throwing
// the reader away each call would drop the byte. `eofState` is sticky
// once true so the EOF error message is consistent across repeated
// calls past end-of-input.
var (
	cachedSrc io.Reader
	bufIn     *bufio.Reader
	eofState  bool
)

// resetInputForTest clears the cached reader and EOF flag. Exported via
// the test-only helper in input_test.go.
func resetInputForTest() {
	cachedSrc = nil
	bufIn = nil
	eofState = false
}

// getReader returns the buffered wrapper for `in`, rebuilding it if the
// underlying source has changed since the last call. A change of source
// also clears the sticky EOF flag - tests substitute new stdin contents
// per case.
func getReader(in io.Reader) *bufio.Reader {
	if in == nil {
		return nil
	}
	if in != cachedSrc {
		cachedSrc = in
		bufIn = bufio.NewReader(in)
		eofState = false
	}
	return bufIn
}

// readLine reads one line from stdin and returns it with the trailing
// newline (`\r\n` or `\n`) stripped. With one string argument the prompt
// is written to stdout first. Calling at end-of-input is a positioned
// runtime error - callers should check `eof()` first.
//
// A final line without a trailing newline is returned on the call that
// reaches it; the subsequent call errors.
func readLine(ctx interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if ctx.InREPL {
		return interpreter.Null(), fmt.Errorf("readLine: stdin is owned by the REPL editor")
	}
	if len(args) > 1 {
		return interpreter.Null(), fmt.Errorf("`readLine` takes 0 or 1 argument, got %d", len(args))
	}
	if len(args) == 1 {
		if args[0].Kind != interpreter.KindString {
			return interpreter.Null(), fmt.Errorf("`readLine` prompt must be string, got %s", args[0].Kind)
		}
		if _, err := io.WriteString(ctx.Out, args[0].Str); err != nil {
			return interpreter.Null(), fmt.Errorf("readLine: writing prompt: %v", err)
		}
	}
	r := getReader(ctx.In)
	if r == nil {
		return interpreter.Null(), fmt.Errorf("readLine: no input source")
	}
	if eofState {
		return interpreter.Null(), fmt.Errorf("readLine: end of input")
	}
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return interpreter.Null(), fmt.Errorf("readLine: %v", err)
	}
	if err == io.EOF {
		eofState = true
		if line == "" {
			return interpreter.Null(), fmt.Errorf("readLine: end of input")
		}
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return interpreter.StringVal(line), nil
}

// eofFn reports whether the next `readLine()` would error. Implementation
// peeks one byte through the buffered reader; the byte stays in the
// buffer for the next read. Once true, eofState is sticky for the rest
// of the run.
func eofFn(ctx interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if ctx.InREPL {
		return interpreter.Null(), fmt.Errorf("eof: stdin is owned by the REPL editor")
	}
	if len(args) != 0 {
		return interpreter.Null(), fmt.Errorf("`eof` takes no arguments, got %d", len(args))
	}
	if eofState {
		return interpreter.BoolVal(true), nil
	}
	r := getReader(ctx.In)
	if r == nil {
		return interpreter.BoolVal(true), nil
	}
	_, err := r.Peek(1)
	if err == io.EOF {
		eofState = true
		return interpreter.BoolVal(true), nil
	}
	if err != nil {
		return interpreter.Null(), fmt.Errorf("eof: %v", err)
	}
	return interpreter.BoolVal(false), nil
}
