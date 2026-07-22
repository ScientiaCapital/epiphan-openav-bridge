# Onboarding — epiphan-openav-bridge

**What it is:** an AI-first control plane for AV rooms. An LLM agent takes plain English
("record room 320-B with the camera tracking the presenter") and drives **Dartmouth OpenAV** +
Epiphan **Pearl / EC20** over **MCP** — on-prem, model-agnostic, no proprietary control programming.
*Positioning: "Epiphan hardware running OpenAV," never "Epiphan OpenAV."*

Full engineer detail lives in **[`HANDOFF.md`](HANDOFF.md)** — this is the 2-minute orientation.

## Get productive in 2 minutes (no hardware)
```bash
cd openav-mcp && python3 -m venv .venv && source .venv/bin/activate && pip install -e ".[dev]"
pytest -q                        # → 39 passed
python scripts/roundtrip_demo.py # → discovers 12 tools, runs a scene, "ROUND-TRIP OK"
```
Go microservices (need `export PATH="/opt/homebrew/bin:$PATH"` on the Mac dev box):
```bash
cd openav-epiphan-ec20   && go test ./source/   # → 70 pass  (fresh clone: bash ./init-framework-mod.sh first)
cd openav-epiphan-pearl  && go test ./source/   # → green
```

## Where things stand (2026-07-22)
- **Shipped to `main`.** The **EC20 hybrid driver is complete** — redesigned onto the device's real
  control planes (the old RESTful `/api/*` model was wrong): **VISCA-over-IP (raw, TCP `:5678`)** for
  PTZ/presets/home/jog/position, **CGI** (`/cgi-bin/auth.cgi` session + `ptzctrl.cgi`/`param.cgi`) for
  AI tracking + status. `openav-mcp` exposes `ec20_jog` + `ec20_preset_save`. All suites green, DA-audited.
- **Next sprint = live hardware bring-up.** Everything is code-complete; what remains needs the devices
  on the `192.168.8.x` LAN. **Read [`HANDOFF.md` → "Next Sprint"](HANDOFF.md)** — it has the exact steps and
  the two isolated CONFIRM-ON-HARDWARE spots.

## Repo map
| Path | What |
|---|---|
| `openav-epiphan-ec20/` | EC20 Go microservice (VISCA + CGI). `source/{driver,visca,cgiauth,microservice}.go` |
| `openav-epiphan-pearl/` | Pearl Go microservice (real REST API v2.0) |
| `openav-mcp/` | Python MCP server — the agent-facing tool layer |
| `demo/` | docker-compose stack + **[`DEPLOY-RPI5.md`](demo/DEPLOY-RPI5.md)** (Raspberry Pi 5 deploy) |
| `.claude/programs/ec20-api-discovery.md` | The reverse-engineered EC20 API contract (ground truth) |
| `.claude/observers/QUALITY.md` | End-of-sprint audit + backlog |

## Gotchas worth knowing up front
- **EC20**: `admin`/`admin`; HTTP **Digest** (not Basic); VISCA is **raw over TCP `:5678`** (not Sony UDP
  `:52381`, not CGI). Its CGI/lighttpd plane can **wedge under repeated hung requests — power-cycle clears it**.
- **Device IPs** (current lab): EC20 `192.168.8.11`, Pearl Mini `192.168.8.4` — same subnet; the dev box
  must be **on the `192.168.8.x` LAN** to reach them.
- The `source/microservice-framework/` submodule ships no `go.mod` (upstream gitignores it) — run
  `bash ./init-framework-mod.sh` in a service dir before `go test`/`go build` on a fresh clone.

## Conventions
- **TDD** (red → green); keep `go vet` + `gofmt` + `go test` and `pytest` + `ruff` green before every commit.
- **No OpenAI / no LLM deps** in the services (the model lives in the external agent). GPL-3.0 (services).
- Credentials are per-request in the URL path (`user:pass@host`) — never hardcoded; the model sees only aliases.
