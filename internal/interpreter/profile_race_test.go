// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package interpreter_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mplx/jennifer-lang/internal/interpreter"
	iolib "github.com/mplx/jennifer-lang/internal/lib/io"
	listslib "github.com/mplx/jennifer-lang/internal/lib/lists"
	tasklib "github.com/mplx/jennifer-lang/internal/lib/task"
	"github.com/mplx/jennifer-lang/internal/parser"
	"github.com/mplx/jennifer-lang/internal/profile"
)

// Profiling a program that fans out to parallel `spawn` workers must not race.
// Two shared structures are involved: the statement timer's self/cumulative
// accumulator (lives on each goroutine's root env, not the shared Interpreter)
// and the Collector's maps (mutex-guarded). Before the fix this crashed with
// "concurrent map read and map write"; run under `go test -race` it also flags
// the accumulator race if either guard regresses.
func TestProfileConcurrentSpawnNoRace(t *testing.T) {
	src := `
use task;
func work(n as int) {
    def total as int init 0;
    def i as int init 0;
    while ($i < $n) {
        $total = $total + $i;
        $i = $i + 1;
    }
    return $total;
}
def tasks as list of task of int init [];
def w as int init 0;
while ($w < 8) {
    $tasks[] = spawn { return work(2000); };
    $w = $w + 1;
}
def results as list of int init task.waitAll($tasks);
`
	prog, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	in := interpreter.New()
	var buf bytes.Buffer
	in.Out = &buf
	iolib.Install(in)
	listslib.Install(in)
	tasklib.Install(in)
	col := profile.NewCollector(profile.ModeStatement, 0)
	in.SetProfiler(col, true, false, false)
	if err := in.Run(prog); err != nil {
		t.Fatalf("run: %v", err)
	}
	// Rendering after the parallel run must not panic, and the spawn-body
	// statements must have been recorded (they run on their own goroutines).
	var out bytes.Buffer
	col.Table(&out)
	if !strings.Contains(out.String(), "statement executions") {
		t.Errorf("profile table missing execution summary:\n%s", out.String())
	}
}
