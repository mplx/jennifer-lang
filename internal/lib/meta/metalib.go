// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

// Package metalib implements Jennifer's `meta` library: a home for
// constants that describe the running Jennifer interpreter itself
// rather than the host environment (which lives in `os`) or any user
// data (which is the rest of the standard library).
//
// Today the library is small - two constants. The library exists
// because `meta.VERSION` and `meta.BUILD` are conceptually the same
// kind of fact ("identity of this interpreter binary") and historically
// `JENNIFER_VERSION` lived in `core` for lack of a better home. With
// `meta` shipping, `core` returns to its strict charter of polymorphic
// structural primitives. The 5+-name rule that normally gates new
// libraries is bent here because (a) interpreter-identity constants
// don't fit any existing library cleanly and (b) future runtime
// introspection (build time, git SHA, GC stats, scheduler info) has a
// natural home here once it lands.
//
// All names are namespaced under the `meta.` prefix. Programs enable
// the library with `use meta;` like every other non-core library.
package metalib

import (
	"runtime"

	"github.com/mplx/jennifer-lang/internal/interpreter"
	"github.com/mplx/jennifer-lang/internal/version"
)

// LibraryName is the Jennifer name programs `use` to enable these
// constants, and doubles as the namespace prefix.
const LibraryName = "meta"

// Install registers the meta library's constants.
func Install(in *interpreter.Interpreter) {
	in.RegisterNamespacedConst(LibraryName, "VERSION", interpreter.StringVal(version.Version))
	in.RegisterNamespacedConst(LibraryName, "BUILD", interpreter.StringVal(buildTag()))
}

// buildTag distinguishes which Go variant compiled the interpreter.
// TinyGo sets `runtime.Compiler` to `"tinygo"`; the standard `go`
// toolchain sets it to `"gc"`, which we normalise to `"go"` because
// it's the user-facing toolchain name even if `gc` is the internal
// compiler identifier. Any other value is passed through so future
// alternative compilers (gccgo, etc.) appear honestly.
func buildTag() string {
	switch runtime.Compiler {
	case "tinygo":
		return "tinygo"
	case "gc":
		return "go"
	default:
		return runtime.Compiler
	}
}
