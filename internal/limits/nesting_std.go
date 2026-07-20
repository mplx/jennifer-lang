// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

//go:build !tinygo

// Package limits holds the shared resource caps that every recursive-descent
// parser in the tree (the language parser and the json / toml / xml decoders)
// enforces so that deeply-nested untrusted input cannot exhaust the Go stack.
//
// The value is build-tag split because the two binaries have very different
// stacks. The interpreter has no recover(): a Go stack overflow is a fatal,
// uncatchable crash, so the cap must sit below the depth at which the
// tree-walker's per-level frames overflow the stack, not merely "beyond any
// real document".
package limits

// MaxNestingDepth caps structural nesting (containers, grouped expressions) in
// every recursive-descent parser. The default binary uses Go's growable stack
// (up to ~1 GB), so 1000 is far below any crash point and leaves ample room for
// even a pathologically deep serialized document.
const MaxNestingDepth = 1000
