// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package mathlib

import (
	"math"
	"testing"

	"jennifer-lang.dev/jennifer/internal/interpreter"
)

// randInt over ranges whose width exceeds 2^63 must not panic (Int63n
// rejects a non-positive span) and must return a value inside [lo, hi].
// Covers both wide-range branches: uspan == 0 (the full int64 range) and
// uspan != 0 (a wide but not-full range).
func TestRandIntWideRanges(t *testing.T) {
	cases := []struct {
		name   string
		lo, hi int64
	}{
		{"full-int64-range", math.MinInt64, math.MaxInt64},
		{"zero-to-max", 0, math.MaxInt64},
		{"min-to-zero", math.MinInt64, 0},
		{"narrow", -5, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			args := []interpreter.Value{interpreter.IntVal(c.lo), interpreter.IntVal(c.hi)}
			for i := 0; i < 1000; i++ {
				v, err := randIntFn(interpreter.BuiltinCtx{}, args)
				if err != nil {
					t.Fatalf("randInt(%d, %d): unexpected error: %v", c.lo, c.hi, err)
				}
				if v.Kind != interpreter.KindInt {
					t.Fatalf("randInt returned %s, want int", v.Kind)
				}
				if v.Int < c.lo || v.Int > c.hi {
					t.Fatalf("randInt(%d, %d) = %d, out of range", c.lo, c.hi, v.Int)
				}
			}
		})
	}
}

// floor / ceil / round must reject values int64 cannot hold (NaN, +/-Inf,
// out of range) rather than returning platform-defined garbage from a bare
// int64(f) cast - math's strict stance.
func TestFloorCeilRoundRejectUnrepresentable(t *testing.T) {
	bad := []interpreter.Value{
		interpreter.FloatVal(1e300),
		interpreter.FloatVal(-1e300),
		interpreter.FloatVal(math.Inf(1)),
		interpreter.FloatVal(math.Inf(-1)),
		interpreter.FloatVal(math.NaN()),
		interpreter.FloatVal(9223372036854775808.0), // exactly 2^63
	}
	for _, fn := range []struct {
		name string
		f    func(interpreter.BuiltinCtx, []interpreter.Value) (interpreter.Value, error)
	}{{"floor", floorFn}, {"ceil", ceilFn}, {"round", roundFn}} {
		for _, v := range bad {
			if _, err := fn.f(interpreter.BuiltinCtx{}, []interpreter.Value{v}); err == nil {
				t.Errorf("%s(%g) should error, got nil", fn.name, v.Float)
			}
		}
	}
	// A representable value still works.
	if r, err := floorFn(interpreter.BuiltinCtx{}, []interpreter.Value{interpreter.FloatVal(3.7)}); err != nil || r.Int != 3 {
		t.Errorf("floor(3.7) = %+v, err %v; want 3", r, err)
	}
}

// abs(MinInt64) has no representable result (|-2^63| = 2^63 > MaxInt64), so it
// must error rather than return the wrong (still-negative) value.
func TestAbsMinInt64Errors(t *testing.T) {
	if _, err := absFn(interpreter.BuiltinCtx{}, []interpreter.Value{interpreter.IntVal(math.MinInt64)}); err == nil {
		t.Error("abs(MinInt64) should error, got nil")
	}
	// A normal negative still flips.
	if r, err := absFn(interpreter.BuiltinCtx{}, []interpreter.Value{interpreter.IntVal(-5)}); err != nil || r.Int != 5 {
		t.Errorf("abs(-5) = %+v, err %v; want 5", r, err)
	}
}
