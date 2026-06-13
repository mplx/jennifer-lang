// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

// Package corelib implements Jennifer's `core` library: the small set of
// structural builtins that every program needs without ceremony.
//
// `core` is special among libraries. It is auto-loaded by the interpreter
// at startup (see Interpreter.New) and writing `use core;` in source is a
// parse-time-equivalent error - the library is already available, and an
// explicit `use core;` signals confusion that's better surfaced loudly.
//
// Contents:
//   - len(string | list | map | bytes) - polymorphic structural length.
//
// M15.1 cleanup: `JENNIFER_VERSION` moved from core to the new `meta`
// library (`meta.JENNIFER_VERSION`). Core's charter is now strictly
// polymorphic structural primitives - interpreter-identity constants
// live in `meta`. Breaking change; pre-1.0 budget covers it.
//
// M9 cleanup: `has()` was removed from core and relocated to the
// `maps` library as `maps.has(m, key)`. Map membership testing is
// domain-specific (only meaningful on maps), so it didn't fit core's
// "universally-needed structural primitives" charter - `len()` stays
// because it is genuinely polymorphic across kinds.
//
// Reserve this library carefully. It is the escape hatch from Jennifer's
// "nothing for free" library discipline; it should hold a handful of
// universally-needed structural primitives and nothing more. Anything that
// could plausibly belong to a topic library (io, math, strings, meta, ...)
// goes there instead.
//
// The Go package is named corelib to follow the convention used by the
// other libraries (iolib, mathlib, stringslib).
package corelib

import (
	"fmt"
	"unicode/utf8"

	"github.com/mplx/jennifer-lang/internal/interpreter"
)

// LibraryName is the Jennifer name the interpreter uses to track the core
// library. Programs do NOT write `use core;` - the interpreter pre-imports
// it and explicit imports are rejected.
const LibraryName = "core"

// Install registers the core library's builtins on an interpreter. `core`
// is the only library that registers names via `RegisterGlobal`: `len` is
// genuinely polymorphic across string/list/map/bytes - the "nothing for
// free except this" exception. It is *only* exposed globally; no
// `core.len` qualified form exists, since publishing the same name two
// ways would violate stance #1 ("one way per thing").
func Install(in *interpreter.Interpreter) {
	in.RegisterGlobal(LibraryName, "len", lenFn)
}

// lenFn returns the structural length of its argument. Polymorphic on
// every kind where "length" is well-defined:
//
//   - string -> rune count (Unicode code points, not bytes)
//   - list   -> element count
//   - map    -> entry count
//   - bytes  -> byte count (M12)
//
// Any other kind is a positioned runtime error.
func lenFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if len(args) != 1 {
		return interpreter.Null(), fmt.Errorf("len() expects 1 argument, got %d", len(args))
	}
	v := args[0]
	switch v.Kind {
	case interpreter.KindString:
		return interpreter.IntVal(int64(utf8.RuneCountInString(v.Str))), nil
	case interpreter.KindList:
		return interpreter.IntVal(int64(len(v.List))), nil
	case interpreter.KindMap:
		return interpreter.IntVal(int64(len(v.Map))), nil
	case interpreter.KindBytes:
		return interpreter.IntVal(int64(len(v.Bytes))), nil
	}
	return interpreter.Null(), fmt.Errorf("len() expects a string, list, map or bytes, got %s", v.Kind)
}

