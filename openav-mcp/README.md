# openav-mcp

**The AI-first control layer for "Epiphan hardware running OpenAV."** A small
[Model Context Protocol](https://modelcontextprotocol.io) server that fronts the
OpenAV↔Epiphan REST bridges in this repo, so an LLM agent can drive AV rooms in
plain English — start recording, track the presenter, run a scene — without any
proprietary control programming.

It does **not** replace OpenAV. OpenAV (Dartmouth) stays the brains/control;
Epiphan stays the reliable hardware. This server just exposes the existing REST
surfaces as agent-callable tools. Orchestrate it with any MCP client
([SilkRoute](https://github.com/ScientiaCapital/silkroute), Hermes, OpenClaw,
Claude Desktop, Cursor…).

## Two tool layers

**Scene-level** (via the OpenAV orchestrator `PUT /api/systems/{system}/state`):
- `run_scene(system, scene)` — e.g. `record_session` (tracking + recording)
- `set_room_state(system, control_set, control, value)`
- `list_room_controls(system)` *(read-only)*

**Device-level** (via the Pearl + EC20 OpenAV microservices):
- `pearl_control_recording(device, start|stop)`, `pearl_singletouch`, `pearl_status` *(ro)*
- `ec20_ptz(device, pan, tilt, zoom)`, `ec20_tracking(device, enable|disable, mode)`,
  `ec20_preset_recall(device, id)`, `ec20_status(device)` *(ro)*

## Security
- **The model never sees passwords.** Devices are referenced by *alias*; the server
  resolves `user:pass@host` from config internally.
- Read-only tools are always exported; mutating (control) tools are gated by
  `--read-only` for demo/monitoring deployments.

## Run

```bash
pip install -e ".[dev]"
# Hardware-free demo/dev:
OPENAV_MOCK=true OPENAV_DEVICES='[{"alias":"room-320b-pearl","host":"pearl-host","username":"admin","password":"...","kind":"pearl"}]' \
  python -m openav_mcp
# Live (point at the running bridge + orchestrator):
OPENAV_ORCHESTRATOR_URL=http://localhost:8080 OPENAV_PEARL_URL=http://localhost:8081 \
OPENAV_EC20_URL=http://localhost:8082 OPENAV_DEVICES='[...]' python -m openav_mcp
```

Config env: `OPENAV_ORCHESTRATOR_URL`, `OPENAV_PEARL_URL`, `OPENAV_EC20_URL`,
`OPENAV_DEVICES` (JSON), `OPENAV_MOCK`.

## Test
```bash
pip install -e ".[dev]" && pytest -q
```

Proven end-to-end: SilkRoute's MCP client bridge connects to `python -m openav_mcp`,
discovers all 10 tools, and drives a scene + device tools over MCP (mock mode) with
no credential leakage.
