package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// ========== fake VISCA-over-TCP device (motion control plane) ==========

// viscaReplyFor returns the reply frames a fake EC20 sends for a request frame.
// Commands (81 01 …) get an ACK then a completion; inquiries (81 09 …) get a
// single data completion whose shape matches the parser the driver runs.
func viscaReplyFor(frame []byte) [][]byte {
	if len(frame) >= 4 && frame[1] == 0x09 { // inquiry
		switch {
		case frame[2] == 0x06 && frame[3] == 0x12: // pan/tilt inquiry -> pan 0x0052, tilt 0
			return [][]byte{{0x90, 0x50, 0x00, 0x00, 0x05, 0x02, 0x00, 0x00, 0x00, 0x00, 0xFF}}
		case frame[2] == 0x04 && frame[3] == 0x47: // zoom inquiry -> 0x1000
			return [][]byte{{0x90, 0x50, 0x01, 0x00, 0x00, 0x00, 0xFF}}
		case frame[2] == 0x00 && frame[3] == 0x02: // version inquiry
			return [][]byte{{0x90, 0x50, 0x00, 0x52, 0xFF}}
		}
		return [][]byte{{0x90, 0x50, 0xFF}} // generic completion
	}
	return [][]byte{{0x90, 0x41, 0xFF}, {0x90, 0x51, 0xFF}} // ACK + completion
}

// fakeVISCADeviceWithReply starts a TCP listener that emulates the EC20's raw
// VISCA plane. It LOOPS accepting connections — controlPTZ and the status/
// position reads each open TWO connections (one frame apiece) — reads one frame
// per connection, records it, and writes back reply(frame). It points viscaPort
// at its ephemeral port and restores it + closes the listener via t.Cleanup.
// The returned socketKey carries a deliberately-wrong ":80" host port to prove
// viscaSend ignores it and dials viscaPort instead. frames() returns a
// thread-safe copy of every request frame the device received.
func fakeVISCADeviceWithReply(t *testing.T, reply func([]byte) [][]byte) (socketKey string, frames func() [][]byte) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	var mu sync.Mutex
	var captured [][]byte

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed by cleanup
			}
			go func(c net.Conn) {
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(2 * time.Second))
				buf := make([]byte, 64)
				n, err := c.Read(buf)
				if n <= 0 || (err != nil && n == 0) {
					return
				}
				frame := append([]byte(nil), buf[:n]...)
				mu.Lock()
				captured = append(captured, frame)
				mu.Unlock()
				for _, r := range reply(frame) {
					_, _ = c.Write(r)
				}
			}(conn)
		}
	}()

	old := viscaPort
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	viscaPort = port
	t.Cleanup(func() {
		viscaPort = old
		ln.Close()
	})

	frames = func() [][]byte {
		mu.Lock()
		defer mu.Unlock()
		out := make([][]byte, len(captured))
		copy(out, captured)
		return out
	}
	return "admin:x@127.0.0.1:80", frames
}

// fakeVISCADevice is the common case: a healthy device that ACKs commands and
// answers inquiries with well-formed data completions.
func fakeVISCADevice(t *testing.T) (socketKey string, frames func() [][]byte) {
	return fakeVISCADeviceWithReply(t, viscaReplyFor)
}

// ========== parseSocketKey tests ==========

func TestParseSocketKey_WithCredentials(t *testing.T) {
	host, user, pass := parseSocketKey("admin:secret@192.168.1.100:80")
	if host != "192.168.1.100:80" {
		t.Errorf("expected host 192.168.1.100:80, got %s", host)
	}
	if user != "admin" {
		t.Errorf("expected user admin, got %s", user)
	}
	if pass != "secret" {
		t.Errorf("expected pass secret, got %s", pass)
	}
}

func TestParseSocketKey_WithProtocolPrefix(t *testing.T) {
	host, user, pass := parseSocketKey("tcp|admin:secret@192.168.1.100:80")
	if host != "192.168.1.100:80" {
		t.Errorf("expected host 192.168.1.100:80, got %s", host)
	}
	if user != "admin" {
		t.Errorf("expected user admin, got %s", user)
	}
	if pass != "secret" {
		t.Errorf("expected pass secret, got %s", pass)
	}
}

func TestParseSocketKey_NoCredentials(t *testing.T) {
	host, user, pass := parseSocketKey("192.168.1.100:80")
	if host != "192.168.1.100:80" {
		t.Errorf("expected host 192.168.1.100:80, got %s", host)
	}
	if user != "" {
		t.Errorf("expected empty user, got %s", user)
	}
	if pass != "" {
		t.Errorf("expected empty pass, got %s", pass)
	}
}

func TestParseSocketKey_PasswordWithColon(t *testing.T) {
	host, user, pass := parseSocketKey("admin:pass:word@192.168.1.100:80")
	if host != "192.168.1.100:80" {
		t.Errorf("expected host 192.168.1.100:80, got %s", host)
	}
	if user != "admin" {
		t.Errorf("expected user admin, got %s", user)
	}
	if pass != "pass:word" {
		t.Errorf("expected pass pass:word, got %s", pass)
	}
}

// ========== GET function tests ==========

func TestGetCameraStatus_Success(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	result, err := getCameraStatus(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if parsed["online"] != true {
		t.Errorf("expected online true, got %v", parsed["online"])
	}
	if _, ok := parsed["pan_units"]; !ok {
		t.Errorf("expected a pan_units key in result, got: %s", result)
	}
}

func TestGetPTZPosition_Success(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	result, err := getPTZPosition(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if _, ok := parsed["pan_units"]; !ok {
		t.Errorf("expected a pan_units key in result, got: %s", result)
	}
}

func TestGetPresets_Success(t *testing.T) {
	// VISCA has no list-presets inquiry; getPresets returns a constant, no network.
	result, err := getPresets("admin:x@127.0.0.1:80")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "unsupported") {
		t.Errorf("expected 'unsupported' in result, got: %s", result)
	}
}

// TestGetPreview_Success verifies preview now resolves to the device RTSP stream
// URL (no network call): the framework-appended :80 host port is stripped and
// :554 (the RTSP default) is used.
func TestGetPreview_Success(t *testing.T) {
	result, err := getPreview("admin:x@127.0.0.1:80")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var url string
	if err := json.Unmarshal([]byte(result), &url); err != nil {
		t.Fatalf("result is not a valid JSON string: %v", err)
	}
	if !strings.HasPrefix(url, "rtsp://") {
		t.Errorf("expected an rtsp:// URL, got: %s", url)
	}
	if url != "rtsp://127.0.0.1:554/1" {
		t.Errorf("expected rtsp://127.0.0.1:554/1, got: %s", url)
	}
}

func TestHealthCheck_Success(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	result, err := healthCheck(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != `"true"` {
		t.Errorf("expected \"true\", got %s", result)
	}
}

func TestHealthCheck_Failure(t *testing.T) {
	// Point at a non-existent server
	result, err := healthCheck("admin:testpass@127.0.0.1:1")
	if err != nil {
		t.Fatalf("healthCheck should not return error, got: %v", err)
	}
	if result != `"false"` {
		t.Errorf("expected \"false\", got %s", result)
	}
}

// ========== SET function tests ==========

func TestControlPTZ_Success(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	result, err := controlPTZ(socketKey, "45.0", "-10.0", `{"zoom":2.0}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlPTZ_QuotedArgs(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	// Framework passes quoted values; controlPTZ strips quotes
	result, err := controlPTZ(socketKey, `"45.0"`, `"-10.0"`, `{"zoom":2.0}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

// TestControlPTZ_CustomSpeedIsForwarded proves the caller's speed reaches the
// wire: it's clamped through clampSpeed to VISCA's per-axis ceilings and carried
// in the absolute pan-tilt frame's speed bytes (index 4 = pan, index 5 = tilt).
// Speed 90 clamps to pan 0x18 (24) and tilt 0x14 (20).
func TestControlPTZ_CustomSpeedIsForwarded(t *testing.T) {
	socketKey, frames := fakeVISCADevice(t)
	_, err := controlPTZ(socketKey, "45.0", "-10.0", `{"zoom":2.0,"speed":90}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var abs []byte
	for _, f := range frames() {
		// absolute pan-tilt frame: 81 01 06 02 <panSpeed> <tiltSpeed> ...
		if len(f) >= 6 && f[0] == 0x81 && f[1] == 0x01 && f[2] == 0x06 && f[3] == 0x02 {
			abs = f
		}
	}
	if abs == nil {
		t.Fatal("no absolute pan/tilt frame (81 01 06 02) captured")
	}
	if abs[4] != 0x18 {
		t.Errorf("pan speed byte = 0x%02X, want 0x18 (clamped from 90)", abs[4])
	}
	if abs[5] != 0x14 {
		t.Errorf("tilt speed byte = 0x%02X, want 0x14 (clamped from 90)", abs[5])
	}
}

func TestControlPTZ_NonPositiveSpeedErrors(t *testing.T) {
	// Speed validation happens before any network call, so no device is needed.
	_, err := controlPTZ("admin:x@127.0.0.1:80", "0", "0", `{"zoom":1,"speed":0}`)
	if err == nil {
		t.Fatal("expected error for non-positive speed")
	}
	if !strings.Contains(err.Error(), "speed must be positive") {
		t.Errorf("expected 'speed must be positive' in error, got: %v", err)
	}
}

func TestControlPTZ_InvalidPan(t *testing.T) {
	_, err := controlPTZ("admin:x@127.0.0.1:80", "notanumber", "0", `{"zoom":1}`)
	if err == nil {
		t.Fatal("expected error for invalid pan value")
	}
	if !strings.Contains(err.Error(), "invalid pan") {
		t.Errorf("expected 'invalid pan' in error, got: %v", err)
	}
}

func TestControlPTZ_InvalidTilt(t *testing.T) {
	_, err := controlPTZ("admin:x@127.0.0.1:80", "0", "notanumber", `{"zoom":1}`)
	if err == nil {
		t.Fatal("expected error for invalid tilt value")
	}
	if !strings.Contains(err.Error(), "invalid tilt") {
		t.Errorf("expected 'invalid tilt' in error, got: %v", err)
	}
}

func TestControlPTZ_InvalidZoom(t *testing.T) {
	_, err := controlPTZ("admin:x@127.0.0.1:80", "0", "0", "notanumber")
	if err == nil {
		t.Fatal("expected error for invalid zoom value")
	}
	if !strings.Contains(err.Error(), "invalid zoom") {
		t.Errorf("expected 'invalid zoom' in error, got: %v", err)
	}
}

// TestControlPTZ_DeviceError covers the device-error path: the camera answers
// the pan-tilt command with a VISCA error reply (90 6y FF) instead of a
// completion. controlPTZ must surface that as an error, not report success.
func TestControlPTZ_DeviceError(t *testing.T) {
	socketKey, _ := fakeVISCADeviceWithReply(t, func([]byte) [][]byte {
		return [][]byte{{0x90, 0x60, 0x02, 0xFF}} // VISCA syntax error
	})
	_, err := controlPTZ(socketKey, "45.0", "-10.0", `{"zoom":2.0}`)
	if err == nil {
		t.Fatal("expected error when the device replies with a VISCA error")
	}
}

func TestControlPTZ_PanOutOfRange(t *testing.T) {
	// Pan limit is DOC-CONFIRMED ±162.5°; 200 is beyond the mechanical range.
	// The range check happens before any network call.
	for _, pan := range []string{"200", "-200"} {
		_, err := controlPTZ("admin:x@127.0.0.1:80", pan, "0", `{"zoom":1}`)
		if err == nil {
			t.Fatalf("expected error for out-of-range pan %s", pan)
		}
		if !strings.Contains(err.Error(), "pan out of range") {
			t.Errorf("expected 'pan out of range' for %s, got: %v", pan, err)
		}
	}
}

func TestControlPTZ_TiltOutOfRange(t *testing.T) {
	// Tilt limit is DOC-CONFIRMED -30°..+90°; -45 and 120 are beyond range.
	for _, tilt := range []string{"-45", "120"} {
		_, err := controlPTZ("admin:x@127.0.0.1:80", "0", tilt, `{"zoom":1}`)
		if err == nil {
			t.Fatalf("expected error for out-of-range tilt %s", tilt)
		}
		if !strings.Contains(err.Error(), "tilt out of range") {
			t.Errorf("expected 'tilt out of range' for %s, got: %v", tilt, err)
		}
	}
}

func TestControlPTZ_BoundaryValues(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	// Exact documented boundaries must be accepted: pan ±162.5, tilt -30 and +90.
	cases := []struct{ pan, tilt string }{
		{"162.5", "90"},
		{"-162.5", "-30"},
	}
	for _, c := range cases {
		result, err := controlPTZ(socketKey, c.pan, c.tilt, `{"zoom":1}`)
		if err != nil {
			t.Fatalf("unexpected error at boundary pan=%s tilt=%s: %v", c.pan, c.tilt, err)
		}
		if result != `"ok"` {
			t.Errorf("expected \"ok\" at boundary pan=%s tilt=%s, got %s", c.pan, c.tilt, result)
		}
	}
}

func TestControlPTZHome_Success(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	result, err := controlPTZHome(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

// ========== jog tests ==========

// TestJogPTZ_Up drives an upward jog and asserts the emitted VISCA pan-tilt drive
// frame carries pan=stop, tilt=up (matching viscaJog(viscaPanStop, viscaTiltUp, …)).
func TestJogPTZ_Up(t *testing.T) {
	socketKey, frames := fakeVISCADevice(t)
	result, err := jogPTZ(socketKey, "up", "10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}

	var jog []byte
	for _, f := range frames() {
		// pan-tilt drive frame: 81 01 06 01 <panSpeed> <tiltSpeed> <panDir> <tiltDir> FF
		if len(f) == 9 && f[0] == 0x81 && f[1] == 0x01 && f[2] == 0x06 && f[3] == 0x01 {
			jog = f
		}
	}
	if jog == nil {
		t.Fatal("no pan-tilt drive frame (81 01 06 01) captured")
	}
	if jog[6] != viscaPanStop {
		t.Errorf("pan dir byte = 0x%02X, want stop 0x%02X", jog[6], viscaPanStop)
	}
	if jog[7] != viscaTiltUp {
		t.Errorf("tilt dir byte = 0x%02X, want up 0x%02X", jog[7], viscaTiltUp)
	}
}

func TestJogPTZ_InvalidDirection(t *testing.T) {
	// Direction validation happens before any network call.
	_, err := jogPTZ("admin:x@127.0.0.1:80", "sideways", "10")
	if err == nil {
		t.Fatal("expected error for invalid direction")
	}
	if !strings.Contains(err.Error(), "invalid direction") {
		t.Errorf("expected 'invalid direction' in error, got: %v", err)
	}
}

func TestRecallPreset_Success(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	result, err := recallPreset(socketKey, "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestRecallPreset_QuotedArg(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	result, err := recallPreset(socketKey, `"2"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestSavePreset_Success(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	result, err := savePreset(socketKey, "3", "Podium")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestSavePreset_QuotedArgs(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	result, err := savePreset(socketKey, `"3"`, `"Podium"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlTracking_Enable(t *testing.T) {
	// Tracking rides the CGI plane (auth.cgi session + ptzctrl.cgi command).
	server := mockEC20CGIDevice(t, "admin", "x", 0)
	defer server.Close()
	socketKey := "admin:x@" + strings.TrimPrefix(server.URL, "http://")
	result, err := controlTracking(socketKey, "enable", "presenter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlTracking_Disable(t *testing.T) {
	server := mockEC20CGIDevice(t, "admin", "x", 0)
	defer server.Close()
	socketKey := "admin:x@" + strings.TrimPrefix(server.URL, "http://")
	result, err := controlTracking(socketKey, "disable", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlTracking_InvalidAction(t *testing.T) {
	// The action switch rejects unknown actions before any network call.
	_, err := controlTracking("admin:x@127.0.0.1:80", "toggle", "")
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
	if !strings.Contains(err.Error(), "invalid action") {
		t.Errorf("expected 'invalid action' in error, got: %v", err)
	}
}

func TestControlTracking_QuotedArgs(t *testing.T) {
	server := mockEC20CGIDevice(t, "admin", "x", 0)
	defer server.Close()
	socketKey := "admin:x@" + strings.TrimPrefix(server.URL, "http://")
	result, err := controlTracking(socketKey, `"enable"`, `"presenter"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlTracking_Zone(t *testing.T) {
	// "zone" is a DOC-CONFIRMED tracking mode alongside "presenter"; it validates
	// cleanly and drives the CGI command (mapped to Frame_Track).
	server := mockEC20CGIDevice(t, "admin", "x", 0)
	defer server.Close()
	socketKey := "admin:x@" + strings.TrimPrefix(server.URL, "http://")
	result, err := controlTracking(socketKey, "enable", "zone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlTracking_DefaultsToPresenter(t *testing.T) {
	// An empty mode on enable defaults to presenter (documented default).
	server := mockEC20CGIDevice(t, "admin", "x", 0)
	defer server.Close()
	socketKey := "admin:x@" + strings.TrimPrefix(server.URL, "http://")
	result, err := controlTracking(socketKey, "enable", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlTracking_InvalidMode(t *testing.T) {
	// Mode validation happens before any network call.
	_, err := controlTracking("admin:x@127.0.0.1:80", "enable", "wander")
	if err == nil {
		t.Fatal("expected error for invalid tracking mode")
	}
	if !strings.Contains(err.Error(), "invalid tracking mode") {
		t.Errorf("expected 'invalid tracking mode' in error, got: %v", err)
	}
}

// ========== doDeviceSpecificGet routing tests ==========

func TestDoDeviceSpecificGet_Status(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	result, err := doDeviceSpecificGet(socketKey, "status", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "pan_units") {
		t.Errorf("expected 'pan_units' in result, got: %s", result)
	}
}

func TestDoDeviceSpecificGet_Healthcheck(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	result, err := doDeviceSpecificGet(socketKey, "healthcheck", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"true"` {
		t.Errorf("expected \"true\", got %s", result)
	}
}

func TestDoDeviceSpecificGet_PTZPosition(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	result, err := doDeviceSpecificGet(socketKey, "ptzposition", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "pan_units") {
		t.Errorf("expected 'pan_units' in result, got: %s", result)
	}
}

func TestDoDeviceSpecificGet_Presets(t *testing.T) {
	// presets routes to getPresets, which returns the no-network unsupported constant.
	result, err := doDeviceSpecificGet("admin:x@127.0.0.1:80", "presets", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "unsupported") {
		t.Errorf("expected 'unsupported' in result, got: %s", result)
	}
}

func TestDoDeviceSpecificGet_Preview(t *testing.T) {
	// preview routes to getPreview, which returns the RTSP stream URL (no network).
	result, err := doDeviceSpecificGet("admin:x@127.0.0.1:80", "preview", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "rtsp://") {
		t.Errorf("expected an rtsp:// URL in result, got: %s", result)
	}
}

func TestDoDeviceSpecificGet_Unknown(t *testing.T) {
	_, err := doDeviceSpecificGet("admin:test@127.0.0.1:80", "unknown_setting", "", "")
	if err == nil {
		t.Fatal("expected error for unknown setting")
	}
	if !strings.Contains(err.Error(), "unrecognized setting") {
		t.Errorf("expected 'unrecognized setting' in error, got: %v", err)
	}
}

// ========== doDeviceSpecificSet routing tests ==========

func TestDoDeviceSpecificSet_PTZ(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	// PUT /:addr/ptz/:pan/:tilt  body={"zoom":<num>,"speed":<optional int>}
	result, err := doDeviceSpecificSet(socketKey, "ptz", "45.0", "-10.0", `{"zoom":2.0}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestDoDeviceSpecificSet_PTZHome(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	result, err := doDeviceSpecificSet(socketKey, "ptzhome", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestDoDeviceSpecificSet_Preset(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	// PUT /:addr/preset/:presetId
	result, err := doDeviceSpecificSet(socketKey, "preset", "1", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestDoDeviceSpecificSet_PresetSave(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	// PUT /:addr/presetsave/:presetId  body=name
	result, err := doDeviceSpecificSet(socketKey, "presetsave", "3", "Podium", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestDoDeviceSpecificSet_Tracking(t *testing.T) {
	// tracking routes to controlTracking, which drives the CGI plane.
	// PUT /:addr/tracking/:action  body=mode
	server := mockEC20CGIDevice(t, "admin", "x", 0)
	defer server.Close()
	socketKey := "admin:x@" + strings.TrimPrefix(server.URL, "http://")
	result, err := doDeviceSpecificSet(socketKey, "tracking", "enable", "presenter", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestDoDeviceSpecificSet_Jog(t *testing.T) {
	socketKey, _ := fakeVISCADevice(t)
	// PUT /:addr/jog/:dir/:speed
	result, err := doDeviceSpecificSet(socketKey, "jog", "up", "10", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestDoDeviceSpecificSet_Unknown(t *testing.T) {
	_, err := doDeviceSpecificSet("admin:test@127.0.0.1:80", "unknown_setting", "", "", "")
	if err == nil {
		t.Fatal("expected error for unknown setting")
	}
	if !strings.Contains(err.Error(), "unrecognized setting") {
		t.Errorf("expected 'unrecognized setting' in error, got: %v", err)
	}
}

// ========== validatePresetID tests ==========

func TestValidatePresetID_Valid(t *testing.T) {
	tests := []struct {
		name     string
		presetID string
	}{
		{"zero (valid - preset 0 exists per EC20 docs)", "0"},
		{"minimum nonzero", "1"},
		{"middle", "128"},
		{"maximum", "255"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePresetID(tt.presetID)
			if err != nil {
				t.Errorf("expected no error for %s, got: %v", tt.presetID, err)
			}
		})
	}
}

func TestValidatePresetID_Invalid(t *testing.T) {
	tests := []struct {
		name     string
		presetID string
	}{
		{"empty string", ""},
		{"negative", "-1"},
		{"too large", "256"},
		{"non-numeric", "abc"},
		{"decimal", "1.5"},
		{"hex", "0xFF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePresetID(tt.presetID)
			if err == nil {
				t.Errorf("expected error for %s, got nil", tt.presetID)
			}
		})
	}
}

func TestRecallPreset_InvalidPresetID(t *testing.T) {
	// validatePresetID rejects out-of-range IDs before any network call.
	_, err := recallPreset("admin:x@127.0.0.1:80", "999")
	if err == nil {
		t.Fatal("expected error for out-of-range presetID")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("expected 'out of range' in error, got: %v", err)
	}
}

func TestSavePreset_InvalidPresetID(t *testing.T) {
	// validatePresetID rejects non-numeric IDs before any network call.
	_, err := savePreset("admin:x@127.0.0.1:80", "abc", "TestPreset")
	if err == nil {
		t.Fatal("expected error for non-numeric presetID")
	}
	if !strings.Contains(err.Error(), "must be numeric") {
		t.Errorf("expected 'must be numeric' in error, got: %v", err)
	}
}

func TestSavePreset_NameTooLong(t *testing.T) {
	// The name-length guard fires before any network call.
	longName := strings.Repeat("a", 65)
	_, err := savePreset("admin:x@127.0.0.1:80", "1", longName)
	if err == nil {
		t.Fatal("expected error for name longer than 64 chars")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("expected 'too long' in error, got: %v", err)
	}
}

// ========== digest auth tests ==========

// md5hex is a test helper mirroring the real EC20's MD5 digest computation.
// (Also used by cgiauth_test.go.)
func md5hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// parseDigestHeader parses an "Authorization: Digest k=v, ..." header into a map.
// Test-only; our controlled values never contain commas inside quoted fields.
func parseDigestHeader(h string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(strings.TrimPrefix(h, "Digest "), ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		out[strings.TrimSpace(kv[0])] = strings.Trim(strings.TrimSpace(kv[1]), `"`)
	}
	return out
}

// TestEC20DigestAuthHeader is a self-contained unit test of the driver's RFC 2617
// Digest response math (used by the CGI transport layer): given a known MD5/qop=auth
// challenge, it independently recomputes the expected response from the nc/cnonce the
// generated header carries and asserts equality.
func TestEC20DigestAuthHeader(t *testing.T) {
	const (
		realm    = "r"
		nonce    = "n"
		username = "admin"
		password = "pass"
		method   = "GET"
		uri      = "/x"
	)
	challenge := `Digest realm="r", nonce="n", qop="auth", algorithm=MD5`

	header, err := ec20DigestAuthHeader(challenge, method, uri, username, password)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p := parseDigestHeader(header)
	if p["username"] != username {
		t.Errorf("username = %q, want %q", p["username"], username)
	}
	if p["realm"] != realm || p["nonce"] != nonce || p["uri"] != uri {
		t.Errorf("unexpected header fields: %+v", p)
	}

	ha1 := md5hex(username + ":" + realm + ":" + password)
	ha2 := md5hex(method + ":" + uri)
	want := md5hex(strings.Join([]string{ha1, nonce, p["nc"], p["cnonce"], "auth", ha2}, ":"))
	if p["response"] != want {
		t.Errorf("response = %q, want %q", p["response"], want)
	}
}
