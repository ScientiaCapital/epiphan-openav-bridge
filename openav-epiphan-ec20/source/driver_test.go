package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockEC20API creates a test server that simulates EC20 REST API responses.
func mockEC20API(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// checkAuth validates Basic Auth on every handler.
	checkAuth := func(w http.ResponseWriter, r *http.Request) bool {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "testpass" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return false
		}
		return true
	}

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"result": map[string]interface{}{
				"model":    "EC20",
				"firmware": "1.0.0",
				"tracking": "enabled",
				"position": map[string]interface{}{
					"pan":  0,
					"tilt": 0,
					"zoom": 1,
				},
			},
		})
	})

	mux.HandleFunc("/api/ptz/position", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"result": map[string]interface{}{
				"pan":  45.0,
				"tilt": -10.0,
				"zoom": 2.0,
			},
		})
	})

	mux.HandleFunc("/api/ptz/pan", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			http.Error(w, "Expected application/json", http.StatusBadRequest)
			return
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if _, ok := body["degrees"]; !ok {
			http.Error(w, "Missing degrees", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	})

	mux.HandleFunc("/api/ptz/tilt", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			http.Error(w, "Expected application/json", http.StatusBadRequest)
			return
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if _, ok := body["degrees"]; !ok {
			http.Error(w, "Missing degrees", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	})

	mux.HandleFunc("/api/ptz/zoom", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			http.Error(w, "Expected application/json", http.StatusBadRequest)
			return
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if _, ok := body["level"]; !ok {
			http.Error(w, "Missing level", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	})

	mux.HandleFunc("/api/ptz/home", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	})

	mux.HandleFunc("/api/ptz/presets", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"result": []interface{}{
				map[string]interface{}{"id": "1", "name": "Center"},
				map[string]interface{}{"id": "2", "name": "Wide"},
			},
		})
	})

	mux.HandleFunc("/api/ptz/preset/goto", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			http.Error(w, "Expected application/json", http.StatusBadRequest)
			return
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if _, ok := body["preset_id"]; !ok {
			http.Error(w, "Missing preset_id", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	})

	mux.HandleFunc("/api/ptz/preset/save", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			http.Error(w, "Expected application/json", http.StatusBadRequest)
			return
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if _, ok := body["preset_id"]; !ok {
			http.Error(w, "Missing preset_id", http.StatusBadRequest)
			return
		}
		if _, ok := body["name"]; !ok {
			http.Error(w, "Missing name", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	})

	mux.HandleFunc("/api/tracking/enable", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			http.Error(w, "Expected application/json", http.StatusBadRequest)
			return
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if _, ok := body["mode"]; !ok {
			http.Error(w, "Missing mode", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	})

	mux.HandleFunc("/api/tracking/disable", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	})

	mux.HandleFunc("/api/preview", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0})
	})

	return httptest.NewServer(mux)
}

// socketKeyFromServer builds a socketKey from the test server URL.
func socketKeyFromServer(server *httptest.Server) string {
	addr := strings.TrimPrefix(server.URL, "http://")
	return "admin:testpass@" + addr
}

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

// ========== ec20APIGet tests ==========

func TestEC20APIGet_Success(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := ec20APIGet(socketKey, ec20EndpointStatus)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status ok, got %v", result["status"])
	}
}

func TestEC20APIGet_Unauthorized(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:wrongpass@" + addr

	_, err := ec20APIGet(socketKey, ec20EndpointStatus)
	if err == nil {
		t.Fatal("expected error for bad credentials")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected '401' in error, got: %v", err)
	}
}

func TestEC20APIGet_ConnectionRefused(t *testing.T) {
	_, err := ec20APIGet("admin:testpass@127.0.0.1:1", ec20EndpointStatus)
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestEC20APIGet_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	_, err := ec20APIGet(socketKey, "/api/status")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "parsing JSON") {
		t.Errorf("expected 'parsing JSON' in error, got: %v", err)
	}
}

func TestEC20APIGet_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": "Device busy",
		})
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	_, err := ec20APIGet(socketKey, "/api/status")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("expected 'API error' in error, got: %v", err)
	}
}

func TestEC20APIGet_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	_, err := ec20APIGet(socketKey, "/api/status")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected '404' in error, got: %v", err)
	}
}

// ========== ec20APIPost tests ==========

func TestEC20APIPost_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	err := ec20APIPost(socketKey, "/api/ptz/home")
	if err != nil {
		t.Fatalf("non-JSON 200 response should not error, got: %v", err)
	}
}

func TestEC20APIPost_Unauthorized(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:wrongpass@" + addr

	err := ec20APIPost(socketKey, ec20EndpointHome)
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected '401' in error, got: %v", err)
	}
}

func TestEC20APIPost_ConnectionRefused(t *testing.T) {
	err := ec20APIPost("admin:testpass@127.0.0.1:1", ec20EndpointHome)
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestEC20APIPost_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	err := ec20APIPost(socketKey, "/api/ptz/home")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected '500' in error, got: %v", err)
	}
}

func TestEC20APIPost_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": "Camera busy",
		})
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	err := ec20APIPost(socketKey, "/api/ptz/home")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("expected 'API error' in error, got: %v", err)
	}
}

// ========== ec20APIPostJSON tests ==========

func TestEC20APIPostJSON_Success(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := ec20APIPostJSON(socketKey, ec20EndpointPan, map[string]interface{}{
		"degrees": 45.0,
		"speed":   50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status ok, got %v", result["status"])
	}
}

func TestEC20APIPostJSON_Unauthorized(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:wrongpass@" + addr

	_, err := ec20APIPostJSON(socketKey, ec20EndpointPan, map[string]interface{}{
		"degrees": 45.0,
		"speed":   50,
	})
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected '401' in error, got: %v", err)
	}
}

func TestEC20APIPostJSON_ConnectionRefused(t *testing.T) {
	_, err := ec20APIPostJSON("admin:testpass@127.0.0.1:1", ec20EndpointPan, map[string]interface{}{
		"degrees": 45.0,
	})
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestEC20APIPostJSON_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": "Invalid parameter",
		})
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	_, err := ec20APIPostJSON(socketKey, "/api/ptz/pan", map[string]interface{}{
		"degrees": 999,
	})
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("expected 'API error' in error, got: %v", err)
	}
}

func TestEC20APIPostJSON_NonJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	result, err := ec20APIPostJSON(socketKey, "/api/ptz/pan", map[string]interface{}{
		"degrees": 45.0,
	})
	if err != nil {
		t.Fatalf("non-JSON 200 response should not error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil empty map for non-JSON response")
	}
}

func TestEC20APIPostJSON_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	_, err := ec20APIPostJSON(socketKey, "/api/ptz/pan", map[string]interface{}{
		"degrees": 45.0,
	})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected '500' in error, got: %v", err)
	}
}

// ========== ec20APIGetRaw tests ==========

func TestEC20APIGetRaw_Success(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	data, err := ec20APIGetRaw(socketKey, ec20EndpointPreview)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	if len(data) != len(expected) {
		t.Fatalf("expected %d bytes, got %d", len(expected), len(data))
	}
	for i, b := range expected {
		if data[i] != b {
			t.Errorf("byte %d: expected 0x%02X, got 0x%02X", i, b, data[i])
		}
	}
}

func TestEC20APIGetRaw_Unauthorized(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:wrongpass@" + addr

	_, err := ec20APIGetRaw(socketKey, ec20EndpointPreview)
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected '401' in error, got: %v", err)
	}
}

func TestEC20APIGetRaw_ConnectionRefused(t *testing.T) {
	_, err := ec20APIGetRaw("admin:testpass@127.0.0.1:1", ec20EndpointPreview)
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestEC20APIGetRaw_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	_, err := ec20APIGetRaw(socketKey, "/api/preview")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected '404' in error, got: %v", err)
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

func TestGetPreview_Success(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := getPreview(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Result is a JSON-encoded base64 string (quoted)
	var b64str string
	err = json.Unmarshal([]byte(result), &b64str)
	if err != nil {
		t.Fatalf("result is not a valid JSON string: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(b64str)
	if err != nil {
		t.Fatalf("result is not valid base64: %v", err)
	}

	expected := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	if len(decoded) != len(expected) {
		t.Fatalf("expected %d decoded bytes, got %d", len(expected), len(decoded))
	}
	for i, b := range expected {
		if decoded[i] != b {
			t.Errorf("byte %d: expected 0x%02X, got 0x%02X", i, b, decoded[i])
		}
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
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := controlPTZ(socketKey, "0", "0", `{"zoom":1,"speed":0}`)
	if err == nil {
		t.Fatal("expected error for non-positive speed")
	}
	if !strings.Contains(err.Error(), "speed must be positive") {
		t.Errorf("expected 'speed must be positive' in error, got: %v", err)
	}
}

func TestControlPTZ_InvalidPan(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := controlPTZ(socketKey, "notanumber", "0", `{"zoom":1}`)
	if err == nil {
		t.Fatal("expected error for invalid pan value")
	}
	if !strings.Contains(err.Error(), "invalid pan") {
		t.Errorf("expected 'invalid pan' in error, got: %v", err)
	}
}

func TestControlPTZ_InvalidTilt(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := controlPTZ(socketKey, "0", "notanumber", `{"zoom":1}`)
	if err == nil {
		t.Fatal("expected error for invalid tilt value")
	}
	if !strings.Contains(err.Error(), "invalid tilt") {
		t.Errorf("expected 'invalid tilt' in error, got: %v", err)
	}
}

func TestControlPTZ_InvalidZoom(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := controlPTZ(socketKey, "0", "0", "notanumber")
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
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	// Pan limit is DOC-CONFIRMED ±162.5°; 200 is beyond the mechanical range.
	for _, pan := range []string{"200", "-200"} {
		_, err := controlPTZ(socketKey, pan, "0", `{"zoom":1}`)
		if err == nil {
			t.Fatalf("expected error for out-of-range pan %s", pan)
		}
		if !strings.Contains(err.Error(), "pan out of range") {
			t.Errorf("expected 'pan out of range' for %s, got: %v", pan, err)
		}
	}
}

func TestControlPTZ_TiltOutOfRange(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	// Tilt limit is DOC-CONFIRMED -30°..+90°; -45 and 120 are beyond range.
	for _, tilt := range []string{"-45", "120"} {
		_, err := controlPTZ(socketKey, "0", tilt, `{"zoom":1}`)
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
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := controlTracking(socketKey, "toggle", "")
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
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := controlTracking(socketKey, "enable", "wander")
	if err == nil {
		t.Fatal("expected error for invalid tracking mode")
	}
	if !strings.Contains(err.Error(), "invalid tracking mode") {
		t.Errorf("expected 'invalid tracking mode' in error, got: %v", err)
	}
}

// ========== Mock server validation tests ==========

func TestMockEC20API_PanValidatesBody(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	// POST /api/ptz/pan without Content-Type should fail
	addr := strings.TrimPrefix(server.URL, "http://")
	req, _ := http.NewRequest("POST", "http://"+addr+"/api/ptz/pan", nil)
	req.SetBasicAuth("admin", "testpass")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 for POST without Content-Type")
	}
}

func TestMockEC20API_PresetGotoValidatesBody(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	// POST /api/ptz/preset/goto with missing preset_id should fail
	addr := strings.TrimPrefix(server.URL, "http://")
	body := strings.NewReader(`{"other":"value"}`)
	req, _ := http.NewRequest("POST", "http://"+addr+"/api/ptz/preset/goto", body)
	req.SetBasicAuth("admin", "testpass")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 for POST missing preset_id")
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
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := doDeviceSpecificGet(socketKey, "preview", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Preview returns a JSON-quoted base64 string
	if len(result) == 0 {
		t.Error("expected non-empty preview result")
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
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := recallPreset(socketKey, "999")
	if err == nil {
		t.Fatal("expected error for out-of-range presetID")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("expected 'out of range' in error, got: %v", err)
	}
}

func TestSavePreset_InvalidPresetID(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := savePreset(socketKey, "abc", "TestPreset")
	if err == nil {
		t.Fatal("expected error for non-numeric presetID")
	}
	if !strings.Contains(err.Error(), "must be numeric") {
		t.Errorf("expected 'must be numeric' in error, got: %v", err)
	}
}

func TestSavePreset_NameTooLong(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	longName := strings.Repeat("a", 65)
	_, err := savePreset(socketKey, "1", longName)
	if err == nil {
		t.Fatal("expected error for name longer than 64 chars")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("expected 'too long' in error, got: %v", err)
	}
}

// md5hex is a test helper mirroring the real EC20's MD5 digest computation.
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

// mockEC20DigestAPI simulates the real EC20: it demands HTTP Digest auth
// (realm="", MD5, qop="auth" — exactly what lighttpd/1.4.75 on the device sends)
// and rejects Basic auth. On a valid Digest response it serves /api/status.
// It recomputes the expected response from the client's own nonce/nc/cnonce so
// the test genuinely exercises our client's digest math, not a stub.
func mockEC20DigestAPI(t *testing.T) *httptest.Server {
	t.Helper()
	const realm = ""
	const nonce = "6a5bb5d7:testnonce"

	challenge := func(w http.ResponseWriter) {
		w.Header().Set("WWW-Authenticate",
			fmt.Sprintf(`Digest realm="%s", charset="UTF-8", algorithm=MD5, nonce="%s", qop="auth"`, realm, nonce))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if !strings.HasPrefix(authz, "Digest ") {
			challenge(w) // Basic auth (or none) is rejected — this is the whole point
			return
		}
		p := parseDigestHeader(authz)
		ha1 := md5hex("admin:" + realm + ":testpass")
		ha2 := md5hex(r.Method + ":" + p["uri"])
		expected := md5hex(strings.Join([]string{ha1, p["nonce"], p["nc"], p["cnonce"], p["qop"], ha2}, ":"))
		if p["username"] != "admin" || p["response"] != expected {
			challenge(w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"result": map[string]interface{}{"model": "EC20"},
		})
	}
	return httptest.NewServer(http.HandlerFunc(handler))
}

// TestEC20APIGet_DigestAuth_Success proves the driver can authenticate against a
// device that requires HTTP Digest (the real EC20 does; Basic auth is rejected).
func TestEC20APIGet_DigestAuth_Success(t *testing.T) {
	server := mockEC20DigestAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := ec20APIGet(socketKey, ec20EndpointStatus)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status ok, got %v", result["status"])
	}
}

// TestEC20APIGet_DigestAuth_WrongPassword ensures a bad password still fails
// under the digest handshake (no silent success).
func TestEC20APIGet_DigestAuth_WrongPassword(t *testing.T) {
	server := mockEC20DigestAPI(t)
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:wrongpass@" + addr

	_, err := ec20APIGet(socketKey, ec20EndpointStatus)
	if err == nil {
		t.Fatal("expected error for wrong password under digest auth")
	}
}
