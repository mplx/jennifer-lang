// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package convert

import (
	"math"
	"testing"

	"jennifer-lang.dev/jennifer/internal/interpreter"
)

// convert.toInt truncates a float toward zero but must reject the values int64
// cannot hold (NaN, +/-Inf, out of range) rather than returning garbage from a
// bare int64(f) cast - convert is canonical-only.
func TestToIntRejectsUnrepresentableFloats(t *testing.T) {
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 1e300, -1e300, 9223372036854775808.0} {
		if _, err := toIntFn(interpreter.BuiltinCtx{}, []interpreter.Value{interpreter.FloatVal(v)}); err == nil {
			t.Errorf("toInt(%g) should error, got nil", v)
		}
	}
	// In-range truncation still works.
	if r, err := toIntFn(interpreter.BuiltinCtx{}, []interpreter.Value{interpreter.FloatVal(3.9)}); err != nil || r.Int != 3 {
		t.Errorf("toInt(3.9) = %+v, err %v; want 3", r, err)
	}
}
