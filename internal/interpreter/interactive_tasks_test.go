// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package interpreter_test

import (
	"os"
	"strings"
	"testing"

	"jennifer-lang.dev/jennifer/internal/interpreter"
	iolib "jennifer-lang.dev/jennifer/internal/lib/io"
	mathlib "jennifer-lang.dev/jennifer/internal/lib/math"
	tasklib "jennifer-lang.dev/jennifer/internal/lib/task"
	timelib "jennifer-lang.dev/jennifer/internal/lib/time"
)

// Spawned tasks resolve method calls, struct lookups, and namespace
// prefixes by name from their own goroutines. The REPL's next input may
// define new methods / structs or add imports, which writes those shared
// tables - a data race that can kill the process ("concurrent map read and
// map write"). EvalInteractive must reject table-mutating input while any
// task is still running, and accept it again once the tasks are observed.
func TestReplRejectsTableWritesWhileTasksRun(t *testing.T) {
	in := interpreter.New()
	iolib.Install(in)
	mathlib.Install(in)
	timelib.Install(in)
	tasklib.Install(in)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Input 1: activate libs and park a task on a sleep.
	if _, err := evalRepl(t, in, cwd,
		`use time; use task; def tk as task of int init spawn { time.sleep(time.fromMilliseconds(300)); return 1; };`); err != nil {
		t.Fatalf("input 1: %v", err)
	}
	// Method definitions are rejected while the task runs.
	if _, err := evalRepl(t, in, cwd, `func foo() { return 2; }`); err == nil || !strings.Contains(err.Error(), "task") {
		t.Fatalf("method def: want live-task rejection, got %v", err)
	}
	// Struct definitions and `use` imports mutate shared tables too.
	if _, err := evalRepl(t, in, cwd, `def struct P { x as int };`); err == nil {
		t.Fatal("struct def: want live-task rejection, got nil")
	}
	if _, err := evalRepl(t, in, cwd, `use math;`); err == nil {
		t.Fatal("use import: want live-task rejection, got nil")
	}
	// Plain statements still evaluate (spawn snapshots are isolated from
	// the live global frame, so those writes race nothing).
	if _, err := evalRepl(t, in, cwd, `def x as int init 41;`); err != nil {
		t.Fatalf("plain statement: %v", err)
	}
	// Observe the task; afterwards definitions work again.
	if _, err := evalRepl(t, in, cwd, `def r as int init task.wait($tk);`); err != nil {
		t.Fatalf("task.wait: %v", err)
	}
	if _, err := evalRepl(t, in, cwd, `func foo() { return 2; }`); err != nil {
		t.Fatalf("redefine after wait: %v", err)
	}
	if _, err := evalRepl(t, in, cwd, `use math;`); err != nil {
		t.Fatalf("use after wait: %v", err)
	}
}
