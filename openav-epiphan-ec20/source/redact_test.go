package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestRedactSocketKey(t *testing.T) {
	cases := []struct{ in, want string }{
		{"admin:s3cret@192.0.2.10:80", "admin:***@192.0.2.10:80"},
		{"tcp|admin:s3cret@192.0.2.10:80", "tcp|admin:***@192.0.2.10:80"},
		{"ssh|admin:s3cret@192.0.2.10:22", "ssh|admin:***@192.0.2.10:22"},
		{"admin:pa:ss:word@192.0.2.10:80", "admin:***@192.0.2.10:80"},      // ':' in password
		{"admin:p@ss@192.0.2.10:80", "admin:***@192.0.2.10:80"},            // '@' in password
		{"admin:pa|ss@192.0.2.10:80", "admin:***@192.0.2.10:80"},           // '|' in password, no protocol
		{"tcp|admin:pa|ss:x@192.0.2.10:80", "tcp|admin:***@192.0.2.10:80"}, // both
		{"admin:p%2Fs%40w@192.0.2.10:80", "admin:***@192.0.2.10:80"},       // percent-encoded
		{"192.0.2.10:80", "192.0.2.10:80"},                                 // no credentials
		{"tcp|192.0.2.10:80", "tcp|192.0.2.10:80"},                         // protocol, no credentials
		{":@192.0.2.10:80", ":@192.0.2.10:80"},                             // framework's empty-creds form
		{"admin@192.0.2.10:80", "admin@192.0.2.10:80"},                     // user only, nothing to hide
		{"", ""},
	}
	for _, c := range cases {
		if got := redactSocketKey(c.in); got != c.want {
			t.Errorf("redactSocketKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// captureStdout runs fn and returns what it wrote to os.Stdout (framework.Log
// prints there).
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	defer func() { os.Stdout = old }()
	fn()
	w.Close()
	os.Stdout = old
	return <-done
}

// TestDriverLogsNeverContainPassword drives every driver entry point that logs
// its socketKey and asserts the password never reaches stdout.
func TestDriverLogsNeverContainPassword(t *testing.T) {
	const pw = "Sup3rS3cretPw"
	host, _ := fakeVISCADevice(t)
	key := "admin:" + pw + "@" + strings.TrimPrefix(host, "admin:x@")
	out := captureStdout(t, func() {
		_, _ = getCameraStatus(key)
		_, _ = getPTZPosition(key)
		_, _ = getPresets(key)
		_, _ = getPreview(key)
		_, _ = controlPTZ(key, "1", "1", `{"zoom":1}`)
		_, _ = controlPTZHome(key)
		_, _ = recallPreset(key, "1")
		_, _ = savePreset(key, "1", "x")
		_, _ = controlTracking(key, "bogus", "")
	})
	if !strings.Contains(out, "admin:***@") {
		t.Fatalf("expected redacted socketKey in logs, got:\n%s", out)
	}
	if strings.Contains(out, pw) {
		t.Errorf("password leaked into logs:\n%s", out)
	}
}

// TestParseSocketKey_PercentEncodedCredentials: openav-mcp percent-encodes the
// username/password in the URL path segment. The framework's echo router hands the
// :address param over still-encoded whenever the request needs a RawPath (e.g. %2F,
// %40, %3A), so parseSocketKey must decode them.
func TestParseSocketKey_PercentEncodedCredentials(t *testing.T) {
	host, user, pass := parseSocketKey("ad%2Fmin:p%2Fs%23s%3Aw%40rd@192.0.2.10:80")
	if host != "192.0.2.10:80" || user != "ad/min" || pass != "p/s#s:w@rd" {
		t.Errorf("parseSocketKey = (%q,%q,%q), want (192.0.2.10:80, ad/min, p/s#s:w@rd)", host, user, pass)
	}
	// A '%' that is not a valid escape is kept literally rather than failing.
	if _, _, pass := parseSocketKey("admin:50%off@192.0.2.10:80"); pass != "50%off" {
		t.Errorf("invalid escape: pass = %q, want 50%%off", pass)
	}
}
