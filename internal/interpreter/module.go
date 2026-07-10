// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package interpreter

import (
	"path/filepath"
	"strings"

	"github.com/mplx/jennifer-lang/internal/module"
	"github.com/mplx/jennifer-lang/internal/parser"
)

// moduleReg is the module registry shared across one program run: the
// run-once cache (canonical path -> loaded module), the in-progress load
// stack for cycle detection, the module search path, and the callbacks the
// loader needs to turn a resolved path into a runnable module. Every module
// loads into a fresh sub-interpreter that shares this registry, so
// run-once, depth-first post-order init, and cycle detection all fall out
// of the recursion.
type moduleReg struct {
	cache  map[string]*loadedModule
	stack  []string                                            // canonical paths currently loading
	search []string                                            // module search dirs (sysmoddir, then -I dirs)
	load   func(canonicalPath string) (*parser.Program, error) // lex/preproc/parse a module file
	setup  func(*Interpreter)                                  // install the standard library into a module interpreter
}

// loadedModule is one initialised module - its own interpreter, which holds
// the module's scope and, once the scope/namespacing layer lands, its
// exported namespace.
type loadedModule struct {
	interp *Interpreter
	path   string
}

// EnableModules wires the module system onto the root interpreter: the base
// directory of the entry file (for local import resolution), the module
// search path (system module dir, then any -I dirs), a loader that turns a
// resolved file into a parsed program, and a setup callback that installs
// the standard library into each module's fresh sub-interpreter.
func (i *Interpreter) EnableModules(baseDir string, searchDirs []string, load func(string) (*parser.Program, error), setup func(*Interpreter)) {
	i.baseDir = baseDir
	i.modReg = &moduleReg{
		cache:  map[string]*loadedModule{},
		search: searchDirs,
		load:   load,
		setup:  setup,
	}
}

// loadModuleImports processes a program's `import "..."` statements before
// its body runs, so a module is fully initialised before the code that
// imports it (depth-first post-order). Errors here are load-time errors:
// they fail the program before the importer's body and are not catchable
// (an `import` is a declaration, not an expression).
func (i *Interpreter) loadModuleImports(prog *parser.Program) error {
	if len(prog.ModuleImports) == 0 {
		return nil
	}
	if i.modReg == nil {
		mi := prog.ModuleImports[0]
		file, line, col := posFor(mi)
		return &runtimeError{Msg: "module imports are not enabled in this context (run a program file)", File: file, Line: line, Col: col}
	}
	for _, mi := range prog.ModuleImports {
		if _, err := i.loadModule(mi.Path, mi); err != nil {
			return err
		}
		// Binding mi.AsName (or the file stem) to the loaded module's
		// namespace, so `NAME.member` resolves at the importer, comes with
		// the scope / namespacing layer.
	}
	return nil
}

// loadModule resolves importPath (relative to this interpreter's base dir,
// or the search path for a bare module name), then loads and runs the
// module exactly once, returning the cached instance. `at` positions any
// resolution / cycle error at the import statement.
func (i *Interpreter) loadModule(importPath string, at parser.Node) (*loadedModule, error) {
	reg := i.modReg
	canonical, err := module.Resolve(importPath, i.baseDir, reg.search)
	if err != nil {
		file, line, col := posFor(at)
		return nil, &runtimeError{Msg: err.Error(), File: file, Line: line, Col: col}
	}

	// Cycle: the module is already on the load stack.
	for _, p := range reg.stack {
		if p == canonical {
			file, line, col := posFor(at)
			chain := strings.Join(append(append([]string{}, reg.stack...), canonical), " -> ")
			return nil, &runtimeError{Msg: "module cycle: " + chain, File: file, Line: line, Col: col}
		}
	}

	// Run-once: already loaded and initialised.
	if m, ok := reg.cache[canonical]; ok {
		return m, nil
	}

	// Parse the module file (errors are positioned in that file).
	modProg, err := reg.load(canonical)
	if err != nil {
		return nil, err
	}

	// A fresh sub-interpreter is the module's own scope; it shares the
	// registry so its own imports use the same cache / stack.
	sub := New()
	reg.setup(sub)
	sub.modReg = reg
	sub.baseDir = filepath.Dir(canonical)

	reg.stack = append(reg.stack, canonical)
	runErr := sub.Run(modProg) // loads sub's imports (post-order), then runs its body
	reg.stack = reg.stack[:len(reg.stack)-1]
	if runErr != nil {
		return nil, runErr
	}

	m := &loadedModule{interp: sub, path: canonical}
	reg.cache[canonical] = m
	return m, nil
}
