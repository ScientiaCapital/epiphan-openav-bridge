package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// parseAuthenticateHeaderTest parses the client's authenticate header
// (username="..",nonce="..",nc=1,cnonce="..",uri="..",response="..") for the mock.
func parseAuthenticateHeaderTest(h string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(h, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			out[strings.TrimSpace(kv[0])] = strings.Trim(strings.TrimSpace(kv[1]), `"`)
		}
	}
	return out
}

// mockEC20AuthCGI simulates /cgi-bin/auth.cgi: a base64-dot challenge, then it
// recomputes the custom digest response from the client's own nc/cnonce and,
// on a match, grants jwt + authorization headers. This exercises OUR handshake
// math, not a stub.
func mockEC20AuthCGI(t *testing.T, username, password string) *httptest.Server {
	t.Helper()
	const realm, nonce, qop = "EC20", "6a5c-server-nonce", "auth"
	challenge := base64.StdEncoding.EncodeToString([]byte(realm)) + "." +
		base64.StdEncoding.EncodeToString([]byte(nonce)) + "." +
		base64.StdEncoding.EncodeToString([]byte(qop))

	mux := http.NewServeMux()
	mux.HandleFunc("/cgi-bin/auth.cgi", func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("authenticate")
		if authz == "" {
			w.Header().Set("WWW-Authenticate", challenge)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		p := parseAuthenticateHeaderTest(authz)
		// Recompute expected response: md5(md5(user:realm:pass):nonce:nc:cnonce:md5("GET:/auth.cgi")).
		ha1 := md5hex(username + ":" + realm + ":" + password)
		ha2 := md5hex("GET:" + p["uri"])
		expected := md5hex(strings.Join([]string{ha1, p["nonce"], p["nc"], p["cnonce"], ha2}, ":"))
		if p["username"] != username || p["response"] != expected {
			w.WriteHeader(201) // bad credentials, per device semantics
			return
		}
		w.Header().Set("jwt", "JWT_TOKEN_123")
		w.Header().Set("authorization", "AUTH_TOKEN_456")
		w.WriteHeader(http.StatusOK)
	})
	return httptest.NewServer(mux)
}

func TestEC20Login_Success(t *testing.T) {
	server := mockEC20AuthCGI(t, "admin", "admin")
	defer server.Close()

	sess, err := ec20Login(&http.Client{}, server.URL, "admin", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.jwt != "JWT_TOKEN_123" {
		t.Errorf("jwt = %q, want JWT_TOKEN_123", sess.jwt)
	}
	if sess.authorization != "AUTH_TOKEN_456" {
		t.Errorf("authorization = %q, want AUTH_TOKEN_456", sess.authorization)
	}
}

func TestEC20Login_WrongPassword(t *testing.T) {
	server := mockEC20AuthCGI(t, "admin", "admin")
	defer server.Close()

	_, err := ec20Login(&http.Client{}, server.URL, "admin", "wrongpass")
	if err == nil {
		t.Fatal("expected error for wrong password under auth.cgi handshake")
	}
}

func TestParseCGIChallenge(t *testing.T) {
	realm := base64.StdEncoding.EncodeToString([]byte("EC20"))
	nonce := base64.StdEncoding.EncodeToString([]byte("nonce123"))
	qop := base64.StdEncoding.EncodeToString([]byte("auth"))

	r, n, q, err := parseCGIChallenge(realm + "." + nonce + "." + qop)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != "EC20" || n != "nonce123" || q != "auth" {
		t.Errorf("parseCGIChallenge = (%q,%q,%q), want (EC20,nonce123,auth)", r, n, q)
	}
}

// TestEC20CGIResponse_KnownVector pins the exact hash formula (raw-int nc, no qop).
func TestEC20CGIResponse_KnownVector(t *testing.T) {
	// Hand-computed expectation using the same primitives.
	ha1 := md5hex("admin:EC20:secret")
	ha2 := md5hex("GET:/auth.cgi")
	want := md5hex(ha1 + ":" + "nonceX" + ":" + "1" + ":" + "cnonceY" + ":" + ha2)

	got := ec20CGIResponse("admin", "EC20", "secret", "nonceX", "cnonceY", 1)
	if got != want {
		t.Errorf("ec20CGIResponse = %q, want %q", got, want)
	}
}
