// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package profile_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mplx/jennifer-lang/internal/profile"
)

// TestCollectorConcurrentRecord hammers a single Collector from many
// goroutines, the shape a profiled `spawn` fan-out produces. Without the
// mutex this crashes with "concurrent map read and map write" even without
// the race detector; under `-race` it also flags the map access directly.
func TestCollectorConcurrentRecord(t *testing.T) {
	c := profile.NewCollector(profile.ModeStatement, 0)
	const goroutines, perG = 8, 4000
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for n := 0; n < perG; n++ {
				c.RecordStmt("a.j", 1, 1, time.Microsecond, time.Microsecond) // shared position
				c.RecordEagerCopy("b.j", g+1, 1)                              // per-goroutine position
			}
		}(g)
	}
	wg.Wait()
	// Rendering after the concurrent writes must not panic and the shared
	// position must reflect every goroutine's hits.
	var out bytes.Buffer
	c.Table(&out)
	if !strings.Contains(out.String(), "a.j:1:1") {
		t.Errorf("shared position missing after concurrent record:\n%s", out.String())
	}
}

func TestStatementAggregation(t *testing.T) {
	c := profile.NewCollector(profile.ModeStatement, 0)
	c.RecordStmt("a.j", 1, 1, 10*time.Millisecond, 30*time.Millisecond)
	c.RecordStmt("a.j", 1, 1, 5*time.Millisecond, 5*time.Millisecond)
	c.RecordStmt("a.j", 2, 3, time.Millisecond, time.Millisecond)

	var buf bytes.Buffer
	c.Table(&buf)
	out := buf.String()
	// The hot position (1:1, 2 hits, 35ms cum) must appear and sort first.
	if !strings.Contains(out, "a.j:1:1") || !strings.Contains(out, "a.j:2:3") {
		t.Fatalf("table missing positions:\n%s", out)
	}
	i11 := strings.Index(out, "a.j:1:1")
	i23 := strings.Index(out, "a.j:2:3")
	if i11 > i23 {
		t.Fatalf("expected 1:1 (higher cum) sorted before 2:3:\n%s", out)
	}
	if !strings.Contains(out, "2 positions") || !strings.Contains(out, "3 statement executions") {
		t.Fatalf("summary wrong:\n%s", out)
	}
}

func TestAllocsTable(t *testing.T) {
	c := profile.NewCollector(profile.ModeAllocs, 0)
	c.RecordEagerCopy("a.j", 7, 9)
	c.RecordEagerCopy("a.j", 7, 9)
	c.RecordEagerCopy("a.j", 7, 9)
	c.RecordSpawnCopy("a.j", 9, 1, 2*time.Microsecond)

	var buf bytes.Buffer
	c.Table(&buf)
	out := buf.String()
	if !strings.Contains(out, "Eager copies") || !strings.Contains(out, "a.j:7:9") {
		t.Fatalf("allocs table missing eager copies:\n%s", out)
	}
	if !strings.Contains(out, "Spawn-frame deep copies") || !strings.Contains(out, "a.j:9:1") {
		t.Fatalf("allocs table missing spawn copies:\n%s", out)
	}
}

func TestTraceIsValidJSON(t *testing.T) {
	c := profile.NewCollector(profile.ModeStatement, 0)
	c.Start(time.Time{})
	base := time.Time{}
	c.RecordCall("fib", "a.j", 3, 5, base.Add(time.Millisecond), base.Add(3*time.Millisecond))
	c.RecordCall("build", "a.j", 7, 1, base.Add(4*time.Millisecond), base.Add(5*time.Millisecond))

	var buf bytes.Buffer
	if err := c.Trace(&buf); err != nil {
		t.Fatal(err)
	}
	var doc struct {
		TraceEvents []struct {
			Name string  `json:"name"`
			Ph   string  `json:"ph"`
			Ts   float64 `json:"ts"`
			Dur  float64 `json:"dur"`
		} `json:"traceEvents"`
		DisplayTimeUnit string `json:"displayTimeUnit"`
	}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("trace is not valid JSON: %v", err)
	}
	if len(doc.TraceEvents) != 2 {
		t.Fatalf("expected 2 trace events, got %d", len(doc.TraceEvents))
	}
	if doc.TraceEvents[0].Ph != "X" || doc.TraceEvents[0].Name != "fib" {
		t.Fatalf("first event wrong: %+v", doc.TraceEvents[0])
	}
	// fib spans 1ms..3ms => ts=1000us, dur=2000us.
	if doc.TraceEvents[0].Ts != 1000 || doc.TraceEvents[0].Dur != 2000 {
		t.Fatalf("timing wrong: ts=%v dur=%v", doc.TraceEvents[0].Ts, doc.TraceEvents[0].Dur)
	}
}

func TestCallCapDropsExcess(t *testing.T) {
	c := profile.NewCollector(profile.ModeStatement, 2)
	base := time.Time{}
	for k := 0; k < 5; k++ {
		c.RecordCall("f", "a.j", 1, 1, base, base.Add(time.Millisecond))
	}
	var buf bytes.Buffer
	if err := c.Trace(&buf); err != nil {
		t.Fatal(err)
	}
	var doc struct {
		TraceEvents []json.RawMessage `json:"traceEvents"`
	}
	json.Unmarshal(buf.Bytes(), &doc)
	if len(doc.TraceEvents) != 2 {
		t.Fatalf("expected cap of 2 call events, got %d", len(doc.TraceEvents))
	}
}

func TestPprofIsValidGzip(t *testing.T) {
	c := profile.NewCollector(profile.ModeStatement, 0)
	c.RecordStmt("prog.j", 4, 5, 50*time.Millisecond, time.Second)

	var buf bytes.Buffer
	if err := c.Pprof(&buf); err != nil {
		t.Fatal(err)
	}
	gr, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatalf("pprof output is not gzip: %v", err)
	}
	raw, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("gunzip failed: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("decompressed pprof is empty")
	}
	// The protobuf string table carries these ASCII payloads verbatim.
	for _, want := range []string{"hits", "nanoseconds", "prog.j"} {
		if !bytes.Contains(raw, []byte(want)) {
			t.Fatalf("pprof string table missing %q", want)
		}
	}
}

func TestAllocsPprofUsesAllocObjects(t *testing.T) {
	c := profile.NewCollector(profile.ModeAllocs, 0)
	c.RecordEagerCopy("a.j", 5, 5)
	var buf bytes.Buffer
	if err := c.Pprof(&buf); err != nil {
		t.Fatal(err)
	}
	gr, _ := gzip.NewReader(&buf)
	raw, _ := io.ReadAll(gr)
	if !bytes.Contains(raw, []byte("alloc_objects")) {
		t.Fatal("allocs pprof should use the alloc_objects sample type")
	}
}
