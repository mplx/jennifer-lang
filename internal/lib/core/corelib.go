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
// Pass 1 contents:
//   - JENNIFER_VERSION (string constant) - the interpreter's build version.
//     Underscored, language-prefixed name follows the PHP_VERSION /
//     RUBY_VERSION precedent and leaves room for future host/build
//     constants (JENNIFER_BUILDTIME, PLATFORM, OSNAME, ARCH).
//
// Pass 2 (during the M5 cleanup that introduced this library) adds:
//   - len(string | list | map) - polymorphic structural length.
//
// Reserve this library carefully. It is the escape hatch from Jennifer's
// "nothing for free" library discipline; it should hold a handful of
// universally-needed structural primitives and nothing more. Anything that
// could plausibly belong to a topic library (io, math, strings, ...) goes
// there instead.
//
// The Go package is named corelib to follow the convention used by the
// other libraries (iolib, mathlib, stringslib).
package corelib

import (
	"fmt"
	"unicode/utf8"

	"github.com/mplx/jennifer-lang/internal/interpreter"
	"github.com/mplx/jennifer-lang/internal/version"
)

// LibraryName is the Jennifer name the interpreter uses to track the core
// library. Programs do NOT write `use core;` - the interpreter pre-imports
// it and explicit imports are rejected.
const LibraryName = "core"

// Install registers the core library's builtins and constants on an
// interpreter. The caller is also expected to mark `core` as
// pre-imported; that happens in Interpreter.New, not here, so this
// installer remains a plain Register-only function consistent with the
// other libraries.
func Install(in *interpreter.Interpreter) {
	in.RegisterConst(LibraryName, "JENNIFER_VERSION", interpreter.StringVal(version.Version))
	in.Register(LibraryName, "len", lenFn)
	in.Register(LibraryName, "has", hasFn)
}

// lenFn returns the structural length of its argument. Polymorphic on
// every kind where "length" is well-defined:
//
//   - string -> rune count (Unicode code points, not bytes)
//   - list   -> element count
//   - map    -> entry count
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
	}
	return interpreter.Null(), fmt.Errorf("len() expects a string, list or map, got %s", v.Kind)
}

// hasFn reports whether a map contains a given key. The companion to
// the M6 decision that reads of missing keys are runtime errors: callers
// who need a non-erroring "does it exist?" check use `has($m, key)`.
//
// Only maps are accepted - "does this list contain this value?" is a
// different question (linear search; future strings library
// `contains(haystack, needle)` is the analogous string-side operation).
func hasFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if len(args) != 2 {
		return interpreter.Null(), fmt.Errorf("has() expects 2 arguments (map, key), got %d", len(args))
	}
	m := args[0]
	if m.Kind != interpreter.KindMap {
		return interpreter.Null(), fmt.Errorf("has() expects a map as the first argument, got %s", m.Kind)
	}
	key := args[1]
	for _, e := range m.Map {
		if e.Key.Equal(key) {
			return interpreter.BoolVal(true), nil
		}
	}
	return interpreter.BoolVal(false), nil
}
