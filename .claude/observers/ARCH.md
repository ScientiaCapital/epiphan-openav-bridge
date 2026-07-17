# Observer: Architecture Report

**Date:** 2026-03-21
**Project:** epiphan-openav-bridge
**Observer Model:** Claude Opus 4.6 (DA audit) + Sonnet (sweep)
**Session:** Initial baseline activation

---

## Blockers (stop work immediately)

_No blockers. Codebase is architecturally sound._

---

## Risks (address this sprint)

[RISK] — openav-epiphan-ec20/source/driver.go:19-34 — All 12 EC20 API endpoints are PLACEHOLDER paths — The EC20 REST API has no public documentation. Driver structure is correct but endpoint paths cannot be validated without real hardware. This is the single largest technical risk in the project.

[RISK] — openav-epiphan-pearl/source/driver.go:18 & openav-epiphan-ec20/source/driver.go:39 — `parseSocketKey()` duplicated across both drivers — Currently acceptable per OpenAV self-contained driver convention. Becomes a real risk if a third driver is added or if the parsing logic needs to change (would require coordinated updates in two places).

---

## Smells (log to backlog)

[RESOLVED 2026-07-17] — License inconsistency — root `README.md:142-143` now states the dual-license correctly (root=MIT, microservices=GPL-3.0); both microservice READMEs state GPL-3.0. No inconsistency remains.

[RESOLVED 2026-07-17] — curl test scripts — `pearl_curl_tests.sh` and `ec20_curl_tests.sh` now exist. (Also added: `ec20_probe.sh`, a non-destructive REST endpoint discovery harness.)

[SMELL] — ROADMAP.md — Phase 1 completion status unclear despite proof script existing — Tracking inconsistency.

---

## Contract Compliance

| Contract | Status | Notes |
|----------|--------|-------|
| No feature contract active | N/A | `.claude/contracts/` is empty — contracts created on-demand by `/build` |

---

## Devil's Advocate Challenges

| File | Challenge | Verdict |
|------|-----------|---------|
| `parseSocketKey()` (both drivers) | Should this be a shared utility package? | **Acceptable tradeoff** — OpenAV convention is self-contained drivers. Framework itself has a similar parser but uses SSH/Sony auth patterns. Custom per-driver is correct for now. |
| `ec20APIPostJSON()` (EC20 only) | Is this needed? Pearl uses `pearlAPIPost()` without JSON body. | **Justified** — EC20 PTZ commands require JSON request bodies (pan/tilt/zoom parameters). Pearl recording commands use URL path-based control. Different devices, different APIs. |
| `ec20APIGetRaw()` (EC20 only) | Is this needed? Could reuse `ec20APIGet()`. | **Justified** — Preview endpoint returns raw JPEG bytes, not JSON. Needs separate handler to avoid JSON parsing failure. |
| Docker `rm -f go.mod go.sum` step | Why delete before `go mod init`? | **Required** — Framework submodule's `.gitignore` excludes `go.mod`/`go.sum`, but `docker COPY` includes all files. Without deletion, `go mod init` fails on existing `go.mod`. |
