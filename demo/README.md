# Smart Room Demo

Phase 3 demo: EC20 AI tracking + Pearl recording, orchestrated by OpenAV.

**Verified 2026-07-18**: the driver URL format in `system-configs/smart-room-demo.json` works correctly
against locally-built (non-GHCR) microservice images — confirmed end-to-end with
`docker compose up --build` + a real `PUT /api/systems/smart-room-demo/state` call. The orchestrator
(via `ADDRESS_MICROSERVICES_BY_NAME=true`) resolves `dartmouth-openav/microservice-epiphan-{pearl,ec20}`
straight to the Compose service names and successfully routes both the recording and PTZ/tracking control
calls into each Go binary's request handler. The only failure at that point is the *microservice → real
device* leg (`pearl-host`/`ec20-host` don't resolve to anything, by design — there's no hardware attached to
this demo). That's expected and correct; the orchestrator-to-microservice wiring itself is confirmed sound.

## Quick Start

```bash
# Build and start all services
docker compose up --build -d

# Verify stack
./test-stack.sh

# View orchestrator UI
open http://localhost:8080

# Shut down
docker compose down
```

## Architecture

```
┌──────────────┐
│ Orchestrator │ :8080 (host)
│   (OpenAV)   │
└──────┬───────┘
       │ Docker DNS
  ┌────┴────┐
  │         │
  ▼         ▼
┌─────┐  ┌─────┐
│Pearl│  │EC20 │
│ :80 │  │ :80 │
└─────┘  └─────┘
```

## Configuration

Edit `system-configs/smart-room-demo.json` to set device credentials and IPs.

The driver URL format follows the OpenAV convention:
```
dartmouth-openav/microservice-name:current/username:password@device-ip/endpoint
```

> ⚠️ **`authorization.json` is `*` (wildcard, allow-all) in this demo stack — dev/local use only.**
> Never deploy this as-is; a real deployment needs a scoped authorization policy.

### Available Endpoints

**Pearl Microservice** (`microservice-epiphan-pearl`):
- GET `/status` - Device info (model, firmware, serial)
- GET `/recordingstatus` - Recording state
- GET `/storages` - Storage capacity info
- GET `/channels` - Channel list
- GET `/healthcheck` - Health status
- PUT `/recording/:action` - Control recording (start/stop)
- PUT `/streaming/:action` - Control streaming (start/stop)
- PUT `/singletouch/:action` - Control recording + streaming together

**EC20 Microservice** (`microservice-epiphan-ec20`):
- GET `/status` - Camera status
- GET `/healthcheck` - Health status
- GET `/ptzposition` - Current PTZ position
- GET `/presets` - List saved presets
- GET `/preview` - Preview image (JPEG)
- PUT `/ptz/:pan/:tilt` - Control PTZ (body: `{"zoom":<float>,"speed":<optional int>}`)
- PUT `/ptzhome` - Return to home position
- PUT `/preset/:presetId` - Recall preset
- PUT `/presetsave/:presetId` - Save preset (body: name)
- PUT `/tracking/:action` - Control AI tracking (body: mode)

## Prerequisites

- Docker and Docker Compose
- Network access to Pearl and EC20 devices
