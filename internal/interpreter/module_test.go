// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package interpreter_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mplx/jennifer-lang/internal/interpreter"
	"github.com/mplx/jennifer-lang/internal/lexer"
	iolib "github.com/mplx/jennifer-lang/internal/lib/io"
	"github.com/mplx/jennifer-lang/internal/parser"
	"github.com/mplx/jennifer-lang/internal/preproc"
)

// moduleProgram lexes, preprocesses, and parses a module file - the same
// pipeline the CLI's loader runs, so the tests exercise real path resolution.
func moduleProgram(path string) (*parser.Program, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	toks, err := lexer.TokenizeWithFile(string(src), path)
	if err != nil {
		return nil, err
	}
	toks, err = preproc.Process(toks, filepath.Dir(path), path)
	if err != nil {
		return nil, err
	}
	return parser.ParseTokens(toks)
}

// runModuleMain writes the named files into a temp dir, then runs main.j with
// the module system enabled. Every module interpreter shares one output
// buffer so init order is observable. Returns the combined output and the run
// error.
func runModuleMain(t *testing.T, files map[string]string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	var buf bytes.Buffer
	setup := func(s *interpreter.Interpreter) {
		s.Out = &buf
		iolib.Install(s)
	}
	in := interpreter.New()
	setup(in)
	in.EnableModules(dir, nil, moduleProgram, setup)

	mainPath := filepath.Join(dir, "main.j")
	prog, err := moduleProgram(mainPath)
	if err != nil {
		t.Fatalf("parse main: %v", err)
	}
	runErr := in.Run(prog)
	return buf.String(), runErr
}

// logHelper is a per-module method that prints its name; each module carries
// its own copy (a fresh sub-interpreter has its own scope, so the name never
// collides across modules).
const logHelper = ` func log(n as string) { io.printf("%s ", $n); return 0; } `

func TestModulePostOrderRunOnce(t *testing.T) {
	// main -> a -> {b -> c, c}. c is imported twice but must init once, and
	// every module must init before the code that imports it: c, b, a, main.
	out, err := runModuleMain(t, map[string]string{
		"c.j":    `use io; def const C as int init log("c");` + logHelper,
		"b.j":    `use io; import "./c.j"; def const B as int init log("b");` + logHelper,
		"a.j":    `use io; import "./b.j"; import "./c.j"; def const A as int init log("a");` + logHelper,
		"main.j": `use io; import "./a.j"; io.printf("main");`,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if got, want := out, "c b a main"; got != want {
		t.Errorf("init order = %q, want %q", got, want)
	}
}

func TestModuleCycleRejected(t *testing.T) {
	_, err := runModuleMain(t, map[string]string{
		"x.j":    `import "./y.j"; def const X as int init 1;`,
		"y.j":    `import "./x.j"; def const Y as int init 1;`,
		"main.j": `import "./x.j";`,
	})
	if err == nil {
		t.Fatal("expected a module-cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "module cycle") {
		t.Errorf("error should name the cycle: %v", err)
	}
	// The chain names every edge that closes the loop.
	if !strings.Contains(err.Error(), "x.j") || !strings.Contains(err.Error(), "y.j") {
		t.Errorf("cycle chain should name both modules: %v", err)
	}
}

func TestModuleParseErrorPositioned(t *testing.T) {
	_, err := runModuleMain(t, map[string]string{
		"bad.j":  `def x as `, // truncated - parse error inside the module
		"main.j": `import "./bad.j";`,
	})
	if err == nil {
		t.Fatal("expected a parse error from the imported module, got nil")
	}
	if !strings.Contains(err.Error(), "bad.j") {
		t.Errorf("parse error should be positioned in the imported file: %v", err)
	}
}

func TestModuleInitThrowFailsAtLoad(t *testing.T) {
	// A throwing initializer fails at load, before the importer's body runs.
	out, err := runModuleMain(t, map[string]string{
		"boom.j": `def const B as int init blow();
			func blow() { throw Error{kind: "x", message: "boom", file: "", line: 0, col: 0}; return 0; }`,
		"main.j": `use io; import "./boom.j"; io.printf("AFTER");`,
	})
	if err == nil {
		t.Fatal("expected the throwing initializer to fail the load, got nil")
	}
	if strings.Contains(out, "AFTER") {
		t.Errorf("importer body ran despite a load-time throw: %q", out)
	}
}

func TestModuleImportsWithoutEnableError(t *testing.T) {
	// A program with module imports but no EnableModules (an embedding that
	// never wired the module system) fails with a clear message rather than
	// silently ignoring the import.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dep.j"), []byte(`def const D as int init 1;`), 0o644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.j")
	if err := os.WriteFile(mainPath, []byte(`import "./dep.j";`), 0o644); err != nil {
		t.Fatal(err)
	}
	prog, err := moduleProgram(mainPath)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	in := interpreter.New()
	iolib.Install(in)
	if err := in.Run(prog); err == nil {
		t.Fatal("expected an error when modules are not enabled, got nil")
	}
}

// TestModuleStructZeroValueAndFieldWrite exercises the cross-module struct
// resolution lookupStructDef / zeroStructFor gained: `def x as m.Struct;`
// (zero value, no init) and `$x.field = ...` both resolve an imported module
// struct's definition, and a nested module struct zeroes and writes through.
// Regression for the "unknown struct" (zero value) and "definition missing"
// (field write) errors those paths raised because module structs live in the
// module's own interpreter, not i.NSStructs.
func TestModuleStructZeroValueAndFieldWrite(t *testing.T) {
	out, err := runModuleMain(t, map[string]string{
		"m.j": `export def struct Inner { n as int };
export def struct Point { tag as string, inner as Inner };`,
		"main.j": `use io;
import "./m.j" as m;
def p as m.Point;
io.printf("zero tag=[%s] n=%d\n", $p.tag, $p.inner.n);
$p.tag = "hi";
$p.inner.n = 7;
io.printf("wrote tag=[%s] n=%d\n", $p.tag, $p.inner.n);`,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	want := "zero tag=[] n=0\nwrote tag=[hi] n=7\n"
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}
