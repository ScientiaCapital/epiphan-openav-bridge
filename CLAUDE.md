# epiphan-openav-bridge

Open-source Go microservices letting Dartmouth's OpenAV control system natively manage Epiphan Pearl encoders and EC20 PTZ cameras.

## Stack

- **Primary**: Go (OpenAV convention — all microservices)
- **Python**: Phase 1 RTSP proof scripts only (`proof/`)
- **Containerization**: Docker (multi-stage builds)
- **License**: GPL-3.0 for microservices (matching Dartmouth-OpenAV repos), MIT for root project

## Directory Structure

```
proof/                    # Phase 1: RTSP compatibility proof (Python)
openav-epiphan-pearl/     # Phase 2: Pearl Go microservice
openav-epiphan-ec20/      # Phase 2: EC20 PTZ Go microservice
demo/                     # Phase 3: docker-compose full-stack demo
openav-mcp/               # Phase 4: MCP server face (Python) — the AI-first layer for agents
.claude/                  # Agent infrastructure (observers, commands, programs)
```

`openav-mcp` is a thin MCP server that exposes the OpenAV orchestrator + the Go microservices as
agent-callable tools. It contains **no LLM dependency itself** — the model lives in the external agent
(e.g. SilkRoute) that consumes this MCP server. So the "No OpenAI" rule below still holds repo-wide.

## Key Commands

```bash
# Phase 2: Go microservices (must export PATH="/opt/homebrew/bin:$PATH" first)
cd openav-epiphan-pearl && go test ./source/ -v
cd openav-epiphan-ec20 && go test ./source/ -v

# Docker builds
docker build -t openav-epiphan-pearl ./openav-epiphan-pearl
docker build -t openav-epiphan-ec20 ./openav-epiphan-ec20

# Phase 3: Full stack
cd demo && docker compose up
```

## Environment Variables

```bash
PEARL_HOST=192.168.x.x
PEARL_USERNAME=admin
PEARL_PASSWORD=your_password
PEARL_PORT=8080
EC20_HOST=192.168.x.x
EC20_USERNAME=admin
EC20_PASSWORD=your_password
SERVICE_PORT=8080
LOG_LEVEL=info
```

## Rules

- **No OpenAI** — no AI/LLM dependencies
- Study Dartmouth-OpenAV repos before writing Go code — follow their patterns exactly
- Each service exposes `/status` and `/health` endpoints
- No Epiphan Cloud — local network REST API only
- No proprietary dependencies (GPL-3.0 compatibility)
- No hardcoded credentials — env vars only

## Decision Rules (Executable Spec)

When encountering these situations, follow the specified protocol:

### Observer Workflow
- **Before writing any code:** Check if observers have run this session. If QUALITY.md still shows "_not yet run_", spawn observer-lite first.
- **When observer finds [BLOCKER]:** Stop work immediately. Disposition the blocker before continuing.
- **When observer finds [WARNING]:** Log to backlog with owner and ETA, then continue.
- **After >5 files modified:** Consider upgrading from observer-lite to observer-full.

### EC20 API Endpoints
- **All 12 EC20 endpoint paths are PLACEHOLDER** — see `.claude/programs/ec20-api-discovery.md`
- **When EC20 hardware is available:** Run the discovery program before writing new EC20 features.
- **When probing an endpoint:** One probe at a time, log result to discovery log, 30s timeout per probe.
- **Metric:** HTTP 200 = confirmed, 4xx = discard, update `driver.go` constant when confirmed.

### Phase Completion
- **When a Phase is complete:** Update ROADMAP.md, archive observer report to `.claude/archive/`, run observer-full as final gate.
- **When starting a new Phase:** Run `/begin` command to sync context and check for blockers.

### Quality Gates
- **Before any commit:** Run `go test ./source/ -v` in both microservice dirs.
- **Before any PR:** Run observer-full, disposition all findings, run `/pr` command.
- **Before session end:** Run `/done` (quick) or `/end` (full) to verify clean state.

## Demand Catalog

Skills, agents, and references are cataloged in `.claude/library.yaml`. Consult the catalog before creating new utilities — reuse existing patterns.

## OpenAV Reference

- Org: https://github.com/Dartmouth-OpenAV
- Architecture wiki: https://github.com/Dartmouth-OpenAV/.github/wiki
- Pearl API reference: see `epiphan-mcp-server` for endpoint mapping (do NOT copy Python patterns)
- Pearl API Swagger: https://epiphan-video.github.io/pearl_api_swagger_ui/
