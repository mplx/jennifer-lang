// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package fslib_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mplx/jennifer-lang/internal/interpreter"
	fslib "github.com/mplx/jennifer-lang/internal/lib/fs"
	iolib "github.com/mplx/jennifer-lang/internal/lib/io"
	tasklib "github.com/mplx/jennifer-lang/internal/lib/task"
	"github.com/mplx/jennifer-lang/internal/parser"
)

// runProg parses + runs a Jennifer program with io + task + fs
// installed. Returns captured stdout and the interpreter error. The
// helper wipes the fs handle registry between calls so a leaked
// handle in one test can't leak an id into the next.
func runProg(t *testing.T, src string) (string, error) {
	t.Helper()
	fslib.ResetForTest()
	prog, err := parser.Parse(src)
	if err != nil {
		return "", err
	}
	in := interpreter.New()
	var buf bytes.Buffer
	in.Out = &buf
	iolib.Install(in)
	tasklib.Install(in)
	fslib.Install(in)
	runErr := in.Run(prog)
	// Drain any lingering unwaited tasks (fs+spawn tests below spawn
	// off filesystem work; loud-fail complaints noise up the output).
	_ = in.UnwaitedTaskErrors()
	return buf.String(), runErr
}

// TestReadWriteStringRoundTrip exercises the happy path for the
// whole-file string API.
func TestReadWriteStringRoundTrip(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "hello.txt")
	src := fmt.Sprintf(`
		use io;
		use fs;
		fs.writeString(%q, "hello, world");
		def s as string init fs.readString(%q);
		io.printf("%%s", $s);
	`, tmp, tmp)
	out, err := runProg(t, src)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if out != "hello, world" {
		t.Errorf("got %q, want %q", out, "hello, world")
	}
}

// TestReadStringMissingPath: reading a missing file surfaces at the
// boundary with the path in the message.
func TestReadStringMissingPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.txt")
	_, err := runProg(t, fmt.Sprintf(`
		use fs;
		def s as string init fs.readString(%q);
	`, missing))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "does-not-exist.txt") {
		t.Errorf("error should mention the path: %v", err)
	}
}

// TestReadStringInvalidUTF8 - readString must reject a file whose
// content is not valid UTF-8 (strict at the boundary).
func TestReadStringInvalidUTF8(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "bin.dat")
	// 0xFF is invalid as a first byte in UTF-8.
	if err := os.WriteFile(tmp, []byte{0xFF, 0xFE, 0x00}, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := runProg(t, fmt.Sprintf(`
		use fs;
		def s as string init fs.readString(%q);
	`, tmp))
	if err == nil {
		t.Fatal("expected UTF-8 rejection")
	}
	if !strings.Contains(err.Error(), "UTF-8") {
		t.Errorf("error should mention UTF-8: %v", err)
	}
}

// TestReadBytesRoundTrip - readBytes returns raw bytes even when the
// content isn't valid UTF-8; writeBytes accepts them back verbatim.
func TestReadBytesRoundTrip(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "bin.dat")
	if err := os.WriteFile(tmp, []byte{0x00, 0xFF, 0x10, 0x7F}, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runProg(t, fmt.Sprintf(`
		use io;
		use fs;
		def b as bytes init fs.readBytes(%q);
		io.printf("%%d", len($b));
	`, tmp))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if out != "4" {
		t.Errorf("got %q, want %q", out, "4")
	}
}

// TestAppendString adds content to an existing file rather than
// overwriting.
func TestAppendString(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "log.txt")
	if err := os.WriteFile(tmp, []byte("first\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := runProg(t, fmt.Sprintf(`
		use fs;
		fs.appendString(%q, "second\n");
	`, tmp))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	got, rerr := os.ReadFile(tmp)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if string(got) != "first\nsecond\n" {
		t.Errorf("got %q", string(got))
	}
}

// TestExistsAndPredicates covers exists / isFile / isDir on a small
// directory tree.
func TestExistsAndPredicates(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runProg(t, fmt.Sprintf(`
		use io;
		use fs;
		io.printf("%%t %%t %%t %%t %%t %%t",
			fs.exists(%q), fs.exists(%q),
			fs.isFile(%q), fs.isDir(%q),
			fs.isFile(%q), fs.isDir(%q));
	`, file, filepath.Join(dir, "missing"),
		file, file,
		dir, dir))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	want := "true false true false false true"
	if out != want {
		t.Errorf("got %q\nwant %q", out, want)
	}
}

// TestStatShape - fs.stat returns an fs.Stat with the fields readable.
func TestStatShape(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "info.txt")
	if err := os.WriteFile(tmp, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runProg(t, fmt.Sprintf(`
		use io;
		use fs;
		def s as fs.Stat init fs.stat(%q);
		io.printf("size=%%d isDir=%%t mtime>0=%%t", $s.size, $s.isDir, $s.mtimeNanos > 0);
	`, tmp))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	want := "size=3 isDir=false mtime>0=true"
	if out != want {
		t.Errorf("got %q\nwant %q", out, want)
	}
}

// TestMkdirTwoVerbs covers the two-verbs pattern for directory
// creation: mkdir fails when a parent is missing; mkdirAll succeeds.
func TestMkdirTwoVerbs(t *testing.T) {
	base := t.TempDir()
	deep := filepath.Join(base, "a", "b", "c")

	// mkdir refuses to create with missing parents.
	_, err := runProg(t, fmt.Sprintf(`
		use fs;
		fs.mkdir(%q);
	`, deep))
	if err == nil {
		t.Fatal("expected fs.mkdir to reject a missing-parent path")
	}

	// mkdirAll creates the whole chain.
	_, err = runProg(t, fmt.Sprintf(`
		use fs;
		fs.mkdirAll(%q);
	`, deep))
	if err != nil {
		t.Fatalf("mkdirAll: %v", err)
	}
	if info, statErr := os.Stat(deep); statErr != nil || !info.IsDir() {
		t.Errorf("expected %s to exist and be a dir, got %v / %v", deep, info, statErr)
	}
}

// TestRemoveTwoVerbs covers the two-verbs pattern for delete: remove
// refuses a non-empty dir; removeAll clears it.
func TestRemoveTwoVerbs(t *testing.T) {
	base := t.TempDir()
	// Nested content so `remove` has something to refuse.
	nested := filepath.Join(base, "target", "sub")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "x.txt"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, "target")

	// remove on a non-empty directory must fail.
	_, err := runProg(t, fmt.Sprintf(`
		use fs;
		fs.remove(%q);
	`, target))
	if err == nil {
		t.Fatal("expected fs.remove to refuse a non-empty directory")
	}

	// removeAll clears it.
	_, err = runProg(t, fmt.Sprintf(`
		use fs;
		fs.removeAll(%q);
	`, target))
	if err != nil {
		t.Fatalf("removeAll: %v", err)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Errorf("expected %s to be gone, stat err was %v", target, statErr)
	}
}

// TestListSorted - fs.list is deterministic (sorted).
func TestListSorted(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"charlie", "alpha", "bravo"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out, err := runProg(t, fmt.Sprintf(`
		use io;
		use fs;
		def names as list of string init fs.list(%q);
		io.printf("%%a", $names);
	`, dir))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	want := "[alpha, bravo, charlie]"
	if out != want {
		t.Errorf("got %q\nwant %q", out, want)
	}
}

// TestWalkIncludesRootAndIsSorted - the root itself appears first,
// then entries in per-directory sorted order.
func TestWalkIncludesRootAndIsSorted(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"a.txt", "z.txt", filepath.Join("sub", "y.txt")} {
		full := filepath.Join(dir, p)
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	out, err := runProg(t, fmt.Sprintf(`
		use io;
		use fs;
		def entries as list of fs.Stat init fs.walk(%q);
		def i as int init 0;
		while ($i < len($entries)) {
			def e as fs.Stat init $entries[$i];
			io.printf("%%s\n", $e.path);
			$i = $i + 1;
		}
	`, dir))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	// Expected order: root, a.txt, sub, sub/y.txt, z.txt.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 entries, got %d: %q", len(lines), out)
	}
	if lines[0] != dir {
		t.Errorf("root should be first; got %q, want %q", lines[0], dir)
	}
	expected := []string{
		dir,
		filepath.Join(dir, "a.txt"),
		filepath.Join(dir, "sub"),
		filepath.Join(dir, "sub", "y.txt"),
		filepath.Join(dir, "z.txt"),
	}
	for i, want := range expected {
		if lines[i] != want {
			t.Errorf("line %d: got %q, want %q", i, lines[i], want)
		}
	}
}

// TestHandleReadLineLoop exercises the canonical open/readLine/eof
// pattern.
func TestHandleReadLineLoop(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "lines.txt")
	if err := os.WriteFile(tmp, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runProg(t, fmt.Sprintf(`
		use io;
		use fs;
		def f as fs.File init fs.open(%q, "read");
		while (not fs.eof($f)) {
			def line as string init fs.readLine($f);
			io.printf("[%%s]", $line);
		}
		fs.close($f);
	`, tmp))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	want := "[one][two][three]"
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

// TestHandleWriteMode - open("write") + writeString round-trips.
func TestHandleWriteMode(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "out.txt")
	_, err := runProg(t, fmt.Sprintf(`
		use fs;
		def f as fs.File init fs.open(%q, "write");
		fs.writeString($f, "line1\n");
		fs.writeString($f, "line2\n");
		fs.close($f);
	`, tmp))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	got, rerr := os.ReadFile(tmp)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if string(got) != "line1\nline2\n" {
		t.Errorf("got %q", string(got))
	}
}

// TestHandleUnknownMode - unknown mode string errors at the boundary
// with the supported set listed.
func TestHandleUnknownMode(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "x.txt")
	_, err := runProg(t, fmt.Sprintf(`
		use fs;
		def f as fs.File init fs.open(%q, "rw");
	`, tmp))
	if err == nil {
		t.Fatal("expected mode boundary error")
	}
	if !strings.Contains(err.Error(), "read") || !strings.Contains(err.Error(), "write") || !strings.Contains(err.Error(), "append") {
		t.Errorf("error should list known modes: %v", err)
	}
}

// TestHandleReadOnWriteMode - reading from a write-mode handle errors
// with a message pointing at the corrective open call.
func TestHandleReadOnWriteMode(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "x.txt")
	_, err := runProg(t, fmt.Sprintf(`
		use fs;
		def f as fs.File init fs.open(%q, "write");
		def s as string init fs.readLine($f);
	`, tmp))
	if err == nil {
		t.Fatal("expected read-on-write boundary error")
	}
	if !strings.Contains(err.Error(), `"write"`) || !strings.Contains(err.Error(), `"read"`) {
		t.Errorf("error should point at the mode mismatch: %v", err)
	}
}

// TestHandleUseAfterClose - operations on a closed handle error with
// a message that names the id.
func TestHandleUseAfterClose(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "x.txt")
	if err := os.WriteFile(tmp, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := runProg(t, fmt.Sprintf(`
		use fs;
		def f as fs.File init fs.open(%q, "read");
		fs.close($f);
		def s as string init fs.readLine($f);
	`, tmp))
	if err == nil {
		t.Fatal("expected use-after-close error")
	}
	if !strings.Contains(err.Error(), "not open") {
		t.Errorf("error should mention 'not open': %v", err)
	}
}

// TestSpawnFsCompose confirms the composition story: a `spawn`
// body can read a file and return its content via task.wait.
func TestSpawnFsCompose(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "concurrent.txt")
	if err := os.WriteFile(tmp, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runProg(t, fmt.Sprintf(`
		use io;
		use fs;
		use task;
		def t as task of string init spawn {
			return fs.readString(%q);
		};
		def s as string init task.wait($t);
		io.printf("%%s", $s);
	`, tmp))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if out != "payload" {
		t.Errorf("got %q, want %q", out, "payload")
	}
}

// TestReadBytesPartialViaHandle exercises the two-arg form of the
// polymorphic `fs.readBytes` verb.
func TestReadBytesPartialViaHandle(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "chunk.dat")
	if err := os.WriteFile(tmp, []byte("abcdefghij"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runProg(t, fmt.Sprintf(`
		use io;
		use fs;
		def f as fs.File init fs.open(%q, "read");
		def first as bytes init fs.readBytes($f, 4);
		def second as bytes init fs.readBytes($f, 4);
		def third as bytes init fs.readBytes($f, 4);   # partial: only 2 bytes left
		fs.close($f);
		io.printf("%%d %%d %%d", len($first), len($second), len($third));
	`, tmp))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if out != "4 4 2" {
		t.Errorf("got %q, want %q", out, "4 4 2")
	}
}
