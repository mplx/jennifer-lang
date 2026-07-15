// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// wsAccept computes the RFC 6455 Sec-WebSocket-Accept for a client key.
func wsAccept(key string) string {
	h := sha1.New()
	h.Write([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// readClientFrame reads one masked client frame and returns its opcode and
// unmasked payload.
func readClientFrame(r *bufio.Reader) (byte, []byte, error) {
	h := make([]byte, 2)
	if _, err := io.ReadFull(r, h); err != nil {
		return 0, nil, err
	}
	opcode := h[0] & 0x0f
	masked := h[1]&0x80 != 0
	n := int(h[1] & 0x7f)
	switch n {
	case 126:
		e := make([]byte, 2)
		io.ReadFull(r, e)
		n = int(e[0])<<8 | int(e[1])
	case 127:
		e := make([]byte, 8)
		io.ReadFull(r, e)
		n = 0
		for _, b := range e {
			n = n<<8 | int(b)
		}
	}
	var mask []byte
	if masked {
		mask = make([]byte, 4)
		io.ReadFull(r, mask)
	}
	payload := make([]byte, n)
	io.ReadFull(r, payload)
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return opcode, payload, nil
}

// encodeServerFrame builds an unmasked server frame (FIN set).
func encodeServerFrame(opcode byte, payload []byte) []byte {
	out := []byte{0x80 | opcode}
	n := len(payload)
	switch {
	case n < 126:
		out = append(out, byte(n))
	case n < 65536:
		out = append(out, 126, byte(n>>8), byte(n))
	default:
		out = append(out, 127)
		for s := 56; s >= 0; s -= 8 {
			out = append(out, byte(n>>uint(s)))
		}
	}
	return append(out, payload...)
}

// serveWSEcho does one RFC 6455 handshake then echoes each data frame back
// (unmasked), returning when the client sends a close frame.
func serveWSEcho(ln net.Listener) {
	conn, err := ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	r := bufio.NewReader(conn)
	var key string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "sec-websocket-key:") {
			key = strings.TrimSpace(line[len("sec-websocket-key:"):])
		}
	}
	fmt.Fprint(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\n"+
		"Connection: Upgrade\r\nSec-WebSocket-Accept: "+wsAccept(key)+"\r\n\r\n")
	for {
		opcode, payload, err := readClientFrame(r)
		if err != nil || opcode == 0x8 { // error or close
			return
		}
		conn.Write(encodeServerFrame(opcode, payload))
	}
}

// TestWebsocketRoundTrip drives the websocket client through a real handshake
// (accept-key verified by the module) plus a masked text send and a binary
// send, each echoed by a minimal in-process server, proving the full
// handshake + framing + masking path.
func TestWebsocketRoundTrip(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go serveWSEcho(ln)

	wsMod, err := filepath.Abs(filepath.Join("..", "..", "modules", "websocket.j"))
	if err != nil {
		t.Fatal(err)
	}
	url := "ws://" + ln.Addr().String() + "/"
	dir := t.TempDir()
	prog := fmt.Sprintf(`use testing;
use convert;
import %q as websocket;
def ws as websocket.Conn init websocket.connect(%q);

websocket.send($ws, "hello world");
def m as websocket.Message init websocket.receive($ws);
testing.assertEqual($m.kind, "text");
testing.assertEqual($m.text, "hello world");

websocket.sendBytes($ws, convert.bytesFromString("bin", "utf-8"));
def mb as websocket.Message init websocket.receive($ws);
testing.assertEqual($mb.kind, "binary");
testing.assertEqual(len($mb.data), 3);
testing.assertEqual($mb.data[0], 98);

websocket.close($ws);`, wsMod, url)
	progPath := filepath.Join(dir, "ws.j")
	if err := os.WriteFile(progPath, []byte(prog), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, code := loadForTest(progPath); code != testExitPass {
		t.Fatalf("websocket program failed with code %d", code)
	}
}
