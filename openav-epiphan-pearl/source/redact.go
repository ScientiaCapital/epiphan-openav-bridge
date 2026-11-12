package main

import "strings"

// redactSocketKey returns socketKey with its password replaced by "***", for
// logging. It handles "user:pass@host[:port]" and "proto|user:pass@host[:port]",
// including passwords that contain ':', '@' or '|': the credentials end at the
// LAST '@', and the password starts after the first ':' following the optional
// protocol prefix. Keys without a password (or with an empty one) are returned
// unchanged.
//
// This only covers log lines written by this driver. The upstream
// microservice-framework submodule logs the raw socketKey itself on many paths;
// see SECURITY.md.
func redactSocketKey(socketKey string) string {
	at := strings.LastIndex(socketKey, "@")
	if at < 0 {
		return socketKey
	}
	creds := socketKey[:at]
	start := 0
	// A protocol prefix is a run of letters ending in '|' (tcp|, udp|, ssh|, telnet|).
	if i := strings.Index(creds, "|"); i > 0 && isLetters(creds[:i]) {
		start = i + 1
	}
	colon := strings.Index(creds[start:], ":")
	if colon < 0 || start+colon+1 == at {
		return socketKey
	}
	return socketKey[:start+colon+1] + "***" + socketKey[at:]
}

func isLetters(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}
