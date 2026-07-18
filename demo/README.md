# Smart Room Demo

Phase 3 demo: EC20 AI tracking + Pearl recording, orchestrated by OpenAV.

**Note**: Driver URLs in `system-configs/smart-room-demo.json` are PLACEHOLDERS. The exact format for local builds (vs GHCR images) needs verification against the orchestrator. The current URLs follow the OpenAV convention but may need adjustment based on how the orchestrator resolves container names vs image references.

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
