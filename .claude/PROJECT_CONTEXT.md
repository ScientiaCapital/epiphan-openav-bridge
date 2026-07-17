# epiphan-openav-bridge

**Branch**: main | **Updated**: 2026-07-17 (EC20 doc-driven unblock sprint)

## Status
Go microservices bridging Dartmouth **OpenAV** ↔ Epiphan **Pearl + EC20**, plus a new **`openav-mcp`**
Python MCP server — the **AI-first layer** that lets an LLM agent drive AV rooms in plain English.
Pearl microservice ✅ (46 tests, builds clean). EC20 ✅ built (84 test runs) — **behavior now matches
Epiphan docs**; only the REST URL *paths* remain PLACEHOLDER pending a hardware probe. `openav-mcp` ✅
built (round-trip verified). Positioning: OpenAV = brains, Epiphan = reliable hardware, agent =
backbone above — stay separate ("Epiphan hardware running OpenAV", never "Epiphan OpenAV").

## Done (This Session — EC20 doc-driven unblock)
- **Mined Epiphan's own docs** (EC20 AI User Guide + Q-SYS plugin README via Epiphan Knowledge) to
  de-risk EC20 without hardware. Findings logged in `.claude/programs/ec20-api-discovery.md`.
- **Fixed a real preset bug** in `driver.go`: `validatePresetID` rejected preset 0 (range was 1–255);
  docs confirm range is **0–255** (preset 0 is valid). Now corrected + tested.
- Added **DOC-CONFIRMED validation**: tracking modes restricted to `presenter`/`zone`; PTZ range
  checks (pan ±162.5°, tilt −30°..+90°). New tests; both Go suites green (Pearl 63, EC20 84 runs).
- Built **`ec20_probe.sh`** — non-destructive (GET-only) hardware-confirmation harness implementing
  the discovery program. Verified end-to-end in dry mode. Makes hardware day a ~30-min job.
- Refreshed observers (`QUALITY.md` 0 blockers; 2 stale warnings confirmed already resolved).

## Today's Focus (next session)
1. [ ] **Confirm EC20 paths on hardware** — run `bash openav-epiphan-ec20/ec20_probe.sh` against a real
       EC20 (or inspect its web-UI JS), paste CONFIRMED paths into `driver.go`, update the mock + tests.
2. [ ] **Live bring-up** — `cd demo && docker compose up` (Pearl + EC20 + OpenAV orchestrator), point
       `openav-mcp` at it (drop `OPENAV_MOCK`), run the agent path in `HANDOFF.md` §3.
3. [ ] **Publish/CI `openav-mcp`** — add CI + package it for the OpenAV community (open-core).

## Blockers
- EC20 REST URL *paths* still need a hardware probe (behavior is doc-confirmed; harness is ready).
  Live demo needs a Pearl/EC20 + OpenAV orchestrator on the network.

## Start here
**`HANDOFF.md`** — verified no-hardware smoke test (`openav-mcp/scripts/roundtrip_demo.py`) + go-live.
The agent that drives this lives in the sibling repo **`silkroute`** (its `README.md` "Agentic AV
control plane"). Business strategy: `../epiphan-pi-strategic-report.md`.

## Tech Stack
Go 1.25 (Echo microservices, GPL-3.0) | Python 3.11+ (`openav-mcp`: mcp + httpx + structlog) |
Docker Compose | Dartmouth OpenAV orchestrator
