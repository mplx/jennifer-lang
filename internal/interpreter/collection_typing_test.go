// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package interpreter_test

import "testing"

// A generic collection (a fresh literal, or a json.decode result - neither
// records an element type) must be validated entry by entry against the
// declared element type, at every binding boundary, so a mismatched or
// heterogeneous collection can't be relabeled as matching. Literals stand in
// for decode results here (both are generic).

func TestGenericCollectionRejectedAtDefInit(t *testing.T) {
	cases := []string{
		`def x as map of string to string init {"a": 1};`,
		`def x as map of string to int init {"a": "s"};`,
		`def x as list of string init [1, 2];`,
		`def x as list of int init [1, "s"];`,                    // heterogeneous
		`def x as map of string to int init {"a": 1, "b": "s"};`, // heterogeneous
	}
	for _, src := range cases {
		if _, err := run(t, src); err == nil {
			t.Errorf("expected a type error for %q", src)
		}
	}
}

func TestGenericCollectionRejectedAtAssign(t *testing.T) {
	cases := []string{
		`def x as map of string to string; $x = {"a": 1};`,
		`def x as list of int; $x = ["s"];`,
		`def x as map of string to int init {"a": 1}; $x = {"b": "s"};`,
	}
	for _, src := range cases {
		if _, err := run(t, src); err == nil {
			t.Errorf("expected a type error for %q", src)
		}
	}
}

func TestHomogeneousCollectionBinds(t *testing.T) {
	cases := []string{
		`use io; def x as map of string to int init {"a": 1, "b": 2}; io.printf("%d", $x["a"]);`,
		`use io; def x as list of string init ["a", "b"]; io.printf("%s", $x[0]);`,
		`use io; def x as list of list of int init [[1], [2, 3]]; io.printf("%d", $x[1][1]);`,
		// empty collections stay assignable to any declared collection type
		`use io; def x as map of string to int init {}; def y as list of string init []; io.printf("ok");`,
		`use io; def x as list of int; $x = [1, 2, 3]; io.printf("%d", $x[2]);`,
	}
	for _, src := range cases {
		if _, err := run(t, src); err != nil {
			t.Errorf("expected a clean bind for %q, got %v", src, err)
		}
	}
}
