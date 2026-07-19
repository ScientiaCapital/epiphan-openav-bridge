package main

// EC20 CGI app-layer authentication.
//
// Above lighttpd's transport HTTP Digest (see ec20DoWithDigest), the EC20's CGI
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
