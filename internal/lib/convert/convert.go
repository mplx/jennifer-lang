// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

// Package convert implements Jennifer's `convert` library: explicit value
// conversions between the primitive kinds, plus `typeOf` for runtime kind
// introspection.
package convert

import (
	"fmt"
	"strconv"

	"github.com/mplx/jennifer-lang/internal/interpreter"
)

// LibraryName is the Jennifer name programs `use` to enable these functions.
const LibraryName = "convert"

// Install registers convert library functions on an interpreter. Every
// name is namespaced behind `convert.` (M10+). The four conversion
// callees are named `toInt`, `toFloat`, `toString`, `toBool` so they
// don't collide with the type keywords (`int`, `float`, ...); the
// `to`-prefixed verb also reads as English at the call site
// (`convert.toInt("42")`). `typeOf` stays as-is - it doesn't have a
// keyword collision and the name carries its own intent.
func Install(in *interpreter.Interpreter) {
	in.RegisterNamespaced(LibraryName, "toInt", toIntFn)
	in.RegisterNamespaced(LibraryName, "toFloat", toFloatFn)
	in.RegisterNamespaced(LibraryName, "toString", toStringFn)
	in.RegisterNamespaced(LibraryName, "toBool", toBoolFn)
	in.RegisterNamespaced(LibraryName, "typeOf", typeOfFn)
}

// arityOne returns an error if args doesn't contain exactly one value.
func arityOne(name string, args []interpreter.Value) error {
	if len(args) != 1 {
		return fmt.Errorf("%s expects 1 argument, got %d", name, len(args))
	}
	return nil
}

// toIntFn implements `convert.toInt(v)`:
//   - int    -> identity
//   - float  -> truncate toward zero (Go's int64 cast)
//   - string -> strconv.ParseInt(base 10, 64-bit); error on bad input
//   - bool   -> true=1, false=0
//   - null   -> error
func toIntFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityOne("toInt", args); err != nil {
		return interpreter.Null(), err
	}
	v := args[0]
	switch v.Kind {
	case interpreter.KindInt:
		return v, nil
	case interpreter.KindFloat:
		return interpreter.IntVal(int64(v.Float)), nil
	case interpreter.KindString:
		n, err := strconv.ParseInt(v.Str, 10, 64)
		if err != nil {
			return interpreter.Null(), fmt.Errorf("toInt(%q): not a valid integer", v.Str)
		}
		return interpreter.IntVal(n), nil
	case interpreter.KindBool:
		if v.Bool {
			return interpreter.IntVal(1), nil
		}
		return interpreter.IntVal(0), nil
	}
	return interpreter.Null(), fmt.Errorf("toInt(): cannot convert %s to int", v.Kind)
}

// toFloatFn implements `convert.toFloat(v)`:
//   - int    -> convert
//   - float  -> identity
//   - string -> strconv.ParseFloat(64-bit); error on bad input
//   - bool   -> true=1.0, false=0.0
//   - null   -> error
func toFloatFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityOne("toFloat", args); err != nil {
		return interpreter.Null(), err
	}
	v := args[0]
	switch v.Kind {
	case interpreter.KindInt:
		return interpreter.FloatVal(float64(v.Int)), nil
	case interpreter.KindFloat:
		return v, nil
	case interpreter.KindString:
		f, err := strconv.ParseFloat(v.Str, 64)
		if err != nil {
			return interpreter.Null(), fmt.Errorf("toFloat(%q): not a valid float", v.Str)
		}
		return interpreter.FloatVal(f), nil
	case interpreter.KindBool:
		if v.Bool {
			return interpreter.FloatVal(1.0), nil
		}
		return interpreter.FloatVal(0.0), nil
	}
	return interpreter.Null(), fmt.Errorf("toFloat(): cannot convert %s to float", v.Kind)
}

// toStringFn implements `convert.toString(v)`: returns the value's
// display form. Never fails (every kind has a defined Display).
func toStringFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityOne("toString", args); err != nil {
		return interpreter.Null(), err
	}
	return interpreter.StringVal(args[0].Display()), nil
}

// toBoolFn implements `convert.toBool(v)` with strict canonical-only conversions:
//
//   - bool   -> identity
//   - int    -> 0 = false, 1 = true; any other int errors
//   - float  -> 0.0 = false, 1.0 = true; any other float errors
//   - string -> "true" = true, "false" = false; any other string errors
//   - null   -> always errors
//
// If you want "nonzero counts as true" semantics, write the comparison
// explicitly: `def b as bool init $x != 0;`.
func toBoolFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityOne("toBool", args); err != nil {
		return interpreter.Null(), err
	}
	v := args[0]
	switch v.Kind {
	case interpreter.KindBool:
		return v, nil
	case interpreter.KindInt:
		switch v.Int {
		case 0:
			return interpreter.BoolVal(false), nil
		case 1:
			return interpreter.BoolVal(true), nil
		}
		return interpreter.Null(), fmt.Errorf("toBool(%d): only 0 and 1 are accepted (use `$x != 0` for truthiness)", v.Int)
	case interpreter.KindFloat:
		switch v.Float {
		case 0:
			return interpreter.BoolVal(false), nil
		case 1:
			return interpreter.BoolVal(true), nil
		}
		return interpreter.Null(), fmt.Errorf("toBool(%s): only 0.0 and 1.0 are accepted (use `$x != 0.0` for truthiness)", interpreter.DisplayFloat(v.Float))
	case interpreter.KindString:
		switch v.Str {
		case "true":
			return interpreter.BoolVal(true), nil
		case "false":
			return interpreter.BoolVal(false), nil
		}
		return interpreter.Null(), fmt.Errorf("toBool(%q): only \"true\" or \"false\" are accepted", v.Str)
	}
	return interpreter.Null(), fmt.Errorf("toBool(): cannot convert %s to bool", v.Kind)
}

// typeOfFn returns the runtime kind name of its argument as a string:
// "null", "int", "float", "string", or "bool". Useful for debugging and
// runtime introspection.
func typeOfFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityOne("typeOf", args); err != nil {
		return interpreter.Null(), err
	}
	return interpreter.StringVal(args[0].Kind.String()), nil
}
