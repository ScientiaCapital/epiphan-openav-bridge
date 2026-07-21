# OpenAV Epiphan EC20 Microservice

An OpenAV microservice for controlling Epiphan EC20 PTZ cameras. Built on the
[Dartmouth OpenAV microservice framework](https://github.com/Dartmouth-OpenAV/microservice-framework).

The microservice exposes the usual OpenAV `/:address/:setting` HTTP contract, but internally it speaks
the EC20's **two real, hardware-verified control planes** (the device has no RESTful `/api/*` surface):

- **VISCA-over-IP (raw, TCP `:5678`)** — pan/tilt/zoom, presets, home, position/zoom inquiries.
  Standardized binary protocol; verified against a real EC20 (fw 3.3.40).
- **CGI (`:80`, `param.cgi` / `ptzctrl.cgi`)** — AI tracking + device status. Two auth layers: HTTP
  **Digest** (lighttpd) plus a custom `/cgi-bin/auth.cgi` **session token** (see `source/cgiauth.go`).

> **Two items are CONFIRM-ON-HARDWARE** (isolated to one function each): the exact AI-tracking command
> (`ec20TrackingCommand` in `cgiauth.go`) and the degrees→VISCA-units mapping for absolute PTZ
> (`controlPTZ` calibration constants in `driver.go`). Everything else is verified. See
> `.claude/programs/ec20-api-discovery.md`.

## Requirements

- Epiphan EC20 PTZ Camera (VISCA-over-IP enabled if you use absolute PTZ)
- Network access to the EC20 (TCP `:80` for CGI, TCP `:5678` for VISCA)
- Docker (for deployment)

## Supported Endpoints

### GET (Query)

| Endpoint | Plane | Description |
|----------|-------|-------------|
| `/:address/status` | VISCA | Online + pan/tilt/zoom units |
| `/:address/healthcheck` | VISCA | Device reachability (VISCA version inquiry) → `"true"`/`"false"` |
| `/:address/ptzposition` | VISCA | Current pan/tilt/zoom (raw VISCA units) |
| `/:address/presets` | — | Not supported (VISCA has no list-presets inquiry; recall/save by slot) |
| `/:address/preview` | — | Live stream RTSP URL (`rtsp://<host>:554/1`) |

### PUT (Control)

| Endpoint | Body | Plane | Description |
|----------|------|-------|-------------|
| `/:address/jog/:dir/:speed` | (none) | VISCA | Nudge: `dir` ∈ up/down/left/right/upleft/upright/downleft/downright/stop; `speed` 1-24. **Preferred for live framing** |
| `/:address/preset/:presetId` | (none) | VISCA | Recall preset (0-255) — the primary way to frame known shots |
| `/:address/presetsave/:presetId/:name` | (none) | VISCA | Save current position to a preset slot |
| `/:address/ptzhome` | (none) | VISCA | Return camera to home position |
| `/:address/ptz/:pan/:tilt` | `{"zoom":<float>,"speed":<optional int>}` | VISCA | Absolute PTZ in degrees (secondary; calibration best-effort) |
| `/:address/tracking/:action` | tracking mode | CGI | Enable/disable AI tracking (`presenter`/`zone`) |

### Address Format

Include EC20 credentials in the address (default `admin`/`admin`):
```
admin:password@192.168.1.100
```

## Docker Build & Run

```bash
docker build -t openav-epiphan-ec20 .
docker run -p 8082:80 openav-epiphan-ec20
```

## Usage Examples

```bash
# Status (VISCA units) / health / position
curl http://localhost:8082/admin:admin@192.168.1.100/status
curl http://localhost:8082/admin:admin@192.168.1.100/healthcheck
curl http://localhost:8082/admin:admin@192.168.1.100/ptzposition

# Live preview URL (consume with any RTSP client)
curl http://localhost:8082/admin:admin@192.168.1.100/preview

# Jog: nudge up at speed 10, then stop
curl -X PUT http://localhost:8082/admin:admin@192.168.1.100/jog/up/10
curl -X PUT http://localhost:8082/admin:admin@192.168.1.100/jog/stop/0

# Recall preset 1 / save current position to preset 1 as "Podium"
curl -X PUT http://localhost:8082/admin:admin@192.168.1.100/preset/1
curl -X PUT http://localhost:8082/admin:admin@192.168.1.100/presetsave/1/Podium

# Absolute PTZ (secondary): pan=45°, tilt=-10°, zoom, optional speed
curl -X PUT http://localhost:8082/admin:admin@192.168.1.100/ptz/45/-10 \
  -H "Content-Type: application/json" -d '{"zoom":2.0}'

# Home / AI tracking
curl -X PUT http://localhost:8082/admin:admin@192.168.1.100/ptzhome
curl -X PUT http://localhost:8082/admin:admin@192.168.1.100/tracking/enable \
  -H "Content-Type: application/json" -d '"presenter"'
curl -X PUT http://localhost:8082/admin:admin@192.168.1.100/tracking/disable
```

## Testing

Go unit tests use a fake VISCA-over-TCP device + a mock `auth.cgi`/CGI server (no hardware):
```bash
export PATH="/opt/homebrew/bin:$PATH"
go test ./source/ -v
```

## EC20 Reference

- [EC20 Product Page](https://www.epiphan.com/products/ec20/)
- Control planes reverse-engineered + verified on hardware — see `.claude/programs/ec20-api-discovery.md`.
  Epiphan's own sanctioned integration paths are VISCA-over-IP (`:52381`, if enabled) and ONVIF.

## License

GPL-3.0 — see [LICENSE](LICENSE)
