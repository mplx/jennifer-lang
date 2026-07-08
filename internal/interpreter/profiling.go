// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package interpreter

import (
	"time"

	"github.com/mplx/jennifer-lang/internal/parser"
)

// Profiler receives instrumentation events from the evaluator when profiling
// is active. It is an interface so the interpreter (which compiles into both
// binaries) carries no profiling machinery of its own - the concrete
// collector lives in internal/profile and is injected only by the default
// binary's `jennifer profile` subcommand. All time values are wall-clock.
type Profiler interface {
	// Start marks the profile's zero time; call spans are relative to it.
	Start(t time.Time)
	// RecordStmt reports one statement execution's self time (excluding
	// nested statements) and cumulative time (including them).
	RecordStmt(file string, line, col int, self, cum time.Duration)
	// RecordCall reports one method-call span for the trace timeline.
	RecordCall(name, file string, line, col int, start, end time.Time)
	// RecordDetach counts one COW detachment (an Ensure that copied a shared
	// backing) at the mutation site.
	RecordDetach(file string, line, col int)
	// RecordEagerCopy counts one eager deep copy at a value-storage site
	// (def / assignment / parameter binding), where value semantics turn a
	// store into a real copy up front rather than deferring to Ensure.
	RecordEagerCopy(file string, line, col int)
	// RecordSpawnCopy counts one spawn-frame deep copy and its cost at the
	// spawn site.
	RecordSpawnCopy(file string, line, col int, dur time.Duration)
}

// SetProfiler installs a profiler and selects which streams to record.
// timeStmts drives the statement profile (default mode); timeCalls records the
// method-call timeline (the trace form); trackAllocs records COW detachments
// and spawn-frame copies (--allocs mode). Passing nil disables profiling.
func (i *Interpreter) SetProfiler(p Profiler, timeStmts, timeCalls, trackAllocs bool) {
	i.prof = p
	i.profStmts = timeStmts
	i.profCalls = timeCalls
	i.profAllocs = trackAllocs
}

// ensureCOW is the profiling-aware form of Value.Ensure used at mutation
// sites. It behaves exactly like Ensure (detach a shared backing, else return
// the value unchanged) but, when allocation profiling is on and a real
// detachment happens, attributes the copy to node n's source position. Because
// Value and Interpreter share this package, the interpreter can read the
// shared marker directly rather than needing Ensure to signal a copy.
func (i *Interpreter) ensureCOW(v Value, n parser.Node) Value {
	if v.shared != nil && *v.shared {
		if i.prof != nil && i.profAllocs {
			file, line, col := posFor(n)
			i.prof.RecordDetach(file, line, col)
		}
		return v.DeepCopy()
	}
	return v
}

// isCompoundCopyKind reports whether a value of kind k gets a real deep copy
// from Copy() (bytes/list/map/struct). Scalars copy trivially and tasks share
// a pointer, so neither is a meaningful "eager copy" to count.
func isCompoundCopyKind(k ValueKind) bool { return k >= KindBytes && k <= KindStruct }

// eagerCopy is the profiling-aware form of Value.Copy() used at value-storage
// sites (def / assignment). It behaves exactly like Copy() but, when
// allocation profiling is on and the value is a compound (so the copy is real
// work), attributes the copy to node n's position.
func (i *Interpreter) eagerCopy(v Value, n parser.Node) Value {
	if i.prof != nil && i.profAllocs && isCompoundCopyKind(v.Kind) {
		file, line, col := posFor(n)
		i.prof.RecordEagerCopy(file, line, col)
	}
	return v.Copy()
}
