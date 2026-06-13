// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package oslib

import (
	"bytes"
	"fmt"
	stdos "os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"

	"github.com/mplx/jennifer-lang/internal/interpreter"
	"github.com/mplx/jennifer-lang/internal/parser"
)

// execSupported reports whether the host's Go runtime actually
// supports `os/exec`. TinyGo (today) compiles `os/exec` but the
// underlying syscall layer panics or errors with cryptic messages
// like "files setting not implemented". The pre-check turns that
// into a friendly Jennifer-level error pointing at the build choice.
func execSupported() bool {
	return runtime.Compiler != "tinygo"
}

func execUnsupportedErr(fnName string) error {
	return fmt.Errorf("%s: external-program execution is not supported by this build (`%s`); use the `jennifer-go` binary (`make build` produces both, or `make build-go` for just the Go binary)", fnName, runtime.Compiler)
}

// processState holds the Go-side state for an `os.spawn`'d child.
// The Jennifer-visible `os.Process` value only carries the pid; the
// rest of the bookkeeping lives in this struct and is reached
// through the handles map keyed by pid.
type processState struct {
	cmd      *exec.Cmd
	stdout   *bytes.Buffer
	stderr   *bytes.Buffer
	done     chan struct{} // closed when cmd.Wait() returns
	exitCode int64
	waitErr  error
}

var (
	handlesMu sync.Mutex
	handles   = map[int64]*processState{}
)

// argvFromList unwraps a Jennifer `list of string` into a Go []string.
// Empty list and non-list values produce a typed error tagged with
// the caller's function name (`os.run` / `os.spawn`).
func argvFromList(fnName string, v interpreter.Value) ([]string, error) {
	if v.Kind != interpreter.KindList {
		return nil, fmt.Errorf("%s: argv must be a list of string, got %s", fnName, v.Kind)
	}
	if len(v.List) == 0 {
		return nil, fmt.Errorf("%s: argv must contain at least one element (the program name)", fnName)
	}
	out := make([]string, len(v.List))
	for i, elem := range v.List {
		if elem.Kind != interpreter.KindString {
			return nil, fmt.Errorf("%s: argv[%d] must be string, got %s", fnName, i, elem.Kind)
		}
		out[i] = elem.Str
	}
	return out, nil
}

// makeResult constructs an `os.Result{...}` Value from the captured
// streams and exit code. Used by both `os.run` (synchronous path) and
// `os.wait` (waiting on a `spawn`'d handle).
func makeResult(exitCode int64, stdout, stderr string) interpreter.Value {
	return interpreter.NamespacedStructVal("os", "Result", []interpreter.StructField{
		{Name: "exitCode", Value: interpreter.IntVal(exitCode)},
		{Name: "stdout", Value: interpreter.StringVal(stdout)},
		{Name: "stderr", Value: interpreter.StringVal(stderr)},
	})
}

// makeProcess constructs an `os.Process{pid}` Value.
func makeProcess(pid int64) interpreter.Value {
	return interpreter.NamespacedStructVal("os", "Process", []interpreter.StructField{
		{Name: "pid", Value: interpreter.IntVal(pid)},
	})
}

// extractPid pulls the pid field out of an `os.Process` value. A
// non-Process value is a typed runtime error; missing field is an
// internal error (the struct was built outside this library).
func extractPid(fnName string, v interpreter.Value) (int64, error) {
	if v.Kind != interpreter.KindStruct || v.StructNS != "os" || v.StructName != "Process" {
		return 0, fmt.Errorf("%s: argument must be an os.Process, got %s", fnName, v.Kind)
	}
	for _, f := range v.Fields {
		if f.Name == "pid" {
			if f.Value.Kind != interpreter.KindInt {
				return 0, fmt.Errorf("%s: os.Process.pid is not int (got %s)", fnName, f.Value.Kind)
			}
			return f.Value.Int, nil
		}
	}
	return 0, fmt.Errorf("%s: os.Process has no pid field", fnName)
}

// runFn implements `os.run(argv) -> os.Result`. Blocking: runs the
// command to completion and returns the captured streams and exit
// code as a single result struct. A non-zero exit code is NOT an
// error - the caller branches on `$result.exitCode`. Boundary
// failures (program not found, not executable, fork/exec failure)
// are typed runtime errors.
func runFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if !execSupported() {
		return interpreter.Null(), execUnsupportedErr("os.run")
	}
	if len(args) != 1 {
		return interpreter.Null(), fmt.Errorf("os.run expects 1 argument (argv list), got %d", len(args))
	}
	argv, err := argvFromList("os.run", args[0])
	if err != nil {
		return interpreter.Null(), err
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		// ExitError means the process ran and reported non-zero exit;
		// that's a result, not a boundary error.
		if exitErr, ok := err.(*exec.ExitError); ok {
			return makeResult(int64(exitErr.ExitCode()), outBuf.String(), errBuf.String()), nil
		}
		return interpreter.Null(), fmt.Errorf("os.run: %v", err)
	}
	return makeResult(int64(cmd.ProcessState.ExitCode()), outBuf.String(), errBuf.String()), nil
}

// spawnFn implements `os.spawn(argv) -> os.Process`. Non-blocking:
// starts the child and returns immediately with a handle. A
// background goroutine calls cmd.Wait() and records the result;
// `os.wait` / `os.poll` consult the recorded state.
func spawnFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if !execSupported() {
		return interpreter.Null(), execUnsupportedErr("os.spawn")
	}
	if len(args) != 1 {
		return interpreter.Null(), fmt.Errorf("os.spawn expects 1 argument (argv list), got %d", len(args))
	}
	argv, err := argvFromList("os.spawn", args[0])
	if err != nil {
		return interpreter.Null(), err
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	state := &processState{
		cmd:    cmd,
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		done:   make(chan struct{}),
	}
	cmd.Stdout = state.stdout
	cmd.Stderr = state.stderr
	if err := cmd.Start(); err != nil {
		return interpreter.Null(), fmt.Errorf("os.spawn: %v", err)
	}
	pid := int64(cmd.Process.Pid)
	handlesMu.Lock()
	handles[pid] = state
	handlesMu.Unlock()
	go func() {
		err := cmd.Wait()
		handlesMu.Lock()
		state.waitErr = err
		if cmd.ProcessState != nil {
			state.exitCode = int64(cmd.ProcessState.ExitCode())
		}
		close(state.done)
		handlesMu.Unlock()
	}()
	return makeProcess(pid), nil
}

// waitFn implements `os.wait(p) -> os.Result`. Blocks until the
// process terminates, then returns the captured streams and exit
// code. Subsequent calls on the same handle return the same result
// (idempotent).
func waitFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if len(args) != 1 {
		return interpreter.Null(), fmt.Errorf("os.wait expects 1 argument (process handle), got %d", len(args))
	}
	pid, err := extractPid("os.wait", args[0])
	if err != nil {
		return interpreter.Null(), err
	}
	handlesMu.Lock()
	state, ok := handles[pid]
	handlesMu.Unlock()
	if !ok {
		return interpreter.Null(), fmt.Errorf("os.wait: unknown process handle (pid %d)", pid)
	}
	<-state.done
	return makeResult(state.exitCode, state.stdout.String(), state.stderr.String()), nil
}

// pollFn implements `os.poll(p) -> bool`. Pure predicate, no side
// effects: true if and only if a following `os.wait` would return
// immediately.
func pollFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if len(args) != 1 {
		return interpreter.Null(), fmt.Errorf("os.poll expects 1 argument (process handle), got %d", len(args))
	}
	pid, err := extractPid("os.poll", args[0])
	if err != nil {
		return interpreter.Null(), err
	}
	handlesMu.Lock()
	state, ok := handles[pid]
	handlesMu.Unlock()
	if !ok {
		return interpreter.Null(), fmt.Errorf("os.poll: unknown process handle (pid %d)", pid)
	}
	select {
	case <-state.done:
		return interpreter.BoolVal(true), nil
	default:
		return interpreter.BoolVal(false), nil
	}
}

// killFn implements `os.kill(p)`. Sends SIGTERM to the child. A
// subsequent `os.wait` returns whatever the OS reports for the
// terminated process. Signal variants beyond SIGTERM are
// deliberately out of scope - users who need them reach for a
// future enhancement.
func killFn(_ interpreter.BuiltinCtx, args []interpreter.Value) (interpreter.Value, error) {
	if len(args) != 1 {
		return interpreter.Null(), fmt.Errorf("os.kill expects 1 argument (process handle), got %d", len(args))
	}
	pid, err := extractPid("os.kill", args[0])
	if err != nil {
		return interpreter.Null(), err
	}
	handlesMu.Lock()
	state, ok := handles[pid]
	handlesMu.Unlock()
	if !ok {
		return interpreter.Null(), fmt.Errorf("os.kill: unknown process handle (pid %d)", pid)
	}
	if state.cmd.Process == nil {
		return interpreter.Null(), fmt.Errorf("os.kill: process has no live OS handle")
	}
	if err := state.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// "process already finished" is a benign race - the wait
		// goroutine just hasn't recorded yet. Surface as a no-op.
		if err != stdos.ErrProcessDone {
			return interpreter.Null(), fmt.Errorf("os.kill: %v", err)
		}
	}
	return interpreter.Null(), nil
}

// parser import for the package - used in oslib.go's Install for
// struct field types. Re-exported here so go build doesn't drop the
// import when oslib.go is the only consumer.
var _ = parser.PrimitiveType
