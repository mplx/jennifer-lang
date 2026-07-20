// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

//go:build tinygo

package limits

// MaxNestingDepth caps structural nesting for jennifer-tiny, whose goroutine
// stack is a fixed 2 MB (see the Makefile's -stack-size and the TinyGo notes in
// docs/technical/tinygo.md). The tree-walker overflows that stack on the
// heaviest shape - a deeply-nested map literal - between depth 96 (survives) and
// 128 (segfaults); a nested list crashes near 160. 64 sits comfortably below the
// tightest floor with margin for shape variance, and is still an order of
// magnitude past the point where nesting is a code smell.
const MaxNestingDepth = 64
