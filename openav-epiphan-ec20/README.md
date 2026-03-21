# OpenAV Epiphan EC20 Microservice

An OpenAV microservice for controlling Epiphan EC20 PTZ cameras via REST API. Built on the [Dartmouth OpenAV microservice framework](https://github.com/Dartmouth-OpenAV/microservice-framework).

> **WARNING: Placeholder API Endpoints**
> The EC20 REST API is not publicly documented. All API endpoint paths in this microservice are placeholders.
> They must be verified and updated when testing against real hardware.
> See the endpoint constants in `source/driver.go` for the full list.

## Requirements

- Epiphan EC20 PTZ Camera
- Network access to the EC20 device
- Docker (for deployment)

## Supported Endpoints

### GET (Query)

| Endpoint | Description |
|----------|-------------|
| `/:address/status` | Camera status (model, firmware, tracking, position) |
| `/:address/healthcheck` | Device reachability check |
| `/:address/ptzposition` | Current pan/tilt/zoom position |
| `/:address/presets` | List saved presets |
| `/:address/preview` | Camera preview (base64 JPEG) |

### PUT (Control)

| Endpoint | Body | Description |
|----------|------|-------------|
| `/:address/ptz/:pan/:tilt` | zoom level (float) | Move to absolute PTZ position |
| `/:address/ptzhome` | (none) | Return camera to home position |
| `/:address/preset/:presetId` | (none) | Recall a saved preset |
| `/:address/presetsave/:presetId` | preset name | Save current position as preset |
| `/:address/tracking/:action` | tracking mode | Enable/disable AI tracking |

### Address Format

Include EC20 credentials in the address:
```
admin:password@192.168.1.100
```

## Docker Build & Run

```bash
docker build -t openav-epiphan-ec20 .
docker run -p 80:80 openav-epiphan-ec20
```

## Usage Examples

```bash
# Get camera status
curl http://localhost/admin:mypass@192.168.1.100/status

# Get PTZ position
curl http://localhost/admin:mypass@192.168.1.100/ptzposition

# Get presets
curl http://localhost/admin:mypass@192.168.1.100/presets

# Move camera (pan=45, tilt=-10, zoom=2.0)
curl -X PUT http://localhost/admin:mypass@192.168.1.100/ptz/45/-10 \
  -H "Content-Type: application/json" -d '"2.0"'

# Return to home position
curl -X PUT http://localhost/admin:mypass@192.168.1.100/ptzhome

# Recall preset 1
curl -X PUT http://localhost/admin:mypass@192.168.1.100/preset/1

# Save preset 1 as "Center"
curl -X PUT http://localhost/admin:mypass@192.168.1.100/presetsave/1 \
  -H "Content-Type: application/json" -d '"Center"'

# Enable presenter tracking
curl -X PUT http://localhost/admin:mypass@192.168.1.100/tracking/enable \
  -H "Content-Type: application/json" -d '"presenter"'

# Disable tracking
curl -X PUT http://localhost/admin:mypass@192.168.1.100/tracking/disable

# Health check
curl http://localhost/admin:mypass@192.168.1.100/healthcheck
```

## Testing

Run the Go unit tests (includes mock EC20 API server):
```bash
go test ./source/ -v
```

## EC20 Reference

- [EC20 Product Page](https://www.epiphan.com/products/ec20/)
- No public API documentation available — endpoints are placeholders

## License

GPL-3.0 — see [LICENSE](LICENSE)
