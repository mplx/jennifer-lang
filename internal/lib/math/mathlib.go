// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

// Package mathlib implements Jennifer's `math` library: a small set of
// frequently-needed numeric functions plus the constants PI and E. Strict
// error policy: anything mathematically undefined (sqrt of negative, NaN/Inf
// results) is rejected at runtime rather than letting NaN propagate.
//
// The Go package is named mathlib to avoid colliding with Go's standard
// `math` package, which this implementation depends on.
package mathlib

import (
	"fmt"
	"math"

	"github.com/mplx/jennifer-lang/internal/interpreter"
)

// LibraryName is the Jennifer name programs `use` to enable these functions.
const LibraryName = "math"

// Install registers math library functions and constants on an interpreter.
func Install(in *interpreter.Interpreter) {
	in.Register(LibraryName, "abs", absFn)
	in.Register(LibraryName, "min", minFn)
	in.Register(LibraryName, "max", maxFn)
	in.Register(LibraryName, "sqrt", sqrtFn)
	in.Register(LibraryName, "pow", powFn)
	in.Register(LibraryName, "floor", floorFn)
	in.Register(LibraryName, "ceil", ceilFn)
	in.Register(LibraryName, "round", roundFn)

	in.RegisterConst(LibraryName, "PI", interpreter.FloatVal(math.Pi))
	in.RegisterConst(LibraryName, "E", interpreter.FloatVal(math.E))
}

func arityOne(name string, args []interpreter.Value) error {
	if len(args) != 1 {
		return fmt.Errorf("%s expects 1 argument, got %d", name, len(args))
	}
	return nil
}

func arityTwo(name string, args []interpreter.Value) error {
	if len(args) != 2 {
		return fmt.Errorf("%s expects 2 arguments, got %d", name, len(args))
	}
	return nil
}

// absFn returns the absolute value, preserving the operand's type.
//   - int    -> int
//   - float  -> float
func absFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityOne("abs", args); err != nil {
		return interpreter.Null(), err
	}
	v := args[0]
	switch v.Kind {
	case interpreter.KindInt:
		if v.Int < 0 {
			return interpreter.IntVal(-v.Int), nil
		}
		return v, nil
	case interpreter.KindFloat:
		return interpreter.FloatVal(math.Abs(v.Float)), nil
	}
	return interpreter.Null(), fmt.Errorf("abs(): requires int or float, got %s", v.Kind)
}

// minFn returns the lesser of two numeric values. (int,int) -> int;
// mixed or (float,float) -> float (matches the `+` promotion rule).
func minFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityTwo("min", args); err != nil {
		return interpreter.Null(), err
	}
	return cmpPick(args[0], args[1], true, "min")
}

// maxFn is symmetric with minFn.
func maxFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityTwo("max", args); err != nil {
		return interpreter.Null(), err
	}
	return cmpPick(args[0], args[1], false, "max")
}

// cmpPick selects a or b based on `wantSmaller`. Same numeric-promotion rule
// as the `+` operator.
func cmpPick(a, b interpreter.Value, wantSmaller bool, name string) (interpreter.Value, error) {
	// Same-int fast path keeps the int type.
	if a.Kind == interpreter.KindInt && b.Kind == interpreter.KindInt {
		if wantSmaller {
			if a.Int <= b.Int {
				return a, nil
			}
			return b, nil
		}
		if a.Int >= b.Int {
			return a, nil
		}
		return b, nil
	}
	af, aok := a.AsFloat()
	bf, bok := b.AsFloat()
	if !aok || !bok {
		return interpreter.Null(), fmt.Errorf("%s(): requires numeric operands, got %s and %s", name, a.Kind, b.Kind)
	}
	if wantSmaller {
		if af <= bf {
			return interpreter.FloatVal(af), nil
		}
		return interpreter.FloatVal(bf), nil
	}
	if af >= bf {
		return interpreter.FloatVal(af), nil
	}
	return interpreter.FloatVal(bf), nil
}

// sqrtFn returns the square root as a float. Errors on negative input.
func sqrtFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityOne("sqrt", args); err != nil {
		return interpreter.Null(), err
	}
	v := args[0]
	f, ok := v.AsFloat()
	if !ok {
		return interpreter.Null(), fmt.Errorf("sqrt(): requires int or float, got %s", v.Kind)
	}
	if f < 0 {
		return interpreter.Null(), fmt.Errorf("sqrt(): undefined for negative input %s", interpreter.DisplayFloat(f))
	}
	return interpreter.FloatVal(math.Sqrt(f)), nil
}

// powFn returns x**y as a float (always - avoids "is this int or float"
// ambiguity). NaN or Infinity results are rejected.
func powFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityTwo("pow", args); err != nil {
		return interpreter.Null(), err
	}
	a, aok := args[0].AsFloat()
	b, bok := args[1].AsFloat()
	if !aok || !bok {
		return interpreter.Null(), fmt.Errorf("pow(): requires numeric operands, got %s and %s", args[0].Kind, args[1].Kind)
	}
	r := math.Pow(a, b)
	if math.IsNaN(r) || math.IsInf(r, 0) {
		return interpreter.Null(), fmt.Errorf("pow(%s, %s): result is undefined or infinite", interpreter.DisplayFloat(a), interpreter.DisplayFloat(b))
	}
	return interpreter.FloatVal(r), nil
}

// floorFn rounds toward negative infinity. Accepts int (identity) or float
// (returns int).
func floorFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityOne("floor", args); err != nil {
		return interpreter.Null(), err
	}
	v := args[0]
	if v.Kind == interpreter.KindInt {
		return v, nil
	}
	if v.Kind != interpreter.KindFloat {
		return interpreter.Null(), fmt.Errorf("floor(): requires int or float, got %s", v.Kind)
	}
	return interpreter.IntVal(int64(math.Floor(v.Float))), nil
}

// ceilFn rounds toward positive infinity. Accepts int (identity) or float
// (returns int).
func ceilFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityOne("ceil", args); err != nil {
		return interpreter.Null(), err
	}
	v := args[0]
	if v.Kind == interpreter.KindInt {
		return v, nil
	}
	if v.Kind != interpreter.KindFloat {
		return interpreter.Null(), fmt.Errorf("ceil(): requires int or float, got %s", v.Kind)
	}
	return interpreter.IntVal(int64(math.Ceil(v.Float))), nil
}

// roundFn rounds half-away-from-zero (Go's math.Round). Accepts int
// (identity) or float (returns int).
func roundFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if err := arityOne("round", args); err != nil {
		return interpreter.Null(), err
	}
	v := args[0]
	if v.Kind == interpreter.KindInt {
		return v, nil
	}
	if v.Kind != interpreter.KindFloat {
		return interpreter.Null(), fmt.Errorf("round(): requires int or float, got %s", v.Kind)
	}
	return interpreter.IntVal(int64(math.Round(v.Float))), nil
}
