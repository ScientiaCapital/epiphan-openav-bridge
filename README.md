# epiphan-openav-bridge

Open-source Go microservices that let [Dartmouth OpenAV](https://github.com/Dartmouth-OpenAV) natively control [Epiphan Pearl](https://www.epiphan.com/products/pearl-mini/) encoders and [EC20 PTZ cameras](https://www.epiphan.com/products/ec20/).

## What This Is

[OpenAV](https://github.com/Dartmouth-OpenAV) is an open-source AV control system built at Dartmouth College. It runs 165+ production rooms using Raspberry Pi orchestrators, Go microservices, and REST APIs to control Sony displays, projectors, DSPs, and cameras.

This project adds Epiphan Pearl and EC20 as first-class OpenAV devices, following the same microservice patterns used by every other device in the ecosystem.

## Components

| Component | Description | Status |
|-----------|-------------|--------|
| **RTSP Proof** | Scripts proving EC20/Pearl streams work with ffmpeg (OpenAV's recording pipeline) | In Progress |
| **Pearl Microservice** | Go service wrapping Pearl REST API v2.0 for OpenAV | ✅ Built — 46 tests (mock-verified) |
| **EC20 Microservice** | Go service wrapping EC20 REST API for OpenAV | ⚠️ Built — 80 tests, but REST endpoints are **PLACEHOLDER** pending hardware |
| **Smart Room Demo** | Full pipeline (EC20 tracking → Pearl recording) via the OpenAV orchestrator | ✅ Demo stack present (`demo/docker-compose.yml`) |
| **openav-mcp** | Python MCP server fronting the above for LLM agents (the AI-first layer) | ✅ Built — 11 tests, SilkRoute round-trip verified |

> **Handoff:** to run the whole agentic stack end-to-end with no hardware, see **[`HANDOFF.md`](HANDOFF.md)**.

## Motivation

OpenAV currently uses Magewell HDMI-to-RTSP converters for video capture. They work, but they're passive converters with no intelligence. Adding Epiphan hardware as native OpenAV devices enables:

- **AI auto-tracking** (EC20) — presenter following without manual camera operation
- **On-device backup recording** (Pearl) — local recording continues if the ffmpeg server goes down
- **Native CMS scheduling** (Pearl) — Panopto/Kaltura/Opencast events auto-start/stop
- **Multi-source recording** (Pearl) — camera + screen + document cam captured simultaneously
- **Full REST API control** — same pattern OpenAV uses for every other device

Both device families expose well-documented REST APIs, making them straightforward to wrap in OpenAV's Go microservice convention.

## Quick Start

### RTSP Compatibility Proof (Phase 1)

```bash
cd proof/
pip install -r requirements.txt

# Test EC20 RTSP stream
python rtsp_test.py --ec20-ip 192.168.1.50

# Test Pearl RTSP stream
python rtsp_test.py --pearl-ip 192.168.1.100

# Test both + compare
python rtsp_test.py --ec20-ip 192.168.1.50 --pearl-ip 192.168.1.100
```

### Pearl Microservice (Phase 2)

These are **stateless** OpenAV microservices — they take no device config. Credentials are supplied
**per request** in the URL path (`user:pass@host`, the OpenAV `socketKey` convention), so there is no
`.env` to set up. The service listens on **:80** inside the container.

```bash
cd openav-epiphan-pearl/
go build -o pearl-service ./source/   # source lives in ./source/
./pearl-service                       # binds :80

# Or with Docker (container :80 → host 8081)
docker build -t openav-epiphan-pearl .
docker run -p 8081:80 openav-epiphan-pearl

# Call it (creds embedded in the path):
curl http://localhost:8081/admin:password@192.168.1.100/status
curl -X PUT http://localhost:8081/admin:password@192.168.1.100/recording \
     -H 'content-type: application/json' -d '"start"'
```

### EC20 Microservice (Phase 2)

> ⚠️ **Pre-hardware:** the EC20 REST endpoint paths in `source/driver.go` are PLACEHOLDER and must be
> verified against a real EC20 before production use (see `.claude/programs/ec20-api-discovery.md`).

```bash
cd openav-epiphan-ec20/
go build -o ec20-service ./source/
./ec20-service                        # binds :80

# Or with Docker (container :80 → host 8082)
docker build -t openav-epiphan-ec20 .
docker run -p 8082:80 openav-epiphan-ec20

curl http://localhost:8082/admin:password@192.168.1.50/status
```

## Architecture

```
LLM agent (SilkRoute / Hermes / OpenClaw / Claude Desktop)   ← AI-first layer
        │  plain English, over MCP
        ▼
openav-mcp  (this repo — MCP face; scene + device tools)     ← this project adds
        │  REST
        ▼
OpenAV Orchestrator (Raspberry Pi + Docker)                  ← Dartmouth OpenAV (the brains)
├── Sony Display / Projector / DSP Microservices (existing)
├── Pearl Microservice (this project)     ──REST──▶ Pearl encoder   ← Epiphan (the reliable iron)
└── EC20 Microservice (this project)      ──REST──▶ EC20 PTZ Camera
```

**Positioning:** OpenAV is the brains/control; Epiphan is the reliable hardware; the agent is the
backbone *above* OpenAV. These stay separate — "Epiphan hardware running OpenAV," never "Epiphan
OpenAV." `openav-mcp` doesn't replace OpenAV; it just exposes the existing REST surfaces as
agent-callable MCP tools. See [`openav-mcp/`](openav-mcp/) and [`HANDOFF.md`](HANDOFF.md).

Each microservice follows OpenAV conventions:
- Single Go binary in Docker container
- REST API with `/status`, `/health`, and device-specific endpoints
- Environment variables for all configuration
- Stdout logging (Docker collects)

## Supported Hardware

| Device | Connection | Capabilities via OpenAV |
|--------|-----------|------------------------|
| Pearl Nano | REST API v2.0 | Recording, streaming, layout switching, CMS events |
| Pearl Mini | REST API v2.0 | Recording, streaming, layout switching, CMS events |
| Pearl Nexus | REST API v2.0 | Recording, streaming, layout switching, CMS events |
| Pearl-2 | REST API v2.0 | Recording, streaming, layout switching, CMS events |
| EC20 PTZ Camera | REST API | Pan/tilt/zoom, presets, AI tracking, preview |

All Pearl models share the same REST API v2.0.

## Related Projects

- [Dartmouth OpenAV](https://github.com/Dartmouth-OpenAV) — The open-source AV control system this integrates with
- [Pearl REST API Documentation](https://epiphan-video.github.io/pearl_api_swagger_ui/) — Official Swagger docs

## Contributing

Contributions welcome. If you're running OpenAV at your institution and want to test with Epiphan hardware, open an issue.

## License

This project uses a dual-license structure:

- **Root project** (proof scripts, demo, docs): MIT — see [LICENSE](LICENSE)
- **Go microservices** (openav-epiphan-pearl, openav-epiphan-ec20): GPL-3.0 — matching the [Dartmouth-OpenAV](https://github.com/Dartmouth-OpenAV) framework they integrate with. See each microservice's `LICENSE` file.

## Acknowledgments

- [Mark Franklin](https://github.com/Dartmouth-OpenAV) and the Dartmouth team for building OpenAV
- [Epiphan Video](https://www.epiphan.com/) for the Pearl and EC20 product lines and their well-documented REST APIs
