# epiphan-openav-bridge

Open-source Go microservices letting Dartmouth's OpenAV control system natively manage Epiphan Pearl encoders and EC20 PTZ cameras.

## Stack

- **Primary**: Go (OpenAV convention — all microservices)
- **Python**: Phase 1 RTSP proof scripts only (`proof/`)
- **Containerization**: Docker (multi-stage builds)
- **License**: GPL-3.0 (matching Dartmouth-OpenAV repos)

## Directory Structure

```
proof/                    # Phase 1: RTSP compatibility proof (Python)
openav-epiphan-pearl/     # Phase 2: Pearl Go microservice
openav-epiphan-ec20/      # Phase 2: EC20 PTZ Go microservice
demo/                     # Phase 3: docker-compose full-stack demo
```

## Key Commands

```bash
# Phase 1: Python RTSP proof
cd proof && python rtsp_test.py --ec20-ip 192.168.x.x --pearl-ip 192.168.x.x

# Phase 2: Go microservices
cd openav-epiphan-pearl && go mod tidy && go test ./... && go build -o pearl-service .
cd openav-epiphan-ec20 && go mod tidy && go test ./... && go build -o ec20-service .

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

## OpenAV Reference

- Org: https://github.com/Dartmouth-OpenAV
- Architecture wiki: https://github.com/Dartmouth-OpenAV/.github/wiki
- Pearl API reference: see `epiphan-mcp-server` for endpoint mapping (do NOT copy Python patterns)
