# Engineer Handoff — Agentic AV Control Plane

**What this is:** an AI-first control layer for AV rooms. An LLM agent takes a plain-English
request ("get room 320-B recording with the camera tracking the presenter") and drives
[Dartmouth OpenAV](https://github.com/Dartmouth-OpenAV) + Epiphan Pearl/EC20 over
[MCP](https://modelcontextprotocol.io) — on-prem, model-agnostic, no proprietary control programming.

**Positioning (important):** OpenAV is the brains/control; Epiphan is the reliable hardware; the agent
is the backbone *above* OpenAV. They stay separate — **"Epiphan hardware running OpenAV," never
"Epiphan OpenAV."** `openav-mcp` does not replace OpenAV; it exposes the existing REST surfaces as
agent-callable tools.

## The stack (how the repos connect)

```
LLM agent  ──MCP──▶  openav-mcp  ──REST──▶  OpenAV orchestrator ─┬─▶ Pearl microservice ─▶ Pearl
(SilkRoute /         (this repo,            (Dartmouth, :8080)   └─▶ EC20 microservice  ─▶ EC20 PTZ
 Hermes / OpenClaw /  openav-mcp/)          + Go microservices (this repo)
 Claude Desktop)
```

- **`epiphan-openav-bridge`** (this repo) — Go microservices (Pearl, EC20) + the `demo/` orchestrator
  stack + **`openav-mcp/`** (the MCP face).
- **`silkroute`** (sibling repo) — the model-agnostic agent orchestrator that consumes `openav-mcp`.
- **`epiphan-mcp-server`** (sibling repo) — 130-tool Pearl/Cloud/CMS MCP server (optional, complementary).

Clone the two repos as **siblings** under the same parent dir:
```bash
git clone https://github.com/ScientiaCapital/epiphan-openav-bridge.git
git clone https://github.com/ScientiaCapital/silkroute.git
```

---

## 1. Plug-and-play smoke test — no hardware, no cloud (VERIFIED)

Proves the MCP layer works end-to-end in mock mode. Only needs Python 3.11+.

```bash
cd epiphan-openav-bridge/openav-mcp
python3 -m venv .venv && source .venv/bin/activate
pip install -e ".[dev]"
pytest -q                      # → 39 passed
python scripts/roundtrip_demo.py
```

Expected tail (12 tools — EC20 now exposes jog + preset_save):
```
discovered 12 tools: ['ec20_jog', 'ec20_preset_recall', 'ec20_preset_save', 'ec20_ptz',
 'ec20_status', 'ec20_tracking', 'list_room_controls', 'pearl_control_recording',
 'pearl_singletouch', 'pearl_status', 'run_scene', 'set_room_state']
run_scene -> {"ok": true, ... "steps": [ ...tracking..., ...record... ]}
ROUND-TRIP OK
```

For a full narrated walkthrough (a whole lecture-capture scenario + read-only safety
gate + credential-leak assertion), run `python scripts/demo_smart_room.py` — it also
regenerates the shareable **[openav-mcp/DEMO.md](openav-mcp/DEMO.md)**.

## 2. Full agentic path — plain English → room (mock, needs a local model)

The SilkRoute orchestrator decides which tools to call. Needs [Ollama](https://ollama.com) + a model.

```bash
cd silkroute
python3 -m venv .venv && source .venv/bin/activate && pip install -e ".[dev]"
ollama pull qwen2.5:14b            # or any tool-calling model
# Register openav-mcp as an upstream MCP server, then run the agent:
export SILKROUTE_OLLAMA_ENABLED=true
export SILKROUTE_MCP_EPIPHAN_ENABLED=true
export SILKROUTE_MCP_EPIPHAN_COMMAND="$(pwd)/.venv/bin/python"
export SILKROUTE_MCP_EPIPHAN_ARGS='["-m","openav_mcp"]'
export OPENAV_MOCK=true
export OPENAV_DEVICES='[{"alias":"room-320b-pearl","host":"pearl-host","username":"admin","password":"x","kind":"pearl"},{"alias":"room-320b-cam","host":"ec20-host","username":"admin","password":"x","kind":"ec20"}]'
export PYTHONPATH="$(pwd)/../epiphan-openav-bridge/openav-mcp"
silkroute run "Get room 320-B recording with the camera tracking the presenter, then confirm status"
```

The generic **N-server** bridge means you can register `openav-mcp` and `epiphan-mcp-server` and
Sony/DSP servers at once (see `silkroute` `MCPConfig.servers`). The `--mock-mcp` demo
(`python demo/agent_ready_av_demo.py --mock-mcp` in silkroute) is the canned Pearl variant.

## 3. Go live (real hardware)

1. Bring up the Go microservices + OpenAV orchestrator: `cd demo && docker compose up`
   (Pearl svc + EC20 svc + `ghcr.io/dartmouth-openav/orchestrator`). Services listen on **:80**;
   creds are supplied per-request in the URL path (`user:pass@host`) — no `.env` needed.
2. Point `openav-mcp` at the orchestrator (drop `OPENAV_MOCK`):
   ```bash
   export OPENAV_ORCHESTRATOR_URL=http://localhost:8080
   export OPENAV_DEVICES='[{"alias":"room-320b-pearl","host":"<pearl-ip>",...}]'
   ```
   > Note: base `demo/docker-compose.yml` publishes a host port **only for the orchestrator** (`8080:80`).
   > `demo/docker-compose.override.yml` (auto-merged) publishes the Pearl/EC20 microservices on
   > `localhost:8081`/`8082` for the `openav-mcp` device layer + adds restart-on-boot.
3. Run the agent as in §2 without `OPENAV_MOCK`.

> **Deploying to a permanent room host (Raspberry Pi 5 / Ubuntu ARM64):** follow the step-by-step
> **[`demo/DEPLOY-RPI5.md`](demo/DEPLOY-RPI5.md)** — OS/Docker, the orchestrator ARM64 check, device
> `.env`, bring-up, live verification, `openav-mcp`, and autostart.

## Status — done / placeholder / decisions

| Area | State |
|---|---|
| Pearl Go microservice | ✅ Built, tests green. Real Pearl REST API v2.0. |
| EC20 Go microservice | ✅ **Hybrid driver — redesigned onto the real control planes** (the old `/api/*` REST model was wrong): **VISCA-over-IP (raw, TCP :5678)** for PTZ/presets/home/jog/position + **CGI** (`auth.cgi` session + `ptzctrl.cgi`/`param.cgi`) for AI tracking + status. 70 tests. **2 CONFIRM-ON-HARDWARE items remain** — see next sprint below. |
| `openav-mcp` (MCP face) | ✅ Built, 39 tests + ruff clean. Exposes `ec20_jog` + `ec20_preset_save`; ranges schema-enforced. |
| SilkRoute orchestration | ✅ N-server bridge + fit-to-hardware routing shipped; round-trip to openav-mcp verified. |
| Live hardware run | ⛳ Needs the devices on the `192.168.8.x` LAN (EC20 `.8.11`, Pearl `.8.4`) — see next sprint. |
| Cloud model (better tool-calling than a 14B) | Set `SILKROUTE_OPENROUTER_API_KEY` and `--model deepseek/deepseek-v3.2` (or Claude/GPT/Gemini via OpenRouter — model-agnostic). |

## Next Sprint — live hardware bring-up (all gated on a box on the `192.168.8.x` LAN)

The whole software stack is built, DA-reviewed, and on `main`. Everything below needs the devices
powered + reachable (**EC20 `192.168.8.11`** admin/admin, **Pearl Mini `192.168.8.4`**). The two EC20
unknowns are each isolated to **one function**, so confirming them is a one-line change.

1. **EC20 live end-to-end + the 2 CONFIRM-ON-HARDWARE items.**
   Run the service: `docker build -t openav-epiphan-ec20 ./openav-epiphan-ec20 && docker run --rm -p 8082:80 openav-epiphan-ec20`
   - **Tracking command** — `ec20TrackingCommand()` in `openav-epiphan-ec20/source/cgiauth.go` defaults to
     `ptzctrl.cgi?post_aimode&<Single_Track|Frame_Track|Off>`. Confirm live (alt seen in device JS:
     `/cgi-bin/vip?set_ai_vip&vip=1`); also verify the `authorization` header case + whether `/cgi-bin/`
     needs transport Digest.
   - **PTZ degree calibration** — `panUnitsPerDegree`/`tiltUnitsPerDegree` in `driver.go` are PLACEHOLDER
     (`14.0`). Drive to known references, read `/ptzposition`, set the measured units-per-degree. (Presets +
     jog + tracking work *without* this; it only gates absolute-degree `ec20_ptz`.)
   - Drive it: `curl -X PUT http://localhost:8082/admin:admin@192.168.8.11/preset/1` (camera should move),
     `.../jog/up/10`, `.../tracking/enable -d '"presenter"'`, `.../status`.
2. **Pearl Mini live** (`192.168.8.4`): `curl -m8 http://192.168.8.4/api/v2.0/device` → verify auth type
   (Basic vs Digest; if Digest, port the EC20 fix to `openav-epiphan-pearl/source/driver.go`), then drive
   recording start→stop end-to-end.
3. **Deploy the full stack on the Raspberry Pi 5** — follow **[`demo/DEPLOY-RPI5.md`](demo/DEPLOY-RPI5.md)**.

**Audit backlog** (`.claude/observers/QUALITY.md`, no blockers): redact `socketKey` in `framework.Log`
calls (logs currently print `user:pass@host` — repo-wide hygiene); validate `ec20_tracking` mode
client-side in `openav-mcp` (Go + schema already do); document zoom semantics + verify the `getPreview`
RTSP port at the MCP boundary.

## Security
- `openav-mcp` injects device credentials from `OPENAV_DEVICES` — the **model never sees passwords**
  (it references devices by alias). Run with `--read-only` to expose only status tools.
- SilkRoute API is auth-on-by-default in production (`SILKROUTE_ENVIRONMENT=production`).

## Licenses
Go microservices: GPL-3.0 (matching Dartmouth-OpenAV). Root/demo/`openav-mcp`: see each dir's header
(`openav-mcp` is GPL-3.0-or-later). No proprietary deps.
