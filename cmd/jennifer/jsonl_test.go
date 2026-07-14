// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestJsonlFileAndStreaming drives the jsonl module's fs-backed surface
// (writeFile / appendFile / readFile and the streaming Reader) against a real
// temp file. The in-memory encode / decode are covered by the overlay
// (modules/jsonl_test.j); this exercises the parts that need `fs`.
func TestJsonlFileAndStreaming(t *testing.T) {
	dir := t.TempDir()
	dataFile := filepath.Join(dir, "data.jsonl")
	jsonlMod, err := filepath.Abs(filepath.Join("..", "..", "modules", "jsonl.j"))
	if err != nil {
		t.Fatal(err)
	}
	prog := fmt.Sprintf(`import %q as jsonl;
use json;
use testing;
def const PATH as string init %q;

func testWriteReadAppend() {
    jsonl.writeFile(PATH, [json.decode("{\"a\":1}"), json.decode("{\"a\":2}")]);
    jsonl.appendFile(PATH, [json.decode("{\"a\":3}")]);
    def all as list of json.Value init jsonl.readFile(PATH);
    testing.assertEqual(len($all), 3);
    testing.assertEqual(json.asInt(json.get($all[0], "/a")), 1);
    testing.assertEqual(json.asInt(json.get($all[2], "/a")), 3);
}

func testStreaming() {
    def r as jsonl.Reader init jsonl.openReader(PATH);
    def n as int init 0;
    def sum as int init 0;
    while (jsonl.hasMore($r)) {
        def rec as json.Value init jsonl.readRecord($r);
        $n = $n + 1;
        $sum = $sum + json.asInt(json.get($rec, "/a"));
    }
    jsonl.closeReader($r);
    testing.assertEqual($n, 3);
    testing.assertEqual($sum, 6);
}

func testReadRecordThrowsAtEnd() {
    def r as jsonl.Reader init jsonl.openReader(PATH);
    while (jsonl.hasMore($r)) { jsonl.readRecord($r); }
    def threw as bool init false;
    try {
        jsonl.readRecord($r);
    } catch (e) {
        $threw = true;
    }
    jsonl.closeReader($r);
    testing.assertTrue($threw);
}
`, jsonlMod, dataFile)
	progPath := filepath.Join(dir, "prog.j")
	if err := os.WriteFile(progPath, []byte(prog), 0o644); err != nil {
		t.Fatal(err)
	}

	in, code := loadForTest(progPath)
	if in == nil || code != testExitPass {
		t.Fatalf("loadForTest failed: code %d", code)
	}
	// testWriteReadAppend must run first (it creates the file the others read).
	for _, name := range []string{"testWriteReadAppend", "testStreaming", "testReadRecordThrowsAtEnd"} {
		if _, err := in.CallByName(name); err != nil {
			t.Errorf("%s failed: %v", name, err)
		}
	}

	// Independently confirm the file is exactly three JSON lines.
	body, err := os.ReadFile(dataFile)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 JSONL lines, got %d: %q", len(lines), string(body))
	}
	for i, ln := range lines {
		if !strings.HasPrefix(ln, "{") || !strings.HasSuffix(ln, "}") {
			t.Errorf("line %d is not a compact JSON object: %q", i, ln)
		}
	}
}
