# epiphan-openav-bridge

**Branch**: main | **Updated**: 2026-07-17

## Status
Go microservices bridging Dartmouth **OpenAV** ↔ Epiphan **Pearl + EC20**, plus a new **`openav-mcp`**
Python MCP server — the **AI-first layer** that lets an LLM agent drive AV rooms in plain English.
Pearl microservice ✅ (46 tests, builds clean). EC20 ✅ built (80 tests) but **REST endpoints are
PLACEHOLDER pending hardware**. `openav-mcp` ✅ built (11 tests, fresh-venv install + round-trip
verified). Positioning: OpenAV = brains, Epiphan = reliable hardware, agent = backbone above — stay
separate ("Epiphan hardware running OpenAV", never "Epiphan OpenAV").

## Today's Focus (next session)
1. [ ] **Verify EC20 endpoints on real hardware** — the paths in `openav-epiphan-ec20/source/driver.go`
       are placeholders; run `.claude/programs/ec20-api-discovery.md` against a real EC20.
2. [ ] **Live bring-up** — `cd demo && docker compose up` (Pearl + EC20 + OpenAV orchestrator), point
       `openav-mcp` at it (drop `OPENAV_MOCK`), run the agent path in `HANDOFF.md` §3.
3. [ ] **Publish/CI `openav-mcp`** — add CI + package it for the OpenAV community (open-core).

## Done (This Session)
- Built **`openav-mcp/`** — MCP face over the OpenAV orchestrator (scene layer) + Pearl/EC20
  microservices (device layer), creds injected from config (LLM never sees passwords), `--read-only`
  gate, mock mode. 11 tests + `scripts/roundtrip_demo.py` (verified).
- Added **`HANDOFF.md`** (engineer/Vadim plug-and-play guide) + fixed README/CLAUDE errors
  (`.env.example` didn't exist; build path `./source/`; service binds `:80`; stale "Planned" statuses).

## Blockers
- EC20 REST endpoints unverified (no hardware yet). Live demo needs a Pearl/EC20 + OpenAV orchestrator.

## Start here
**`HANDOFF.md`** — verified no-hardware smoke test (`openav-mcp/scripts/roundtrip_demo.py`) + go-live.
The agent that drives this lives in the sibling repo **`silkroute`** (its `README.md` "Agentic AV
control plane"). Business strategy: `../epiphan-pi-strategic-report.md`.

## Tech Stack
Go 1.25 (Echo microservices, GPL-3.0) | Python 3.11+ (`openav-mcp`: mcp + httpx + structlog) |
Docker Compose | Dartmouth OpenAV orchestrator
