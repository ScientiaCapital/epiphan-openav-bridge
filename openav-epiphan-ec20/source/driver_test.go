package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := getCameraStatus(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if parsed["model"] != "EC20" {
		t.Errorf("expected model EC20, got %v", parsed["model"])
	}
	if parsed["firmware"] != "1.0.0" {
		t.Errorf("expected firmware 1.0.0, got %v", parsed["firmware"])
	}
}

func TestGetCameraStatus_NoResult(t *testing.T) {
	// When no "result" key exists, getCameraStatus returns the top-level data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "ok",
			"model":    "EC20",
			"firmware": "1.0.0",
		})
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:testpass@" + addr

	result, err := getCameraStatus(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// Should contain top-level fields since there is no "result" key
	if parsed["model"] != "EC20" {
		t.Errorf("expected model EC20, got %v", parsed["model"])
	}
}

func TestGetCameraStatus_Unauthorized(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:wrongpass@" + addr

	_, err := getCameraStatus(socketKey)
	if err == nil {
		t.Fatal("expected error for bad credentials")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected '401' in error, got: %v", err)
	}
}

func TestGetPTZPosition_Success(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := getPTZPosition(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if parsed["pan"] != 45.0 {
		t.Errorf("expected pan 45.0, got %v", parsed["pan"])
	}
	if parsed["tilt"] != -10.0 {
		t.Errorf("expected tilt -10.0, got %v", parsed["tilt"])
	}
	if parsed["zoom"] != 2.0 {
		t.Errorf("expected zoom 2.0, got %v", parsed["zoom"])
	}
}

func TestGetPresets_Success(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := getPresets(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed []interface{}
	err = json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("result is not valid JSON array: %v", err)
	}

	if len(parsed) != 2 {
		t.Fatalf("expected 2 presets, got %d", len(parsed))
	}

	preset := parsed[0].(map[string]interface{})
	if preset["name"] != "Center" {
		t.Errorf("expected first preset name Center, got %v", preset["name"])
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
	server := mockEC20API(t)
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

func TestControlPTZ_Success(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := controlPTZ(socketKey, "45.0", "-10.0", "2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlPTZ_QuotedArgs(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	// Framework passes quoted values; controlPTZ strips quotes
	result, err := controlPTZ(socketKey, `"45.0"`, `"-10.0"`, `"2.0"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlPTZ_InvalidPan(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	_, err := controlPTZ(socketKey, "notanumber", "0", "1")
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
	_, err := controlPTZ(socketKey, "0", "notanumber", "1")
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

func TestControlPTZ_Unauthorized(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:wrongpass@" + addr

	_, err := controlPTZ(socketKey, "0", "0", "1")
	if err == nil {
		t.Fatal("expected error for bad credentials")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected '401' in error, got: %v", err)
	}
}

func TestControlPTZHome_Success(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := controlPTZHome(socketKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlPTZHome_Unauthorized(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	socketKey := "admin:wrongpass@" + addr

	_, err := controlPTZHome(socketKey)
	if err == nil {
		t.Fatal("expected error for bad credentials")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected '401' in error, got: %v", err)
	}
}

func TestRecallPreset_Success(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := recallPreset(socketKey, "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestRecallPreset_QuotedArg(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := recallPreset(socketKey, `"2"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestSavePreset_Success(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := savePreset(socketKey, "3", "Podium")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestSavePreset_QuotedArgs(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := savePreset(socketKey, `"3"`, `"Podium"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlTracking_Enable(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := controlTracking(socketKey, "enable", "presenter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestControlTracking_Disable(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
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
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := controlTracking(socketKey, `"enable"`, `"presenter"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
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
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := doDeviceSpecificGet(socketKey, "status", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "EC20") {
		t.Errorf("expected EC20 in result, got: %s", result)
	}
}

func TestDoDeviceSpecificGet_Healthcheck(t *testing.T) {
	server := mockEC20API(t)
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

func TestDoDeviceSpecificGet_PTZPosition(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := doDeviceSpecificGet(socketKey, "ptzposition", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "pan") {
		t.Errorf("expected 'pan' in result, got: %s", result)
	}
}

func TestDoDeviceSpecificGet_Presets(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := doDeviceSpecificGet(socketKey, "presets", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Center") {
		t.Errorf("expected 'Center' in result, got: %s", result)
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
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	// PUT /:addr/ptz/:pan/:tilt  body=zoom
	result, err := doDeviceSpecificSet(socketKey, "ptz", "45.0", "-10.0", "2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestDoDeviceSpecificSet_PTZHome(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	result, err := doDeviceSpecificSet(socketKey, "ptzhome", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"ok"` {
		t.Errorf("expected \"ok\", got %s", result)
	}
}

func TestDoDeviceSpecificSet_Preset(t *testing.T) {
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
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
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
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
	server := mockEC20API(t)
	defer server.Close()

	socketKey := socketKeyFromServer(server)
	// PUT /:addr/tracking/:action  body=mode
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
		{"minimum", "1"},
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
		{"zero", "0"},
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
