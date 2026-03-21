# OpenAV Epiphan Pearl Microservice

An OpenAV microservice for controlling Epiphan Pearl encoders via the Pearl REST API v2.0. Built on the [Dartmouth OpenAV microservice framework](https://github.com/Dartmouth-OpenAV/microservice-framework).

## Requirements

- Epiphan Pearl Mini (firmware 4.14.2+)
- Network access to the Pearl device
- Docker (for deployment)

## Supported Endpoints

### GET (Query)

| Endpoint | Description |
|----------|-------------|
| `/:address/status` | Device identity + storage info |
| `/:address/recordingstatus` | All recorder states |
| `/:address/storages` | Storage capacity details |
| `/:address/channels` | Channel listing |
| `/:address/healthcheck` | Device reachability check |

### PUT (Control)

| Endpoint | Body | Description |
|----------|------|-------------|
| `/:address/recording` | `"start"` / `"stop"` | Start/stop all recorders |
| `/:address/streaming/:channelId` | `"start"` / `"stop"` | Start/stop channel streaming |
| `/:address/singletouch` | `"start"` / `"stop"` | Start/stop all recording + streaming |

### Address Format

Include Pearl credentials in the address:
```
admin:password@192.168.1.100
```

## Docker Build & Run

```bash
docker build -t openav-epiphan-pearl .
docker run -p 80:80 openav-epiphan-pearl
```

## Usage Examples

```bash
# Get device status
curl http://localhost/admin:mypass@192.168.1.100/status

# Start recording
curl -X PUT http://localhost/admin:mypass@192.168.1.100/recording \
  -H "Content-Type: application/json" -d '"start"'

# Stop recording
curl -X PUT http://localhost/admin:mypass@192.168.1.100/recording \
  -H "Content-Type: application/json" -d '"stop"'

# Health check
curl http://localhost/admin:mypass@192.168.1.100/healthcheck
```

## Testing

Run the Go unit tests (includes mock Pearl API server):
```bash
go test ./source/ -v
```

## Pearl API Reference

- [Pearl API v2.0 Swagger](https://epiphan-video.github.io/pearl_api_swagger_ui/)
- [Pearl API Guide](https://www.epiphan.com/userguides/pearl-api/)

## License

GPL-3.0 — see [LICENSE](LICENSE)
