# Observer: Code Quality Report

**Date:** 2026-07-17
**Project:** epiphan-openav-bridge
**Observer Model:** observer-lite re-run (main-loop verified) — refreshes 2026-03-21 baseline
**Session:** Documentation-driven EC20 unblock sprint

---

## Gate Verdict

**0 BLOCKERS — clear to proceed.** No hardcoded credentials/secrets in source, no debt
markers, no silent failures in the new Python layer. Scan now includes `openav-mcp/`
(added 2026-07-17), previously unaudited.

---

## Critical (must fix before merge)

_No critical findings._

---

## Warnings (fix or log to backlog)

[WARNING] — openav-epiphan-pearl/source/driver.go:18 & openav-epiphan-ec20/source/driver.go:39 — `parseSocketKey()` duplicated identically in both drivers — **DISPOSITION: ACCEPTED.** Each microservice is a self-contained single-binary per OpenAV convention; a tiny vendored helper is intentional, not debt. Revisit only if a 3rd driver is added. (see ARCH.md)

---

## Resolved since 2026-03-21 baseline

[RESOLVED] — README license mismatch — root `README.md:142-143` now correctly states the dual-license split (root project = MIT, Go microservices = GPL-3.0). Root `LICENSE` = MIT.

[RESOLVED] — Missing curl test scripts — `openav-epiphan-pearl/pearl_curl_tests.sh` and `openav-epiphan-ec20/ec20_curl_tests.sh` now exist in the repo.

---

## Info (nice to have)

[INFO] — openav-epiphan-ec20/source/driver.go:22-35 — The 12 EC20 REST endpoint URL paths remain PLACEHOLDER — confirmed undiscoverable from public docs (see `.claude/programs/ec20-api-discovery.md`, 2026-07-17 doc-research). Behavioral facts (preset range, tracking modes, PTZ limits, ports) are now DOC-CONFIRMED and applied. Paths need a hardware probe (`ec20_probe.sh`).

[INFO] — openav-mcp/ — New Python MCP layer (7 source files) scanned for the first time. Credentials resolved internally by alias; model never sees passwords. Read-only tools always exported; mutating tools gated behind `--read-only`. No secrets, no bare excepts.

---

## Code Quality Metrics

| Metric | Value |
|--------|-------|
| Files scanned | 6 Go source/test (+2 framework submodule) + 7 Python (openav-mcp) + shell/config + docs |
| Critical findings | 0 |
| Blockers | 0 |
| Warnings | 1 (accepted) |
| Info items | 2 |
| Tech debt markers (TODO/FIXME/HACK/XXX) in source | 0 |
| Hardcoded secrets in source | 0 (fake fixtures in test files only) |
| Silent failures (Python bare except) | 0 |
| Test counts | Pearl: 46 · EC20: 74 · openav-mcp: 18 (top-level test funcs) |

---

## Monitoring Runs

| Date | Session | Task | Files Checked | Findings | Status |
|------|---------|------|--------------|----------|--------|
| 2026-03-21 | Initial baseline | DA audit + activation | 22+ files | 3W, 3I | COMPLETE |
| 2026-07-17 | EC20 unblock sprint | observer-lite re-run (incl. openav-mcp) | Go + Python source | 0 BLOCKER, 1W (accepted), 2 resolved | COMPLETE |
