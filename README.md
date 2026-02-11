# epiphan-openav-bridge

Open-source Go microservices that let [Dartmouth OpenAV](https://github.com/Dartmouth-OpenAV) natively control [Epiphan Pearl](https://www.epiphan.com/products/pearl-mini/) encoders and [EC20 PTZ cameras](https://www.epiphan.com/products/ec20/).

## What This Is

[OpenAV](https://github.com/Dartmouth-OpenAV) is an open-source AV control system built at Dartmouth College. It runs 165+ production rooms using Raspberry Pi orchestrators, Go microservices, and REST APIs to control Sony displays, projectors, DSPs, and cameras.

This project adds Epiphan Pearl and EC20 as first-class OpenAV devices, following the same microservice patterns used by every other device in the ecosystem.

## Components

| Component | Description | Status |
|-----------|-------------|--------|
| **RTSP Proof** | Scripts proving EC20/Pearl streams work with ffmpeg (OpenAV's recording pipeline) | In Progress |
| **Pearl Microservice** | Go service wrapping Pearl REST API v2.0 for OpenAV | Planned |
| **EC20 Microservice** | Go service wrapping EC20 REST API for OpenAV | Planned |
| **Smart Room Demo** | Full pipeline: EC20 tracking → Pearl recording → CMS upload via OpenAV | Planned |

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

```bash
cd openav-epiphan-pearl/
cp .env.example .env  # Edit with your Pearl IP and credentials

go build -o pearl-service .
./pearl-service

# Or with Docker
docker build -t openav-epiphan-pearl .
docker run -e PEARL_HOST=192.168.1.100 -e PEARL_USERNAME=admin -e PEARL_PASSWORD=secret -p 8080:8080 openav-epiphan-pearl
```

### EC20 Microservice (Phase 2)

```bash
cd openav-epiphan-ec20/
cp .env.example .env  # Edit with your EC20 IP and credentials

go build -o ec20-service .
./ec20-service

# Or with Docker
docker build -t openav-epiphan-ec20 .
docker run -e EC20_HOST=192.168.1.50 -e EC20_USERNAME=admin -e EC20_PASSWORD=secret -p 8081:8080 openav-epiphan-ec20
```

## Architecture

```
OpenAV Orchestrator (Raspberry Pi + Docker)
├── Sony Display Microservice (existing)
├── Projector Microservice (existing)
├── DSP Microservice (existing)
├── Pearl Microservice (this project)     ──REST──▶ Pearl Mini
└── EC20 Microservice (this project)      ──REST──▶ EC20 PTZ Camera
```

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

MIT — see [LICENSE](LICENSE) for details.

## Acknowledgments

- [Mark Franklin](https://github.com/Dartmouth-OpenAV) and the Dartmouth team for building OpenAV
- [Epiphan Video](https://www.epiphan.com/) for the Pearl and EC20 product lines and their well-documented REST APIs
