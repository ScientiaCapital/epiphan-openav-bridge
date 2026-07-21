# epiphan-openav-bridge

**Branch**: main | **Updated**: 2026-07-18 (mock-mode smart-room demo asset)

## Status
Go microservices bridging Dartmouth **OpenAV** ↔ Epiphan **Pearl + EC20**, plus a new **`openav-mcp`**
Python MCP server — the **AI-first layer** that lets an LLM agent drive AV rooms in plain English.
Pearl microservice ✅ (46 tests, builds clean). EC20 ✅ built (84 test runs) — **behavior now matches
Epiphan docs**; only the REST URL *paths* remain PLACEHOLDER pending a hardware probe. `openav-mcp` ✅
built (round-trip verified). Positioning: OpenAV = brains, Epiphan = reliable hardware, agent =
backbone above — stay separate ("Epiphan hardware running OpenAV", never "Epiphan OpenAV").

## Done (This Session — mock-mode smart-room demo asset)
- Built **`openav-mcp/scripts/demo_smart_room.py`** — a narrated, hardware-free demo of an AI agent
  running a full lecture capture in Room 320B over the real MCP protocol (discover → frame preset →
  PTZ → track → record → confirm-via-status → stop), + a `--read-only` safety-gate showcase and a
  credential-leak assertion. Generates the shareable **`openav-mcp/DEMO.md`** walkthrough.
- **Fixed a cross-layer drift bug**: the Python `ec20_preset_recall` still rejected preset 0 (`1-255`)
  after yesterday's Go fix (`0-255`). Synced `client.py` + `server.py`; added preset-0 tests.
- Tests green (openav-mcp 13 pass, ruff clean); linked DEMO.md from README + HANDOFF.
- Prior session (2026-07-17): EC20 doc-driven unblock — preset bug fix, PTZ/tracking validation,
  `ec20_probe.sh` harness (shipped to main).

## Today's Focus (next session)
1. [ ] **Confirm EC20 paths on hardware** — run `bash openav-epiphan-ec20/ec20_probe.sh` against a real
       EC20 (or inspect its web-UI JS), paste CONFIRMED paths into `driver.go`, update the mock + tests.
2. [ ] **Live bring-up** — `cd demo && docker compose up` (Pearl + EC20 + OpenAV orchestrator), point
       `openav-mcp` at it (drop `OPENAV_MOCK`), run the agent path in `HANDOFF.md` §3.
3. [ ] **Publish/CI `openav-mcp`** — add CI + package it for the OpenAV community (open-core).
       DEMO.md is now a ready-made asset for the Phase 4 blog post.

## Blockers
- EC20 REST URL *paths* still need a hardware probe (behavior is doc-confirmed; harness is ready).
  Live demo needs a Pearl/EC20 + OpenAV orchestrator on the network.

## Start here
**`HANDOFF.md`** — verified no-hardware smoke test (`openav-mcp/scripts/roundtrip_demo.py`) + go-live.
The agent that drives this lives in the sibling repo **`silkroute`** (its `README.md` "Agentic AV
control plane").

## Tech Stack
Go 1.25 (Echo microservices, GPL-3.0) | Python 3.11+ (`openav-mcp`: mcp + httpx + structlog) |
Docker Compose | Dartmouth OpenAV orchestrator
