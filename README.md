<p align="center">
  <a href="https://github.com/ScientiaCapital/epiphan-openav-bridge/actions/workflows/ci.yml"><img alt="ci" src="https://github.com/ScientiaCapital/epiphan-openav-bridge/actions/workflows/ci.yml/badge.svg"></a>
  <img alt="Go 1.27" src="https://img.shields.io/badge/go-1.27-00add8">
  <img alt="Python 3.11+" src="https://img.shields.io/badge/python-3.11%2B-blue">
  <img alt="Status: bench-tested" src="https://img.shields.io/badge/status-bench--tested-lightgrey">
</p>

<p align="center">
  <a href="#what-you-get">What you get</a> ·
  <a href="#quick-start">Quick start</a> ·
  <a href="#how-it-fits-together">How it fits together</a> ·
  <a href="#status">Status</a> ·
  <a href="SECURITY.md">Security</a>
</p>

# epiphan-openav-bridge

Two Go microservices that let [Dartmouth OpenAV](https://github.com/Dartmouth-OpenAV) control Epiphan Pearl
encoders and the EC20 PTZ camera, built on OpenAV's own microservice framework, plus `openav-mcp`, a small
Model Context Protocol (MCP) server that exposes the same controls to an AI agent.

## What you get

| Component | What it does | Tests |
|---|---|---|
| `openav-epiphan-pearl` | Calls the Pearl REST API v2.0: device and channel info, recorder start, stop and status, publisher (stream) start and stop, single-touch, storage | 67 passing |
| `openav-epiphan-ec20` | Drives the EC20 over raw VISCA on TCP 5678 for pan, tilt, zoom, presets, home, jog, position and health, and over the camera's CGI session layer for AI tracking | 90 passing |
| `openav-mcp` | MCP server with 12 tools, 3 of them read-only. `--read-only` hides and refuses the 9 control tools | 42 passing |
| `proof/` | `rtsp_test.py`, which checks that a device's RTSP stream can be read by ffmpeg | needs a device |
| `demo/` | A Docker Compose stack of the OpenAV orchestrator and both services, with a Raspberry Pi 5 guide; `openav-mcp` runs on the same host | no recorded end-to-end run |

## Quick start

The services hold no configuration. Following the OpenAV `socketKey` convention, each request carries the
device's credentials and address in its URL path. Each container listens on port 80.

```bash
git clone --recurse-submodules https://github.com/ScientiaCapital/epiphan-openav-bridge.git
cd epiphan-openav-bridge/openav-epiphan-pearl
bash ./init-framework-mod.sh        # the framework submodule ships without go.mod
go test ./source/                   # offline, against mocks

docker build -t openav-epiphan-pearl .
docker run --rm -p 127.0.0.1:8081:80 openav-epiphan-pearl
curl http://localhost:8081/admin:<password>@192.0.2.100/status
```

OpenAV's framework answers from a cache it refreshes in the background, so the first request for each endpoint
of a device returns `204 No Content` and later requests return the cached value; `GET /<address>/errors` shows
what the driver ran into. The EC20 service is the same with `openav-epiphan-ec20` and port 8082. For the MCP server:

```bash
cd openav-mcp
uv sync --extra dev
uv run python -m pytest -q
uv run python -m openav_mcp --read-only    # stdio MCP server, monitoring tools only
```

## How it fits together

```
AI agent (any MCP client)
    |  MCP over stdio
openav-mcp  ------------------------------------------------- this repo
    |  HTTP: room and scene tools          |  HTTP: device tools
OpenAV orchestrator  (Dartmouth)           |
    |                                      |
    +-- openav-epiphan-pearl  <------------+  --HTTP------> Pearl (REST API v2.0)
    +-- openav-epiphan-ec20   <------------+  --TCP 5678--> EC20 (raw VISCA)
                                              --HTTP------> EC20 (CGI, AI tracking)
```

OpenAV stays in charge of the room. The services add Epiphan devices the way OpenAV adds any other device.
`openav-mcp` sends room and scene requests to the orchestrator and device requests to the services directly.

## Design decisions

- **Follow OpenAV's conventions.** Both services are built on Dartmouth's `microservice-framework` (a git
  submodule) and take credentials per request, as every OpenAV device service does, so they drop into an
  existing OpenAV deployment.
- **Use the ports Epiphan documents and the camera answers on.** Epiphan's EC20 network settings list "TCP Port:
  Default value 5678" and "Sony Visca: Uses port 52381, cannot be changed"
  ([kb: ec20/ec20-network-settings](https://kb.epiphan.com/ec20/ec20-network-settings)). Per the bench notes in
  `openav-epiphan-ec20/source/visca.go`, raw VISCA on TCP 5678 answered and 52381 did not, so the driver uses
  TCP 5678.
  AI tracking goes through the CGI session layer (`source/cgiauth.go`), reverse-engineered from the camera's web
  UI. The working notes are in [docs/ec20-protocol-notes.md](docs/ec20-protocol-notes.md).
- **One Pearl service for the family.** Epiphan documents the REST API v2.0 as applying to "Pearl-2, Pearl Mini,
  Pearl Nexus, Pearl Nano"
  ([kb: pearl-family/pearl-rest-api-base-url-authentication-and-versioning](https://kb.epiphan.com/pearl-family/pearl-rest-api-base-url-authentication-and-versioning)).
- **Login tokens are tied to the credentials that earned them.** The EC20 CGI token cache is keyed by host plus a
  SHA-256 of the credentials, expires after 10 minutes, and holds at most 64 entries (`source/cgiauth.go`).
- **Control can be switched off at the boundary, and agents never see a password.** `--read-only` hides and
  refuses the control tools (`openav-mcp/openav_mcp/server.py`). Devices are addressed by alias and their
  credentials come from `OPENAV_DEVICES` (`openav_mcp/config.py`).
- **Small, pinned, unprivileged images.** Static Go binaries built with Go 1.27.4 and run on Alpine 3.23, every
  base image pinned by digest, running as uid 10001. Our own log lines mask passwords (`source/redact.go`).

## Status

- **Bench-tested on an EC20 (firmware 3.3.40, per the notes in `visca.go`):** the VISCA transport and its
  commands answered as expected. The absolute pan and tilt scale and the zoom range are not calibrated yet;
  `driver.go` marks both.
- **Tested against mocks only:** the Pearl service, the EC20 CGI tracking path, and `openav-mcp`. No run against
  a live Pearl is recorded.
- **Built and run in Docker:** all three images build and run as uid 10001. The two Go images pass their health
  checks; `openav-mcp` is a stdio server and has none.
- **Not yet done:** a recorded end-to-end run of the `demo/` stack with both devices.

## Data handling

Tests use mocks and documentation addresses (192.0.2.0/24) and make no network calls. Device credentials live in
the caller's request or in `OPENAV_DEVICES`. The current tree uses only documentation addresses and placeholder
credentials; earlier commits contain bench addresses and the EC20 factory default login. See [SECURITY.md](SECURITY.md) for what reaches logs.

## Development

CI runs `go vet`, the Go tests and govulncheck for both services; ruff, pytest, pip-audit and bandit for
`openav-mcp`; a Docker build of all three images; and a gitleaks scan of the full history. Dependabot proposes
Go, Python, Docker and Actions updates weekly. See [CONTRIBUTING.md](CONTRIBUTING.md) and
[ONBOARDING.md](ONBOARDING.md).

## License

- Root (proof scripts, demo, docs): MIT, see [LICENSE](LICENSE).
- `openav-epiphan-pearl` and `openav-epiphan-ec20`: GPL-3.0, matching the Dartmouth OpenAV framework they are
  built on. See each service's `LICENSE`.
- `openav-mcp`: GPL-3.0, see [openav-mcp/LICENSE](openav-mcp/LICENSE).

Epiphan, Pearl and EC20 are trademarks of Epiphan Systems Inc. OpenAV is a Dartmouth College project. This is an
independent project, not affiliated with or supported by either.
