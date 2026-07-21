package main

// EC20 CGI app-layer authentication.
//
// Above lighttpd's transport HTTP Digest (see ec20DigestAuthHeader), the EC20's CGI
// surface has its OWN session layer: a custom challenge/response against
// /cgi-bin/auth.cgi that grants a token carried in the `authorization` request
// header on every subsequent CGI call. Reverse-engineered from the device web
// UI (build-new.min.js). See .claude/programs/ec20-api-discovery.md Appendix A.
//
// Handshake (all GET /cgi-bin/auth.cgi):
//  1. No `authenticate` header -> 401 with a CUSTOM challenge header:
//     WWW-Authenticate: base64(realm) "." base64(nonce) "." base64(qop)
//  2. HA1 = md5(user:realm:pass); HA2 = md5("GET:/auth.cgi")
//     response = md5(HA1:nonce:nc:cnonce:HA2)
//     -- nc is a RAW INTEGER (not zero-padded); qop is NOT in the hash.
//  3. Resend with request header:
//     authenticate: username="..",nonce="..",nc=<int>,cnonce="..",uri="/auth.cgi",response=".."
//  4. 200 -> read response headers `jwt` and `authorization`; cache both.
//     201 = bad credentials, 500 = server error.

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ec20CGISession holds the app-layer tokens for one device's CGI session.
type ec20CGISession struct {
	jwt           string
	authorization string
}

// ec20CGIAuthURI is the URI used in the digest hash and the authenticate header.
// The device hashes the short form "/auth.cgi" even though the request path is
// "/cgi-bin/auth.cgi" (matches the web UI exactly).
const ec20CGIAuthURI = "/auth.cgi"

// parseCGIChallenge decodes the auth.cgi challenge: base64(realm).base64(nonce).base64(qop).
func parseCGIChallenge(header string) (realm, nonce, qop string, err error) {
	parts := strings.Split(strings.TrimSpace(header), ".")
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("auth.cgi challenge not a base64 dot-triplet: %q", header)
	}
	decode := func(s string) (string, error) {
		b, e := base64.StdEncoding.DecodeString(s)
		return string(b), e
	}
	if realm, err = decode(parts[0]); err != nil {
		return "", "", "", fmt.Errorf("decoding realm: %w", err)
	}
	if nonce, err = decode(parts[1]); err != nil {
		return "", "", "", fmt.Errorf("decoding nonce: %w", err)
	}
	if qop, err = decode(parts[2]); err != nil {
		return "", "", "", fmt.Errorf("decoding qop: %w", err)
	}
	return realm, nonce, qop, nil
}

// ec20CGIResponse computes the EC20 custom digest response:
// md5(md5(user:realm:pass):nonce:nc:cnonce:md5("GET:/auth.cgi")).
// nc is a raw integer (not zero-padded); qop is intentionally absent from the hash.
func ec20CGIResponse(username, realm, password, nonce, cnonce string, nc int) string {
	ha1 := md5Hex(username + ":" + realm + ":" + password)
	ha2 := md5Hex("GET:" + ec20CGIAuthURI)
	return md5Hex(strings.Join([]string{ha1, nonce, strconv.Itoa(nc), cnonce, ha2}, ":"))
}

// ec20CGIAuthHeader builds the value for the `authenticate` request header.
func ec20CGIAuthHeader(username, nonce, cnonce string, nc int, response string) string {
	return fmt.Sprintf(`username="%s",nonce="%s",nc=%d,cnonce="%s",uri="%s",response="%s"`,
		username, nonce, nc, cnonce, ec20CGIAuthURI, response)
}

// ec20Cnonce32 returns a 32-hex-char client nonce (matches the web UI's cnonce length).
func ec20Cnonce32() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ec20Login performs the auth.cgi handshake against baseURL and returns the session.
// baseURL is the scheme+host (e.g. "http://192.168.8.11"), no trailing path.
func ec20Login(client *http.Client, baseURL, username, password string) (*ec20CGISession, error) {
	url := strings.TrimRight(baseURL, "/") + "/cgi-bin/auth.cgi"

	// Step 1: unauthenticated GET to obtain the challenge.
	challengeResp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("auth.cgi challenge request: %w", err)
	}
	io.Copy(io.Discard, challengeResp.Body)
	challengeResp.Body.Close()
	if challengeResp.StatusCode != http.StatusUnauthorized {
		return nil, fmt.Errorf("auth.cgi: expected 401 challenge, got %d", challengeResp.StatusCode)
	}
	realm, nonce, _, err := parseCGIChallenge(challengeResp.Header.Get("WWW-Authenticate"))
	if err != nil {
		return nil, err
	}

	// Step 2: compute the response and resend with the authenticate header.
	cnonce, err := ec20Cnonce32()
	if err != nil {
		return nil, err
	}
	const nc = 1
	response := ec20CGIResponse(username, realm, password, nonce, cnonce, nc)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("authenticate", ec20CGIAuthHeader(username, nonce, cnonce, nc, response))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth.cgi credential request: %w", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// 201 = bad credentials, 500 = server error, per device semantics.
		return nil, fmt.Errorf("auth.cgi login failed: HTTP %d", resp.StatusCode)
	}
	return &ec20CGISession{
		jwt:           resp.Header.Get("jwt"),
		authorization: resp.Header.Get("authorization"),
	}, nil
}

// ---------- session cache + authenticated CGI requests ----------

var (
	cgiSessionMu sync.Mutex
	cgiSessions  = map[string]*ec20CGISession{} // keyed by device host[:port]
)

// cgiSession returns a cached auth.cgi session for host, logging in if absent or
// force=true (used to refresh after a 401). Safe for concurrent callers.
func cgiSession(client *http.Client, baseURL, host, username, password string, force bool) (*ec20CGISession, error) {
	cgiSessionMu.Lock()
	defer cgiSessionMu.Unlock()
	if !force {
		if s, ok := cgiSessions[host]; ok {
			return s, nil
		}
	}
	s, err := ec20Login(client, baseURL, username, password)
	if err != nil {
		return nil, err
	}
	cgiSessions[host] = s
	return s, nil
}

// ec20CGISendGET issues an authenticated GET to a device CGI path (e.g.
// "/cgi-bin/ptzctrl.cgi?post_aimode&Off"), attaching the cached auth.cgi session
// token. On a 401 it re-logs in once and retries (token expiry). CGI uses the
// device's HTTP port from the socketKey — NOT VISCA's TCP port.
func ec20CGISendGET(socketKey, path string) ([]byte, error) {
	host, username, password := parseSocketKey(socketKey)
	baseURL := "http://" + host
	client := &http.Client{Timeout: 10 * time.Second}

	sess, err := cgiSession(client, baseURL, host, username, password, false)
	if err != nil {
		return nil, err
	}
	body, status, err := ec20CGIDo(client, baseURL+path, username, password, sess.authorization)
	if err != nil {
		return nil, err
	}
	if status == http.StatusUnauthorized {
		// App-layer token likely expired — re-login once and retry.
		sess, err = cgiSession(client, baseURL, host, username, password, true)
		if err != nil {
			return nil, err
		}
		body, status, err = ec20CGIDo(client, baseURL+path, username, password, sess.authorization)
		if err != nil {
			return nil, err
		}
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("ec20 CGI GET %s: HTTP %d", path, status)
	}
	return body, nil
}

// ec20CGIDo performs a GET with the app-layer `authorization` token, transparently
// answering a transport HTTP Digest challenge (lighttpd may guard /cgi-bin/) by
// reusing the driver's Digest helpers. It returns (body, status, err); a bare 401
// (no Digest challenge) is returned as status 401 so the caller can re-login.
func ec20CGIDo(client *http.Client, url, username, password, authToken string) ([]byte, int, error) {
	newReq := func(withDigest string) (*http.Request, error) {
		req, e := http.NewRequest(http.MethodGet, url, nil)
		if e != nil {
			return nil, e
		}
		// Assign the raw header-map keys directly rather than Header.Set: Set
		// canonicalizes "authorization" to "Authorization", which (a) drops the
		// literal lowercase name the EC20 web UI uses for the app token and (b)
		// would collide with / overwrite the transport Digest "Authorization".
		// Raw assignment keeps them as two distinct wire header lines.
		if authToken != "" {
			req.Header["authorization"] = []string{authToken} // app-layer session token (lowercase, web-UI form)
		}
		if withDigest != "" {
			req.Header["Authorization"] = []string{withDigest} // transport HTTP Digest
		}
		return req, nil
	}

	req, err := newReq("")
	if err != nil {
		return nil, 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	// Transport Digest challenge? Answer it and resend (keeping the app token).
	if resp.StatusCode == http.StatusUnauthorized {
		challenge := resp.Header.Get("WWW-Authenticate")
		resp.Body.Close()
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(challenge)), "digest") {
			u, e := http.NewRequest(http.MethodGet, url, nil)
			if e != nil {
				return nil, 0, e
			}
			hdr, e := ec20DigestAuthHeader(challenge, http.MethodGet, u.URL.RequestURI(), username, password)
			if e != nil {
				return nil, 0, e
			}
			req2, e := newReq(hdr)
			if e != nil {
				return nil, 0, e
			}
			resp, err = client.Do(req2)
			if err != nil {
				return nil, 0, err
			}
		} else {
			// Bare 401 = app-layer token expired; let the caller re-login.
			return nil, http.StatusUnauthorized, nil
		}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	return body, resp.StatusCode, nil
}

// ec20TrackingCommand returns the CGI path for an AI-tracking action.
//
// ⚠ CONFIRM ON HARDWARE. The EC20's tracking-toggle command is NOT in public docs.
// Candidates from reverse-engineering (device web-UI JS + the SMTAV ODM family):
//
//	(a) /cgi-bin/ptzctrl.cgi?post_aimode&<Single_Track|Frame_Track|Off>   (SMTAV family)
//	(b) /cgi-bin/vip?set_ai_vip&vip=<1|0>                                  (this unit's JS)
//
// We default to (a), mapping presenter->Single_Track, zone->Frame_Track. This is the
// ONLY place the wire command is defined — after a live probe, adjust here and nothing
// else changes. (See .claude/programs/ec20-api-discovery.md.)
func ec20TrackingCommand(action, mode string) string {
	if action == "disable" {
		return "/cgi-bin/ptzctrl.cgi?post_aimode&Off"
	}
	aiMode := "Single_Track" // presenter (default)
	if mode == "zone" {
		aiMode = "Frame_Track"
	}
	return "/cgi-bin/ptzctrl.cgi?post_aimode&" + aiMode
}
