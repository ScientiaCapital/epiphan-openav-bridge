package main

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestVISCAEncoders(t *testing.T) {
	cases := []struct {
		name string
		got  []byte
		want []byte
	}{
		{"preset recall 5", viscaPresetRecall(5),
			[]byte{0x81, 0x01, 0x04, 0x3F, 0x02, 0x05, 0xFF}},
		{"preset recall 0", viscaPresetRecall(0),
			[]byte{0x81, 0x01, 0x04, 0x3F, 0x02, 0x00, 0xFF}},
		{"preset set 12", viscaPresetSet(12),
			[]byte{0x81, 0x01, 0x04, 0x3F, 0x01, 0x0C, 0xFF}},
		{"home", viscaHome(),
			[]byte{0x81, 0x01, 0x06, 0x04, 0xFF}},
		{"jog left, tilt stop", viscaJog(viscaPanLeft, viscaTiltStop, 0x10, 0x0E),
			[]byte{0x81, 0x01, 0x06, 0x01, 0x10, 0x0E, 0x01, 0x03, 0xFF}},
		{"jog up-right", viscaJog(viscaPanRight, viscaTiltUp, 0x08, 0x08),
			[]byte{0x81, 0x01, 0x06, 0x01, 0x08, 0x08, 0x02, 0x01, 0xFF}},
		{"stop", viscaStop(0x10, 0x0E),
			[]byte{0x81, 0x01, 0x06, 0x01, 0x10, 0x0E, 0x03, 0x03, 0xFF}},
		// Absolute pan-tilt: pan=0x0052 (the value read after a real jog), tilt=0.
		{"abs pan 0x0052 tilt 0", viscaPanTiltAbsolute(0x06, 0x06, 0x0052, 0x0000),
			[]byte{0x81, 0x01, 0x06, 0x02, 0x06, 0x06, 0x00, 0x00, 0x05, 0x02, 0x00, 0x00, 0x00, 0x00, 0xFF}},
		// Absolute with a negative (two's-complement) pan: -1 -> 0xFFFF -> F F F F.
		{"abs pan -1", viscaPanTiltAbsolute(0x06, 0x06, -1, 0x0000),
			[]byte{0x81, 0x01, 0x06, 0x02, 0x06, 0x06, 0x0F, 0x0F, 0x0F, 0x0F, 0x00, 0x00, 0x00, 0x00, 0xFF}},
		// Zoom direct 0x1000 (the value verified live).
		{"zoom 0x1000", viscaZoomDirect(0x1000),
			[]byte{0x81, 0x01, 0x04, 0x47, 0x01, 0x00, 0x00, 0x00, 0xFF}},
		{"pan/tilt inquiry", viscaPanTiltInquiry(),
			[]byte{0x81, 0x09, 0x06, 0x12, 0xFF}},
		{"zoom inquiry", viscaZoomInquiry(),
			[]byte{0x81, 0x09, 0x04, 0x47, 0xFF}},
		{"version inquiry", viscaVersionInquiry(),
			[]byte{0x81, 0x09, 0x00, 0x02, 0xFF}},
	}
	for _, c := range cases {
		if !bytes.Equal(c.got, c.want) {
			t.Errorf("%s = % X, want % X", c.name, c.got, c.want)
		}
	}
}

func TestVISCANibbleCodec(t *testing.T) {
	for _, v := range []uint16{0x0000, 0x0052, 0x1000, 0x4000, 0xFFFF, 0x1234} {
		if got := unnibble16(nibbles16(v)); got != v {
			t.Errorf("round-trip 0x%04X -> 0x%04X", v, got)
		}
	}
}

func TestParsePanTiltReply(t *testing.T) {
	// 90 50 00 00 05 02 FF FF FF FE FF  -> pan=0x0052 (82), tilt=0xFFFE (-2)
	frame := []byte{0x90, 0x50, 0x00, 0x00, 0x05, 0x02, 0x0F, 0x0F, 0x0F, 0x0E, 0xFF}
	pan, tilt, err := parsePanTiltReply(frame)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pan != 0x0052 {
		t.Errorf("pan = %d, want 82", pan)
	}
	if tilt != -2 {
		t.Errorf("tilt = %d, want -2", tilt)
	}
	if _, _, err := parsePanTiltReply([]byte{0x90, 0x50, 0xFF}); err == nil {
		t.Error("expected error on short frame")
	}
}

func TestParseZoomReply(t *testing.T) {
	z, err := parseZoomReply([]byte{0x90, 0x50, 0x01, 0x00, 0x00, 0x00, 0xFF})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if z != 0x1000 {
		t.Errorf("zoom = 0x%04X, want 0x1000", z)
	}
}

// fakeVISCAServer starts a TCP listener that reads one VISCA frame, records it,
// and writes back the given reply frames (e.g. ACK then completion). It returns
// the listener address and a func to fetch the captured request. Mirrors how the
// old httptest mock injected a fake device — here for raw VISCA over TCP.
func fakeVISCAServer(t *testing.T, replies ...[]byte) (host, port string, got func() []byte) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	var captured []byte
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 64)
		n, _ := conn.Read(buf)
		captured = append(captured, buf[:n]...)
		for _, r := range replies {
			conn.Write(r)
		}
	}()

	h, p, _ := net.SplitHostPort(ln.Addr().String())
	return h, p, func() []byte { <-done; return captured }
}

func TestVISCASend_SwallowsAckReturnsCompletion(t *testing.T) {
	host, port, got := fakeVISCAServer(t,
		[]byte{0x90, 0x41, 0xFF}, // ACK
		[]byte{0x90, 0x51, 0xFF}, // completion
	)
	old := viscaPort
	viscaPort = port
	defer func() { viscaPort = old }()

	reply, err := viscaSend("admin:x@"+host+":80", viscaHome())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(reply, []byte{0x90, 0x51, 0xFF}) {
		t.Errorf("reply = % X, want 90 51 FF", reply)
	}
	if !bytes.Equal(got(), viscaHome()) {
		t.Errorf("camera received % X, want % X", got(), viscaHome())
	}
}

func TestVISCASend_InquiryReturnsData(t *testing.T) {
	want := []byte{0x90, 0x50, 0x00, 0x52, 0xFF} // version-style data completion
	host, port, _ := fakeVISCAServer(t, want)
	old := viscaPort
	viscaPort = port
	defer func() { viscaPort = old }()

	reply, err := viscaSend("admin:x@"+host+":80", viscaVersionInquiry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(reply, want) {
		t.Errorf("reply = % X, want % X", reply, want)
	}
}

func TestVISCASend_ErrorReply(t *testing.T) {
	host, port, _ := fakeVISCAServer(t, []byte{0x90, 0x60, 0x02, 0xFF}) // syntax error
	old := viscaPort
	viscaPort = port
	defer func() { viscaPort = old }()

	if _, err := viscaSend("admin:x@"+host+":80", viscaHome()); err == nil {
		t.Fatal("expected error on VISCA error reply")
	}
}

func TestVISCASend_ConnectionRefused(t *testing.T) {
	old := viscaPort
	viscaPort = "1"
	defer func() { viscaPort = old }()
	if _, err := viscaSend("admin:x@127.0.0.1:80", viscaHome()); err == nil {
		t.Fatal("expected dial error")
	}
}
