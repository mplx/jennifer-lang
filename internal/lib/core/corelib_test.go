// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package corelib_test

import (
	"strings"
	"testing"

	"github.com/mplx/jennifer-lang/internal/interpreter"
	corelib "github.com/mplx/jennifer-lang/internal/lib/core"
	iolib "github.com/mplx/jennifer-lang/internal/lib/io"
	"github.com/mplx/jennifer-lang/internal/parser"
)

// TestExplicitUseCoreIsRejected confirms `use core;` errors instead of
// silently passing - core is auto-loaded and an explicit import signals
// confusion that's better surfaced loudly.
func TestExplicitUseCoreIsRejected(t *testing.T) {
	in := interpreter.New()
	iolib.Install(in)
	corelib.Install(in)

	src := `use core;`
	prog, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	err = in.Run(prog)
	if err == nil {
		t.Fatal("expected error rejecting `use core;`, got nil")
	}
	if !strings.Contains(err.Error(), "automatically") && !strings.Contains(err.Error(), "auto-loaded") {
		t.Errorf("expected error to explain core is auto-loaded, got %v", err)
	}
}
