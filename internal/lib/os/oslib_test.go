// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package oslib

import (
	stdos "os"
	"runtime"
	"testing"

	"github.com/mplx/jennifer-lang/internal/interpreter"
)

func TestPlatformReturnsRuntimeGOOS(t *testing.T) {
	v, err := platformFn(interpreter.BuiltinCtx{}, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if v.Kind != interpreter.KindString || v.Str != runtime.GOOS {
		t.Errorf("got (%s, %q), want (string, %q)", v.Kind, v.Str, runtime.GOOS)
	}
}

func TestPlatformRejectsArgs(t *testing.T) {
	_, err := platformFn(interpreter.BuiltinCtx{}, []interpreter.Value{interpreter.IntVal(1)})
	if err == nil {
		t.Fatal("expected arity error, got nil")
	}
}

func TestGetEnvReadsSetVariable(t *testing.T) {
	const key = "JENNIFER_TEST_M8_VAR"
	stdos.Setenv(key, "hello")
	defer stdos.Unsetenv(key)
	v, err := getEnvFn(interpreter.BuiltinCtx{}, []interpreter.Value{interpreter.StringVal(key)})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if v.Kind != interpreter.KindString || v.Str != "hello" {
		t.Errorf("got (%s, %q)", v.Kind, v.Str)
	}
}

func TestGetEnvReturnsEmptyWhenUnset(t *testing.T) {
	const key = "JENNIFER_TEST_M8_UNSET_VAR"
	stdos.Unsetenv(key)
	v, err := getEnvFn(interpreter.BuiltinCtx{}, []interpreter.Value{interpreter.StringVal(key)})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if v.Str != "" {
		t.Errorf("expected empty string, got %q", v.Str)
	}
}

func TestGetEnvRejectsNonString(t *testing.T) {
	_, err := getEnvFn(interpreter.BuiltinCtx{}, []interpreter.Value{interpreter.IntVal(1)})
	if err == nil {
		t.Fatal("expected type error, got nil")
	}
}

func TestPlatformLineEndingLinuxToday(t *testing.T) {
	// Jennifer ships Linux-only today. When cross-platform support
	// lands the matching update will make this test branch.
	if got := platformLineEnding(); got != "\n" {
		t.Errorf("got %q, want \\n on linux", got)
	}
}
