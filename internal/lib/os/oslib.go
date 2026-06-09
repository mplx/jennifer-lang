// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

// Package oslib implements Jennifer's `os` library: a minimal slice of
// operating-system glue shipped in M8 to exercise the new namespaced-call
// machinery end-to-end. The full library expands in M13.1 (`os.args`,
// `os.exit(n)`, the rest of the `JENNIFER_*` constants).
//
// Everything here is namespaced - the user writes `os.platform()`,
// `os.getEnv("HOME")`, `os.JENNIFER_LF`, `os.JENNIFER_OS`. None of these
// names are reachable bare.
//
// The Go package is named oslib to avoid colliding with Go's standard
// `os` package, which this implementation depends on.
package oslib

import (
	"fmt"
	stdos "os"
	"runtime"

	"github.com/mplx/jennifer-lang/internal/interpreter"
)

// LibraryName is the Jennifer name programs `use` to enable these functions.
// It doubles as the namespace prefix: users write `os.platform()`.
const LibraryName = "os"

// Install registers the M8 minimal slice of the `os` library.
func Install(in *interpreter.Interpreter) {
	in.RegisterNamespaced(LibraryName, "platform", platformFn)
	in.RegisterNamespaced(LibraryName, "getEnv", getEnvFn)

	in.RegisterNamespacedConst(LibraryName, "JENNIFER_LF", interpreter.StringVal(platformLineEnding()))
	in.RegisterNamespacedConst(LibraryName, "JENNIFER_OS", interpreter.StringVal(runtime.GOOS))
}

// platformFn returns the operating-system name as reported by the Go runtime
// ("linux", "darwin", "windows", ...). Zero-arg.
func platformFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if len(args) != 0 {
		return interpreter.Null(), fmt.Errorf("os.platform expects 0 arguments, got %d", len(args))
	}
	return interpreter.StringVal(runtime.GOOS), nil
}

// getEnvFn reads an environment variable. Unset variables produce the
// empty string (no error) so callers can use `==`/`!=` against `""` to
// branch. Mirrors Go's `os.Getenv`.
func getEnvFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if len(args) != 1 {
		return interpreter.Null(), fmt.Errorf("os.getEnv expects 1 argument, got %d", len(args))
	}
	if args[0].Kind != interpreter.KindString {
		return interpreter.Null(), fmt.Errorf("os.getEnv: variable name must be string, got %s", args[0].Kind)
	}
	return interpreter.StringVal(stdos.Getenv(args[0].Str)), nil
}

// platformLineEnding returns the conventional line terminator for the host.
// Today Jennifer is Linux-only so this always resolves to "\n"; the switch
// is in place so Windows support (when M11 / cross-compile lands) just
// changes one return value.
func platformLineEnding() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}
