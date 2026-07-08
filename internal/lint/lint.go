// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

// Package lint reports patterns in Jennifer source that are compile-legal
// but stylistically or semantically suspect. It sits between `jennifer fmt`
// (which normalises lexical shape) and the parser (which rejects the
// outright illegal): the linter's slot is "the code parses, it runs, and
// something about it is still worth flagging."
//
// Each check has a stable ID (L001, L002, ...) so suppression comments and
// project configuration are portable and greppable. The public entry point
// is Check; the CLI (cmd/jennifer/lint.go) wraps it with file I/O and
// output-format rendering.
package lint

import "sort"

// Severity ranks a finding. The CLI's exit code is driven by whether any
// finding lands at or above a severity floor (see SeverityFloor).
type Severity int

const (
	// SeverityInfo is advisory - style thresholds a reader may reasonably
	// disagree with (method length, nesting depth). Below the exit-code
	// floor by default, so an info-only run still exits 0.
	SeverityInfo Severity = iota
	// SeverityWarning is the default for correctness-adjacent findings
	// (dead code, empty catch, non-Error throw, constant conditions).
	SeverityWarning
	// SeverityError is reserved for findings a future check deems
	// unambiguously wrong. No v1 check emits it, but the scale leaves room.
	SeverityError
)

// String renders the severity as it appears in diagnostics and the JSON
// output shape.
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}

// SeverityFloor is the exit-code threshold: a run with findings at or above
// this severity exits 1, otherwise 0. Fixed at Warning for v1 (info-level
// style findings are reported but don't fail a build), mirroring the
// gofmt -l / shellcheck triaging shape.
const SeverityFloor = SeverityWarning

// Diagnostic is one finding. The field set matches the --format=json shape
// documented for the CLI: {id, file, line, col, message, severity}.
type Diagnostic struct {
	ID       string   // stable check ID, e.g. "L002"
	File     string   // source file the finding is in
	Line     int      // 1-based line
	Col      int      // 1-based column (rune-counted, matching the lexer)
	Message  string   // human-readable description
	Severity Severity // drives exit code and rendering
}

// check is the internal registry entry for one lint rule. desc feeds the
// help/catalog; run walks the program and appends diagnostics.
type check struct {
	id       string
	desc     string
	severity Severity
	run      func(c *checkCtx)
}

// registry lists every check in ID order. A check with a nil run is a
// reserved-but-unimplemented family (L008 deprecation), which is a valid
// ID for configuration and suppression but emits nothing on its own.
//
// It is populated in init() rather than as a var initializer: the check
// functions transitively reference severityOf, which reads registry, and Go's
// variable-initialization-cycle analysis rejects that even though nothing runs
// at init. An init() body is exempt from that analysis.
var registry []check

func init() {
	registry = []check{
		{id: "L001", desc: "unused-local: a local binding that is never referenced", severity: SeverityWarning, run: checkUnusedLocal},
		{id: "L002", desc: "dead-code-after-terminator: statements after return/throw/exit/break/continue", severity: SeverityWarning, run: checkDeadCode},
		{id: "L003", desc: "empty-catch: a catch block with no body", severity: SeverityWarning, run: checkEmptyCatch},
		{id: "L004", desc: "throw-non-error: a throw whose value is not statically an Error", severity: SeverityWarning, run: checkThrowNonError},
		{id: "L005", desc: "method-too-long: method body exceeds the statement-count threshold", severity: SeverityInfo, run: checkMethodTooLong},
		{id: "L006", desc: "nesting-too-deep: block nesting exceeds the depth threshold", severity: SeverityInfo, run: checkNestingTooDeep},
		{id: "L007", desc: "constant-condition: a statically constant if/while condition", severity: SeverityWarning, run: checkConstantCondition},
		{id: "L008", desc: "deprecation: use of a still-supported but retiring API (populated as features are deprecated)", severity: SeverityWarning, run: nil},
		{id: "L009", desc: "removed-api: use of an API that has been removed", severity: SeverityWarning, run: checkRemovedApi},
		{id: "L010", desc: "line-too-long: a source line exceeds the column limit", severity: SeverityInfo, run: checkLineTooLong},
	}
}

// KnownIDs returns every registered check ID (including reserved families
// like L008) in ID order. Used by the config and suppression parsers to
// reject unknown IDs, which are always errors.
func KnownIDs() []string {
	ids := make([]string, len(registry))
	for i, c := range registry {
		ids[i] = c.id
	}
	sort.Strings(ids)
	return ids
}

// isKnownID reports whether id names a registered check.
func isKnownID(id string) bool {
	for _, c := range registry {
		if c.id == id {
			return true
		}
	}
	return false
}

// Catalog returns each check's ID, description, and default severity for
// help output. Order is stable (registry order).
func Catalog() []struct {
	ID       string
	Desc     string
	Severity Severity
} {
	out := make([]struct {
		ID       string
		Desc     string
		Severity Severity
	}, len(registry))
	for i, c := range registry {
		out[i].ID = c.id
		out[i].Desc = c.desc
		out[i].Severity = c.severity
	}
	return out
}
