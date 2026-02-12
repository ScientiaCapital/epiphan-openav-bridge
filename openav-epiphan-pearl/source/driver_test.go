package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockPearlAPI creates a test server that simulates Pearl REST API v2.0 responses.
func mockPearlAPI(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v2.0/device", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "testpass" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"result": map[string]interface{}{
				"name":     "Pearl Mini",
				"model":    "Pearl Mini",
				"serial":   "SN12345",
				"firmware": "4.24.1",
			},
		})
	})

	mux.HandleFunc("/api/v2.0/storages", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "testpass" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"result": []interface{}{
				map[string]interface{}{
					"id":          "internal",
					"total_bytes": 500000000000,
					"free_bytes":  250000000000,
				},
			},
		})
	})

	mux.HandleFunc("/api/v2.0/recorders/status", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "testpass" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"result": []interface{}{
				map[string]interface{}{
					"id":    "recorder-1",
					"state": "stopped",
				},
			},
		})
	})

	mux.HandleFunc("/api/v2.0/channels", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "testpass" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"result": []interface{}{
				map[string]interface{}{
					"id":   "channel-1",
					"name": "Main Channel",
				},
			},
		})
	})

	mux.HandleFunc("/api/v2.0/recorders/control/start", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "testpass" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	})

	mux.HandleFunc("/api/v2.0/recorders/control/stop", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "testpass" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	})

	mux.HandleFunc("/api/v2.0/singletouch/control/start", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "testpass" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	})

	mux.HandleFunc("/api/v2.0/singletouch/control/stop", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "testpass" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	})

	mux.HandleFunc("/api/v2.0/channels/channel-1/publishers/control/start", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "testpass" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	})

	mux.HandleFunc("/api/v2.0/channels/channel-1/publishers/control/stop", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "testpass" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	})

	return httptest.NewServer(mux)
}

// socketKeyFromServer builds a socketKey from the test server URL.
func socketKeyFromServer(server *httptest.Server) string {
	addr := strings.TrimPrefix(server.URL, "http://")
	return "admin:testpass@" + addr
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

func TestGetDeviceStatus_Success(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := getDeviceStatus(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	device, ok := parsed["device"].(map[string]interface{})
	if !ok {
		t.Fatal("expected device key in status response")
	}
	if device["model"] != "Pearl Mini" {
		t.Errorf("expected model Pearl Mini, got %v", device["model"])
	}

	if _, ok := parsed["storages"]; !ok {
		t.Error("expected storages key in status response")
	}
}

func TestGetRecordingStatus_Success(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := getRecordingStatus(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed []interface{}
	err = json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("result is not valid JSON array: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 recorder, got %d", len(parsed))
	}

	recorder := parsed[0].(map[string]interface{})
	if recorder["state"] != "stopped" {
		t.Errorf("expected state stopped, got %v", recorder["state"])
	}
}

func TestGetStorages_Success(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := getStorages(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed []interface{}
	err = json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("result is not valid JSON array: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 storage, got %d", len(parsed))
	}
}

func TestGetChannels_Success(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := getChannels(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed []interface{}
	err = json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("result is not valid JSON array: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(parsed))
	}

	channel := parsed[0].(map[string]interface{})
	if channel["name"] != "Main Channel" {
		t.Errorf("expected name Main Channel, got %v", channel["name"])
	}
}

func TestHealthCheck_Success(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
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

func TestControlRecording_Start(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := controlRecording(socketKey, `"start"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlRecording_Stop(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := controlRecording(socketKey, `"stop"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlRecording_InvalidAction(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := controlRecording(socketKey, `"pause"`)
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
	if !strings.Contains(err.Error(), "invalid action") {
		t.Errorf("expected 'invalid action' in error, got: %v", err)
	}
}

func TestControlSingleTouch_Start(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := controlSingleTouch(socketKey, `"start"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlSingleTouch_Stop(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := controlSingleTouch(socketKey, `"stop"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlStreaming_Start(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := controlStreaming(socketKey, "channel-1", `"start"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlStreaming_Stop(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := controlStreaming(socketKey, "channel-1", `"stop"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

// ========== Error handling tests ==========

func TestGetDeviceStatus_Unauthorized(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:wrongpass@" + addr

	_, err := getDeviceStatus(socketKey)
	if err == nil {
		t.Fatal("expected error for bad credentials")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected 401 in error, got: %v", err)
	}
}

func TestGetDeviceStatus_ConnectionRefused(t *testing.T) {
	_, err := getDeviceStatus("admin:testpass@127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestControlRecording_Unauthorized(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:wrongpass@" + addr

	_, err := controlRecording(socketKey, `"start"`)
	if err == nil {
		t.Fatal("expected error for bad credentials")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected 401 in error, got: %v", err)
	}
}

func TestPearlAPIGet_APIError(t *testing.T) {
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

	_, err := pearlAPIGet(socketKey, "/device")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("expected 'API error' in error, got: %v", err)
	}
}

func TestPearlAPIGet_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	_, err := pearlAPIGet(socketKey, "/device")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "parsing JSON") {
		t.Errorf("expected 'parsing JSON' in error, got: %v", err)
	}
}

func TestPearlAPIPost_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	err := pearlAPIPost(socketKey, "/recorders/control/start")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected '500' in error, got: %v", err)
	}
}

func TestPearlAPIPost_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": "Recording already in progress",
		})
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	err := pearlAPIPost(socketKey, "/recorders/control/start")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("expected 'API error' in error, got: %v", err)
	}
}

func TestPearlAPIPost_NonJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	err := pearlAPIPost(socketKey, "/recorders/control/start")
	if err != nil {
		t.Fatalf("non-JSON 200 response should not error, got: %v", err)
	}
}

func TestPearlAPIPost_ConnectionRefused(t *testing.T) {
	err := pearlAPIPost("admin:testpass@127.0.0.1:1", "/recorders/control/start")
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestPearlAPIPost_Unauthorized(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:wrongpass@" + addr

	err := pearlAPIPost(socketKey, "/recorders/control/start")
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected '401' in error, got: %v", err)
	}
}

func TestGetRecordingStatus_NoResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	result, err := getRecordingStatus(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"unknown"` {
		t.Errorf("expected \"unknown\", got %s", result)
	}
}

func TestGetStorages_NoResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	result, err := getStorages(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"unknown"` {
		t.Errorf("expected \"unknown\", got %s", result)
	}
}

func TestGetChannels_NoResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	result, err := getChannels(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"unknown"` {
		t.Errorf("expected \"unknown\", got %s", result)
	}
}

func TestControlStreaming_InvalidAction(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := controlStreaming(socketKey, "channel-1", `"invalid"`)
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
	if !strings.Contains(err.Error(), "invalid action") {
		t.Errorf("expected 'invalid action' in error, got: %v", err)
	}
}

func TestControlSingleTouch_InvalidAction(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := controlSingleTouch(socketKey, `"invalid"`)
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
	if !strings.Contains(err.Error(), "invalid action") {
		t.Errorf("expected 'invalid action' in error, got: %v", err)
	}
}

// ========== microservice.go routing tests ==========

func TestDoDeviceSpecificGet_Status(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := doDeviceSpecificGet(socketKey, "status", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Pearl Mini") {
		t.Errorf("expected Pearl Mini in result, got: %s", result)
	}
}

func TestDoDeviceSpecificGet_RecordingStatus(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := doDeviceSpecificGet(socketKey, "recordingstatus", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoDeviceSpecificGet_Healthcheck(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := doDeviceSpecificGet(socketKey, "healthcheck", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"true"` {
		t.Errorf("expected \"true\", got %s", result)
	}
}

func TestDoDeviceSpecificGet_Unknown(t *testing.T) {
	_, err := doDeviceSpecificGet("admin:test@127.0.0.1:80", "unknown_setting", "", "")
	if err == nil {
		t.Fatal("expected error for unknown setting")
	}
}

func TestDoDeviceSpecificSet_Recording(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := doDeviceSpecificSet(socketKey, "recording", `"start"`, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestDoDeviceSpecificSet_Streaming(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := doDeviceSpecificSet(socketKey, "streaming", "channel-1", `"start"`, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestDoDeviceSpecificSet_SingleTouch(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := doDeviceSpecificSet(socketKey, "singletouch", `"start"`, "", "")
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
}

func TestDoDeviceSpecificGet_Storages(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := doDeviceSpecificGet(socketKey, "storages", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoDeviceSpecificGet_Channels(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := doDeviceSpecificGet(socketKey, "channels", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPearlAPIGet_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	_, err := pearlAPIGet(socketKey, "/device")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected '404' in error, got: %v", err)
	}
}

// ========== validateChannelID tests ==========

func TestValidateChannelID_Valid(t *testing.T) {
	tests := []struct {
		name      string
		channelID string
	}{
		{"single digit", "1"},
		{"multiple digits", "123"},
		{"with hyphen", "channel-1"},
		{"with underscore", "my_channel"},
		{"mixed case", "Channel123"},
		{"max length", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateChannelID(tt.channelID)
			if err != nil {
				t.Errorf("expected no error for %s, got: %v", tt.channelID, err)
			}
		})
	}
}

func TestValidateChannelID_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		channelID string
	}{
		{"empty string", ""},
		{"path traversal", "../admin"},
		{"slash in path", "channels/1"},
		{"backslash", "a\\b"},
		{"url encoded slash", "%2F"},
		{"question mark", "a?b"},
		{"hash", "a#b"},
		{"ampersand", "a&b"},
		{"space", "a b"},
		{"null byte", "channel\x00admin"},
		{"too long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateChannelID(tt.channelID)
			if err == nil {
				t.Errorf("expected error for %s, got nil", tt.channelID)
			}
		})
	}
}

func TestControlStreaming_InvalidChannelID(t *testing.T) {
	server := mockPearlAPI(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := controlStreaming(socketKey, "../admin", `"start"`)
	if err == nil {
		t.Fatal("expected error for path traversal in channelID")
	}
	if !strings.Contains(err.Error(), "invalid characters") {
		t.Errorf("expected 'invalid characters' in error, got: %v", err)
	}
}
