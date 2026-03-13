# Coding Rules — epiphan-openav-bridge

## Stack
- **Primary language**: Go (OpenAV convention — all microservices)
- **Python**: Phase 1 RTSP proof scripts only (`proof/`)
- **Containerization**: Docker (multi-stage builds)
- **License**: GPL-3.0 (matching Dartmouth-OpenAV repos)

## Key Patterns
- Study Dartmouth-OpenAV repos before writing any Go code — match their patterns exactly
- Each microservice is a single Go binary in a Docker container
- Standard endpoints every service must expose: `/status`, `/health`
- All config via environment variables — no config files
- Module naming follows: `github.com/scientiacapital/openav-epiphan-<device>`
- Use `openav-epiphan-pearl` as reference for Pearl REST API v2.0 mapping
- Docker Compose in `demo/` for full-stack integration

## Rules
- **No OpenAI** — no AI/LLM dependencies in Go microservices
- Never hardcode device IPs, usernames, or passwords — use env vars
- No Epiphan Cloud/Edge — local network REST API only
- No proprietary dependencies — keep GPL-3.0 compatibility
- Do NOT copy Python patterns from `epiphan-mcp-server` into Go code

## Testing
- Go unit tests: `go test ./...` (in each microservice dir)
- Build check: `go build -o <service-name> .`
- Docker build: `docker build -t openav-epiphan-<device> .`
