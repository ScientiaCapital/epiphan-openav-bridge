# Observer: Code Quality Report

**Date:** 2026-07-21
**Project:** epiphan-openav-bridge
**Observer Model:** observer-full (7 drift patterns + devil's advocate)
**Session:** End-of-sprint DA review of `feat/ec20-hybrid-driver` (6 commits) before merge to main
**Scope reviewed:** `git diff main...HEAD` — EC20 Go driver (driver.go, visca.go, cgiauth.go,
microservice.go + tests) and openav-mcp (client.py, server.py + tests) + demo deploy files.
**Baseline:** Go tests 69 pass · MCP tests 39 pass · vet/ruff/gofmt clean (per hand-off; re-ran `go test` = ok).

---

## Gate Verdict

**0 hard BLOCKERS — clear to merge with the WARNINGs logged.** No crashes, no data loss, no
committed secrets, no swallowed errors that fake success on the *tested* paths. The genuinely risky
code (degree→unit calibration, tracking CGI command, header handling) is honestly labeled
CONFIRM-ON-HARDWARE / PLACEHOLDER in-code and the MCP layer de-emphasizes the uncalibrated verb.
The findings below are latent correctness + hygiene issues that the passing tests, vet, ruff and
gofmt do **not** catch — most because the mocks canonicalize/accept exactly what the code produces.

---

## Warnings (fix or log to backlog)

[WARNING] cgiauth.go:211-215 — **`authorization` vs `Authorization` header collision.** `newReq()`
sets `authorization` (app-layer token) then, in the transport-Digest branch, `Authorization`
(Digest). Go's `http.Header.Set` canonicalizes both to `Authorization`, so the second Set
**overwrites** the app token — directly contradicting the "keeping the app token" comment
(line 228). Fix: carry the Digest response in the same canonical header only if the device truly
needs both, or confirm the device never Digest-guards /cgi-bin/ and drop the branch. Untested path
(the mock never issues a Digest challenge on the data endpoint), so tests stay green.

[WARNING] cgiauth.go:211-213 — **App token transmitted as canonical `Authorization`, not literal
lowercase `authorization`.** The handshake was reverse-engineered from the web UI which uses a
lowercase `authorization` header; Go always sends it title-cased. RFC-compliant (case-insensitive)
servers are fine, but a case-sensitive CGI lookup on the real EC20 would never see the token →
401-loop → re-login-loop → error. Tests can't detect this: `httptest` + `r.Header.Get` canonicalize
on read. CONFIRM-ON-HARDWARE.

[WARNING] driver.go:250-275, 299-372 — **Misleading success on uncalibrated absolute PTZ.**
`controlPTZ` validates real degree ranges then converts with `panUnitsPerDegree = 14.0`
(PLACEHOLDER, one live data point) and returns `"ok"` on VISCA completion — i.e. the camera reports
success while pointing at the *wrong* position until Story-D calibration lands. Mitigated: server.py
labels `ec20_ptz` "secondary — prefer ec20_jog / ec20_preset_recall". Fix: gate absolute PTZ behind
a calibrated flag, or return a "uncalibrated" note until measured.

[WARNING] driver.go:164,301,389,410,497 (+getPTZPosition/getPreview/home) — **Credentials logged in
cleartext.** `framework.Log(function + " ... " + socketKey)` prints the full
`user:pass@host` to stdout; `framework.Log` does not redact (framework:1070). controlPTZ:301 is
NEW this sprint (adds body to the line). Pre-existing OpenAV convention, but it contradicts the
project's stated credential-hygiene invariant (the one enforced in openav-mcp). Fix: log
`parseSocketKey`'s host only, never the raw socketKey.

[WARNING] cgiauth.go:262-282 + cgiauth_test.go:104-190 — **Tracking CGI command is a CONFIRM-
ON-HARDWARE placeholder that tests can't validate.** `ec20TrackingCommand` defaults to
`/cgi-bin/ptzctrl.cgi?post_aimode&...` (one of two reverse-engineered candidates). The test device's
`mux.HandleFunc("/")` catch-all returns 200 for ANY path, so the tracking tests pass regardless of
whether the path is correct → false green. On real hardware a wrong-but-200 path would silently
no-op (success returned, tracking never engaged). One live probe required before trusting tracking.

---

## Smells (log to backlog)

[SMELL] cgiauth.go:137 — `jwt` is parsed from the login response and cached but **never sent** on
any subsequent CGI request (only `authorization` is). If the real device requires the `jwt` header
on data calls, tracking fails. Verify on hardware; either send it or stop capturing it.

[SMELL] cgiauth.go:151-165 — **Global login mutex held across network I/O.** `cgiSession` holds the
single package-level `cgiSessionMu` for the entire `ec20Login` (2 HTTP round trips, 10s timeout
each). This serializes CGI logins across *all* devices — a slow/hung login on device A blocks every
other device's tracking call. The service is explicitly meant to front many devices. Fix: per-host
lock, or release the mutex during the network call (double-checked insert).

[SMELL] driver.go:190-194 — `getPresets` returns `"unsupported: ..."` as a **successful** (nil-err)
response. An agent can't distinguish this from a real result. Prefer an error or a structured
`{"supported": false}` payload.

[SMELL] driver.go:201-219 — `getPreview` hardcodes `rtsp://host:554/1`; the comment itself admits
rtspport is device-configurable and the stream path `/1` is unverified. NEEDS-PROBE.

[SMELL] visca.go:66-69 — `viscaStop` is dead code: production uses `jogDirection("stop")` →
`viscaJog(...Stop...)` directly; `viscaStop` is referenced only by visca_test.go.

[SMELL] client.py:194-202 — `ec20_tracking` does **not** validate `mode ∈ {presenter, zone}` client-
side, though server.py's schema and the Go driver both enforce it. Mock accepts any mode and ignores
it; live hardware (Go) rejects it — the exact mock/live parity gap this sprint fixed for pan/tilt/
preset ranges. Add the enum check to the client for parity.

[SMELL] driver.go:288-297 + server.py ec20_ptz — **zoom semantics undocumented at the contract
boundary.** Go treats zoom as a raw VISCA 16-bit position (0..0x4000), but the MCP schema exposes
`zoom` as an unbounded `number` with no description; an agent has no way to know it's not a zoom
factor/percentage. `zoomMax` tele ceiling is also NEEDS-PROBE.

[SMELL] client.py:230-244 (jog) vs driver.go clampSpeed — jog speed: Python enforces 1-24, Go
*clamps* (and caps tilt at 20). A tilt-only jog at speed 21-24 is accepted by Python but silently
clamped by Go. Cosmetic, but a documented range that isn't actually the enforced range.

---

## Test coverage gaps (new code)

- **ec20CGIDo transport-Digest branch (cgiauth.go:229-248) is untested** — the mock only issues a
  bare 401, never a `WWW-Authenticate: Digest` on the data endpoint, so the entire lighttpd-Digest
  answer path (and the header-collision bug above) has zero coverage.
- **viscaSend frame reassembly untested** — no test splits a completion across multiple TCP reads or
  injects noise before 0x90; the buffering loop looks correct but is unexercised for fragmentation.
- **No concurrency tests** for `viscaSend` (two connections per controlPTZ, non-atomic) or
  `cgiSession` (global mutex, force re-login race).
- **parseCGIChallenge** error paths (short/malformed triplet, non-base64 parts) untested.

---

## Standing disposition (carried forward — unchanged)

[ACCEPTED] parseSocketKey duplicated across both drivers — self-contained single-binary per OpenAV
convention. Revisit only if a 3rd driver is added. (see ARCH.md; confirmed with Tim 2026-07-18.)

---

## Code Quality Metrics

| Metric | Value |
|--------|-------|
| Files reviewed | EC20 Go (4 src + 3 test), openav-mcp (2 src + 1 test), demo (3) |
| Hard blockers | 0 |
| Warnings | 5 |
| Smells | 8 |
| Coverage gaps flagged | 4 |
| Committed secrets | 0 (test fixtures only; EC20 admin/admin is the documented device default) |
| Test counts (this sprint) | Go: 69 · MCP: 39 (both pass) |

---

## Monitoring Runs

| Date | Session | Task | Findings | Status |
|------|---------|------|----------|--------|
| 2026-03-21 | Initial baseline | DA audit + activation | 3W, 3I | COMPLETE |
| 2026-07-17 | EC20 doc unblock | observer-lite re-run | 0 BLOCKER, 1W (accepted) | COMPLETE |
| 2026-07-21 | ec20-hybrid-driver merge review | observer-full DA | 0 BLOCKER, 5W, 8 SMELL, 4 coverage gaps | COMPLETE |
