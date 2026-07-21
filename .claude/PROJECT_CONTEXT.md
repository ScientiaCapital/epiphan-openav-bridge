# epiphan-openav-bridge

**Branch**: main | **Updated**: 2026-07-21 (EC20 hybrid-driver sprint shipped)

## Status
Go microservices bridging Dartmouth **OpenAV** ↔ Epiphan **Pearl + EC20**, plus the **`openav-mcp`**
Python MCP server — the **AI-first layer** that lets an LLM agent drive AV rooms in plain English.
Pearl microservice ✅. **EC20 driver fully redesigned onto its real hardware control planes** (the old
RESTful `/api/*` placeholder model was wrong): **VISCA-over-IP (raw, TCP `:5678`)** for PTZ/presets/
home/position, **CGI** (`/cgi-bin/auth.cgi` session + `ptzctrl.cgi`/`param.cgi`) for AI tracking + status.
`openav-mcp` aligned (adds `ec20_jog`, `ec20_preset_save`). All green (Go 70 + Pearl + MCP 39), DA-audited
(0 blockers), **shipped to `main`** (`ffc4c77→6d1d16c`, 9 commits). Only live-hardware confirmation remains.
Positioning: OpenAV = brains, Epiphan = reliable hardware, agent = backbone ("Epiphan hardware running OpenAV").

## Done (This Session — EC20 hybrid-driver sprint)
- **Digest auth fix** — real EC20 needs HTTP Digest, not Basic (`driver.go` `ec20DoWithDigest`, stdlib-only).
- **VISCA-over-IP transport** (`visca.go`, TCP `:5678`, hardware-verified fw 3.3.40): preset recall/save,
  home, jog, absolute PTZ, position/zoom/version inquiries; handlers rewired to it.
- **CGI/JWT plane** (`cgiauth.go`): the reverse-engineered `auth.cgi` custom-digest handshake + session
  cache; `controlTracking` wired through it. Fixed an `authorization`/`Authorization` header-collision
  bug the DA audit caught (+ regression test).
- **Removed the dead REST `/api/*` layer**; added the **`jog`** verb; `getPreview` → RTSP URL.
- **`openav-mcp` aligned**: `ec20_jog` + `ec20_preset_save`, min/max range schemas, Go↔Python ranges synced.
- **Pi 5 deployment runbook** (`demo/DEPLOY-RPI5.md`) + `docker-compose.override.yml` + GNU-sed portability fix.
- Docs/README/discovery-log/memory updated; observer/DA audit recorded (`.claude/observers/`).

## Today's Focus (next session — all hardware-gated, need a box on the 192.168.8.x LAN)
1. [ ] **EC20 live end-to-end** (`192.168.8.11`, admin/admin): confirm the 2 CONFIRM-ON-HARDWARE items —
       exact tracking command (`ec20TrackingCommand` in `cgiauth.go`) + degrees→VISCA-units PTZ
       calibration (`controlPTZ` consts in `driver.go`); drive preset recall (camera moves), jog, tracking.
2. [ ] **Pearl Mini live** (`192.168.8.4`): verify auth type (Basic vs Digest — may need the same fix),
       drive recording start→stop end-to-end.
3. [ ] **Deploy full stack on the Raspberry Pi 5** via `demo/DEPLOY-RPI5.md` (orchestrator ARM64 check,
       real IPs in `.env`, `openav-mcp` live mode, autostart).

## Blockers
- Both devices need a box on the `192.168.8.x` LAN with them powered (EC20 `.8.11`, Pearl `.8.4`).
  Note: the EC20 CGI/lighttpd plane can wedge under repeated hung requests — power-cycle clears it.
  VISCA (`:5678`) is unaffected.

## Start here
**`HANDOFF.md`** — verified no-hardware smoke test (`openav-mcp/scripts/roundtrip_demo.py`) + go-live.
For live deployment: **`demo/DEPLOY-RPI5.md`**. The agent that drives this lives in the sibling repo
**`silkroute`** (its `README.md` "Agentic AV control plane").

## Tech Stack
Go 1.25 (Echo microservices, GPL-3.0; EC20 = raw VISCA-over-TCP + CGI) | Python 3.11+ (`openav-mcp`:
mcp + httpx + structlog) | Docker Compose | Dartmouth OpenAV orchestrator
