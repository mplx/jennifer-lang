// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

// Package stringslib implements Jennifer's `strings` library: text utilities
// like length, case conversion, search, trim, and substring extraction. All
// operations that involve indices or lengths work on Unicode code points
// (runes), not bytes - so `len("héllo")` is 5, not 6.
//
// The Go package is named stringslib to avoid colliding with Go's standard
// `strings` package, which this implementation depends on.
package stringslib

import (
	"fmt"
	"io"
	gostrings "strings"
	"unicode"
	"unicode/utf8"

	"github.com/mplx/jennifer-lang/internal/interpreter"
)

// LibraryName is the Jennifer name programs `use` to enable these functions.
const LibraryName = "strings"

// Install registers strings library functions on an interpreter.
func Install(in *interpreter.Interpreter) {
	in.Register(LibraryName, "len", lenFn)
	in.Register(LibraryName, "upper", upperFn)
	in.Register(LibraryName, "lower", lowerFn)
	in.Register(LibraryName, "contains", containsFn)
	in.Register(LibraryName, "startsWith", startsWithFn)
	in.Register(LibraryName, "endsWith", endsWithFn)
	in.Register(LibraryName, "indexOf", indexOfFn)
	in.Register(LibraryName, "trim", trimFn)
	in.Register(LibraryName, "trimLeft", trimLeftFn)
	in.Register(LibraryName, "trimRight", trimRightFn)
	in.Register(LibraryName, "replace", replaceFn)
	in.Register(LibraryName, "repeat", repeatFn)
	in.Register(LibraryName, "substring", substringFn)
}

// ---- helpers ----

func arityN(name string, args []interpreter.Value, want int) error {
	if len(args) != want {
		return fmt.Errorf("%s expects %d argument(s), got %d", name, want, len(args))
	}
	return nil
}

// requireString returns the string value at args[i] or an error mentioning
// the position and function name.
func requireString(name string, args []interpreter.Value, i int) (string, error) {
	v := args[i]
	if v.Kind != interpreter.KindString {
		return "", fmt.Errorf("%s(): argument %d must be string, got %s", name, i+1, v.Kind)
	}
	return v.Str, nil
}

// requireInt returns the int value at args[i] or an error.
func requireInt(name string, args []interpreter.Value, i int) (int64, error) {
	v := args[i]
	if v.Kind != interpreter.KindInt {
		return 0, fmt.Errorf("%s(): argument %d must be int, got %s", name, i+1, v.Kind)
	}
	return v.Int, nil
}

// runeIndex converts a byte offset in s to a rune index (0-based).
// Returns -1 if byteOffset == -1.
func runeIndex(s string, byteOffset int) int {
	if byteOffset < 0 {
		return -1
	}
	return utf8.RuneCountInString(s[:byteOffset])
}

// byteOffsetForRune returns the byte position of the i'th rune in s. If i ==
// runeCount(s), returns len(s). Errors if i is out of range.
func byteOffsetForRune(s string, i int) (int, error) {
	if i < 0 {
		return 0, fmt.Errorf("negative index %d", i)
	}
	idx := 0
	for byteOff := 0; byteOff < len(s); idx++ {
		if idx == i {
			return byteOff, nil
		}
		_, size := utf8.DecodeRuneInString(s[byteOff:])
		byteOff += size
	}
	if idx == i {
		return len(s), nil
	}
	return 0, fmt.Errorf("index %d out of range (length %d)", i, idx)
}

// ---- functions ----

// lenFn returns the rune count of s.
func lenFn(_ io.Writer, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityN("len", args, 1); err != nil {
		return interpreter.Null(), err
	}
	s, err := requireString("len", args, 0)
	if err != nil {
		return interpreter.Null(), err
	}
	return interpreter.IntVal(int64(utf8.RuneCountInString(s))), nil
}

// upperFn returns s with all letters uppercased (Unicode-aware).
func upperFn(_ io.Writer, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityN("upper", args, 1); err != nil {
		return interpreter.Null(), err
	}
	s, err := requireString("upper", args, 0)
	if err != nil {
		return interpreter.Null(), err
	}
	return interpreter.StringVal(gostrings.ToUpper(s)), nil
}

// lowerFn returns s with all letters lowercased (Unicode-aware).
func lowerFn(_ io.Writer, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityN("lower", args, 1); err != nil {
		return interpreter.Null(), err
	}
	s, err := requireString("lower", args, 0)
	if err != nil {
		return interpreter.Null(), err
	}
	return interpreter.StringVal(gostrings.ToLower(s)), nil
}

// containsFn reports whether sub appears anywhere in s.
func containsFn(_ io.Writer, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityN("contains", args, 2); err != nil {
		return interpreter.Null(), err
	}
	s, err := requireString("contains", args, 0)
	if err != nil {
		return interpreter.Null(), err
	}
	sub, err := requireString("contains", args, 1)
	if err != nil {
		return interpreter.Null(), err
	}
	return interpreter.BoolVal(gostrings.Contains(s, sub)), nil
}

// startsWithFn reports whether s begins with prefix.
func startsWithFn(_ io.Writer, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityN("startsWith", args, 2); err != nil {
		return interpreter.Null(), err
	}
	s, err := requireString("startsWith", args, 0)
	if err != nil {
		return interpreter.Null(), err
	}
	prefix, err := requireString("startsWith", args, 1)
	if err != nil {
		return interpreter.Null(), err
	}
	return interpreter.BoolVal(gostrings.HasPrefix(s, prefix)), nil
}

// endsWithFn reports whether s ends with suffix.
func endsWithFn(_ io.Writer, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityN("endsWith", args, 2); err != nil {
		return interpreter.Null(), err
	}
	s, err := requireString("endsWith", args, 0)
	if err != nil {
		return interpreter.Null(), err
	}
	suffix, err := requireString("endsWith", args, 1)
	if err != nil {
		return interpreter.Null(), err
	}
	return interpreter.BoolVal(gostrings.HasSuffix(s, suffix)), nil
}

// indexOfFn returns the rune index of the first occurrence of sub in s, or
// -1 if sub is not present.
func indexOfFn(_ io.Writer, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityN("indexOf", args, 2); err != nil {
		return interpreter.Null(), err
	}
	s, err := requireString("indexOf", args, 0)
	if err != nil {
		return interpreter.Null(), err
	}
	sub, err := requireString("indexOf", args, 1)
	if err != nil {
		return interpreter.Null(), err
	}
	byteIdx := gostrings.Index(s, sub)
	return interpreter.IntVal(int64(runeIndex(s, byteIdx))), nil
}

// trimFn strips leading and trailing whitespace (Unicode-aware).
func trimFn(_ io.Writer, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityN("trim", args, 1); err != nil {
		return interpreter.Null(), err
	}
	s, err := requireString("trim", args, 0)
	if err != nil {
		return interpreter.Null(), err
	}
	return interpreter.StringVal(gostrings.TrimFunc(s, unicode.IsSpace)), nil
}

// trimLeftFn strips leading whitespace only.
func trimLeftFn(_ io.Writer, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityN("trimLeft", args, 1); err != nil {
		return interpreter.Null(), err
	}
	s, err := requireString("trimLeft", args, 0)
	if err != nil {
		return interpreter.Null(), err
	}
	return interpreter.StringVal(gostrings.TrimLeftFunc(s, unicode.IsSpace)), nil
}

// trimRightFn strips trailing whitespace only.
func trimRightFn(_ io.Writer, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityN("trimRight", args, 1); err != nil {
		return interpreter.Null(), err
	}
	s, err := requireString("trimRight", args, 0)
	if err != nil {
		return interpreter.Null(), err
	}
	return interpreter.StringVal(gostrings.TrimRightFunc(s, unicode.IsSpace)), nil
}

// replaceFn returns s with all occurrences of old replaced by new.
func replaceFn(_ io.Writer, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityN("replace", args, 3); err != nil {
		return interpreter.Null(), err
	}
	s, err := requireString("replace", args, 0)
	if err != nil {
		return interpreter.Null(), err
	}
	old, err := requireString("replace", args, 1)
	if err != nil {
		return interpreter.Null(), err
	}
	new, err := requireString("replace", args, 2)
	if err != nil {
		return interpreter.Null(), err
	}
	return interpreter.StringVal(gostrings.ReplaceAll(s, old, new)), nil
}

// repeatFn returns n copies of s concatenated. Negative n is an error.
func repeatFn(_ io.Writer, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityN("repeat", args, 2); err != nil {
		return interpreter.Null(), err
	}
	s, err := requireString("repeat", args, 0)
	if err != nil {
		return interpreter.Null(), err
	}
	n, err := requireInt("repeat", args, 1)
	if err != nil {
		return interpreter.Null(), err
	}
	if n < 0 {
		return interpreter.Null(), fmt.Errorf("repeat(): negative count %d", n)
	}
	// Guard against an absurd allocation (Go's strings.Repeat would panic).
	if n > 0 && len(s) > 0 && int64(len(s))*n/n != int64(len(s)) {
		return interpreter.Null(), fmt.Errorf("repeat(): result would overflow")
	}
	return interpreter.StringVal(gostrings.Repeat(s, int(n))), nil
}

// substringFn returns the slice of s from rune index `start` (inclusive) to
// rune index `end` (exclusive). The end argument is optional - omitting it
// extracts from `start` to the end of the string. Indices are rune-based to
// match `len` and `indexOf`. Out-of-range indices error explicitly (no
// silent clamping).
func substringFn(_ io.Writer, args []interpreter.Value) (interpreter.Value, error) {
	if len(args) != 2 && len(args) != 3 {
		return interpreter.Null(), fmt.Errorf("substring expects 2 or 3 arguments, got %d", len(args))
	}
	s, err := requireString("substring", args, 0)
	if err != nil {
		return interpreter.Null(), err
	}
	start, err := requireInt("substring", args, 1)
	if err != nil {
		return interpreter.Null(), err
	}
	runeLen := int64(utf8.RuneCountInString(s))
	end := runeLen
	if len(args) == 3 {
		end, err = requireInt("substring", args, 2)
		if err != nil {
			return interpreter.Null(), err
		}
	}
	if start < 0 {
		return interpreter.Null(), fmt.Errorf("substring(): start %d is negative", start)
	}
	// Resolve and validate start first so that `substring(s, 99)` (with the
	// 2-arg form, where we auto-fill end) reports the real problem - the
	// start is out of range - rather than confusingly saying "end before
	// start" against the auto-filled end.
	startByte, err := byteOffsetForRune(s, int(start))
	if err != nil {
		return interpreter.Null(), fmt.Errorf("substring(): start out of range (length %d)", runeLen)
	}
	if end < start {
		return interpreter.Null(), fmt.Errorf("substring(): end %d is before start %d", end, start)
	}
	endByte, err := byteOffsetForRune(s, int(end))
	if err != nil {
		return interpreter.Null(), fmt.Errorf("substring(): end out of range (length %d)", runeLen)
	}
	return interpreter.StringVal(s[startByte:endByte]), nil
}
