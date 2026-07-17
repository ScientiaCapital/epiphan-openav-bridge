# Engineer Handoff — Agentic AV Control Plane

**What this is:** an AI-first control layer for AV rooms. An LLM agent takes a plain-English
request ("get room 320-B recording with the camera tracking the presenter") and drives
[Dartmouth OpenAV](https://github.com/open-avc/openavc) + Epiphan Pearl/EC20 over
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
pytest -q                      # → 11 passed
python scripts/roundtrip_demo.py
```

Expected tail:
```
discovered 10 tools: ['ec20_preset_recall', 'ec20_ptz', 'ec20_status', 'ec20_tracking',
 'list_room_controls', 'pearl_control_recording', 'pearl_singletouch', 'pearl_status',
 'run_scene', 'set_room_state']
run_scene -> {"ok": true, ... "steps": [ ...tracking..., ...record... ]}
ROUND-TRIP OK
```

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
2. Point `openav-mcp` at them (drop `OPENAV_MOCK`):
   ```bash
   export OPENAV_ORCHESTRATOR_URL=http://localhost:8080
   export OPENAV_PEARL_URL=http://localhost:8081
   export OPENAV_EC20_URL=http://localhost:8082
   export OPENAV_DEVICES='[{"alias":"room-320b-pearl","host":"<pearl-ip>",...}]'
   ```
3. Run the agent as in §2 without `OPENAV_MOCK`.

## Status — done / placeholder / decisions

| Area | State |
|---|---|
| Pearl Go microservice | ✅ Built, 46 tests (mock). Builds clean (`go build ./source/`). |
| EC20 Go microservice | ⚠️ Built, 80 tests, but **REST endpoints are PLACEHOLDER** — verify on real EC20 hardware (`.claude/programs/ec20-api-discovery.md`) before production. |
| `openav-mcp` (MCP face) | ✅ Built, 11 tests, fresh-venv install verified, round-trip verified. |
| SilkRoute orchestration | ✅ N-server bridge + fit-to-hardware routing shipped; round-trip to openav-mcp verified. |
| Live hardware run | ⛳ Needs a Pearl/EC20 + OpenAV orchestrator on the bench. |
| Cloud model (better tool-calling than a 14B) | Set `SILKROUTE_OPENROUTER_API_KEY` and `--model deepseek/deepseek-v3.2` (or Claude/GPT/Gemini via OpenRouter — model-agnostic). |

## Security
- `openav-mcp` injects device credentials from `OPENAV_DEVICES` — the **model never sees passwords**
  (it references devices by alias). Run with `--read-only` to expose only status tools.
- SilkRoute API is auth-on-by-default in production (`SILKROUTE_ENVIRONMENT=production`).

## Licenses
Go microservices: GPL-3.0 (matching Dartmouth-OpenAV). Root/demo/`openav-mcp`: see each dir's header
(`openav-mcp` is GPL-3.0-or-later). No proprietary deps.
