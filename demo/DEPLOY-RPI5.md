# Deploying OpenAV + Epiphan bridge on a Raspberry Pi 5 (permanent room host)

Stand up the full stack — the OpenAV **orchestrator** + the **Pearl** and **EC20** Go
microservices + the **openav-mcp** AI layer — on a Raspberry Pi 5 (Ubuntu, ARM64), driving the real
devices on the room LAN, auto-starting on boot.

> **Device readiness:** the **Pearl Mini works end-to-end today**. The **EC20** control plane is
> being finished (VISCA preset/jog/position/home are wired + hardware-verified over TCP `:5678`; AI
> **tracking** is CGI-only and needs one live command probe — see the repo's
> `.claude/programs/ec20-api-discovery.md`). Deploy now; EC20 tracking lands with the driver finish.

---

## 0. Prerequisites
- Raspberry Pi 5, **Ubuntu 24.04 LTS (64-bit / arm64)**, on the **same LAN as the devices**.
- Device IPs on that LAN (from the live bring-up): **Pearl `192.168.8.4`**, **EC20 `192.168.8.11`**
  (EC20 default creds `admin`/`admin`; use the Pearl's real admin creds).
- Give the Pi a **static IP or DHCP reservation** (it's the always-on room controller).
- The Pi needs outbound TCP to the devices (`:80` for Pearl/EC20 CGI, `:5678` for EC20 VISCA). A plain
  Docker bridge is fine — containers dial the devices outbound; no host networking needed.

## 1. OS + Docker
```bash
sudo apt update && sudo apt -y upgrade
curl -fsSL https://get.docker.com | sudo sh          # Docker Engine + compose plugin (arm64)
sudo usermod -aG docker "$USER" && newgrp docker      # run docker without sudo
sudo systemctl enable --now docker                    # start on boot
docker run --rm hello-world                           # sanity check
```

## 2. Get the code
```bash
git clone --recurse-submodules <repo-url> epiphan-openav-bridge
cd epiphan-openav-bridge
# If you cloned without submodules: git submodule update --init --recursive
```
(The microservice framework is a git submodule; the Docker build generates its go.mod itself.)

## 3. Orchestrator architecture check (the one ARM64 gotcha)
The Go microservices build **natively** on arm64. The orchestrator is a prebuilt image — check it:
```bash
docker manifest inspect ghcr.io/dartmouth-openav/orchestrator:latest | grep -A2 architecture
```
- **arm64 listed** → nothing to do, it pulls native.
- **amd64 only** → enable emulation, then pin the platform:
  ```bash
  docker run --privileged --rm tonistiigi/binfmt --install all
  ```
  and uncomment `platform: linux/amd64` under `orchestrator:` in `demo/docker-compose.override.yml`.
  (Emulated, but fine for a PHP control plane.)

## 4. Configure the devices
```bash
cd demo
cp .env.example .env
# Edit .env — real IPs + creds:
#   PEARL_HOST=192.168.8.4   PEARL_USERNAME=admin  PEARL_PASSWORD=<pearl pw>
#   EC20_HOST=192.168.8.11   EC20_USERNAME=admin   EC20_PASSWORD=admin
./generate-config.sh          # writes system-configs/smart-room-demo.json (GNU-sed safe)
```

## 5. Bring up the stack
```bash
# from demo/ — docker-compose.override.yml is auto-merged (publishes :8081/:8082 + restart policies)
docker compose up --build -d
docker compose ps             # orchestrator + both microservices "Up"
./test-stack.sh               # containers up + orchestrator reachable on :8080
```

## 6. Verify against the real devices
**Pearl (works today)** — drive a recording through the orchestrator:
```bash
curl -X PUT http://localhost:8080/api/systems/smart-room-demo/state \
  -H 'Content-Type: application/json' \
  -d '{"control_sets":{"recording":{"controls":{"record":{"value":true}}}}}'
# ...verify the Pearl is recording, then stop:
curl -X PUT http://localhost:8080/api/systems/smart-room-demo/state \
  -H 'Content-Type: application/json' \
  -d '{"control_sets":{"recording":{"controls":{"record":{"value":false}}}}}'
```
Or hit the microservice directly: `curl http://localhost:8081/admin:<pw>@192.168.8.4/healthcheck` → `"true"`.

**EC20 (VISCA now; tracking after driver finish)** — direct microservice calls:
```bash
curl http://localhost:8082/admin:admin@192.168.8.11/healthcheck        # VISCA version probe -> "true"
curl -X PUT http://localhost:8082/admin:admin@192.168.8.11/preset/1    # camera physically moves
curl http://localhost:8082/admin:admin@192.168.8.11/ptzposition        # {pan_units,tilt_units,zoom_units}
```

## 7. openav-mcp (the AI layer)
```bash
cd ../openav-mcp
python3 -m venv .venv && source .venv/bin/activate
pip install -e .
export OPENAV_ORCHESTRATOR_URL=http://localhost:8080
export OPENAV_PEARL_URL=http://localhost:8081
export OPENAV_EC20_URL=http://localhost:8082
export OPENAV_DEVICES='[
  {"alias":"room-pearl","host":"192.168.8.4","username":"admin","password":"<pearl pw>","kind":"pearl"},
  {"alias":"room-cam","host":"192.168.8.11","username":"admin","password":"admin","kind":"ec20"}]'
# (do NOT set OPENAV_MOCK — that forces mock mode)
python -m openav_mcp
```
The model references devices by **alias** (`room-pearl`, `room-cam`); openav-mcp injects the creds, so
passwords never reach the model.

> **How does your agent connect?** openav-mcp speaks MCP over **stdio**. If SilkRoute (or whatever
> agent) *spawns* it, no daemon is needed — it launches it with the env above. If instead you need it
> resident, wrap it in the systemd unit in step 8.

## 8. Autostart on boot (permanent host)
- **Stack:** already handled — `docker-compose.override.yml` sets `restart: unless-stopped` on all
  three services, and Docker is enabled (step 1). They return after a reboot.
- **openav-mcp** (only if it must run resident, not agent-spawned): create
  `/etc/systemd/system/openav-mcp.service`:
  ```ini
  [Unit]
  Description=OpenAV MCP server
  After=docker.service
  Requires=docker.service
  [Service]
  WorkingDirectory=/home/<user>/epiphan-openav-bridge/openav-mcp
  EnvironmentFile=/home/<user>/epiphan-openav-bridge/openav-mcp/.env
  ExecStart=/home/<user>/epiphan-openav-bridge/openav-mcp/.venv/bin/python -m openav_mcp
  Restart=on-failure
  [Install]
  WantedBy=multi-user.target
  ```
  then `sudo systemctl enable --now openav-mcp`.
- **Verify reboot survival:** `sudo reboot`, then after it's back: `docker compose ps` (from `demo/`)
  shows all three Up, and `./test-stack.sh` passes.

---

## Troubleshooting
- **`orchestrator` exits / "exec format error"** → it's amd64-only; do step 3's binfmt + platform pin.
- **Microservice can't reach a device (dial timeout)** → confirm the Pi is on `192.168.8.x` and can
  `ping`/`curl` the device IP directly; check the device is powered and on-network.
- **EC20 calls hang** → the CGI/lighttpd plane can wedge under repeated hung requests; power-cycle the
  camera. (VISCA on `:5678` is unaffected.)
- **`generate-config.sh` no-ops** → ensure `.env` is populated; re-run; it's idempotent.
- **Pearl 401** → wrong creds in `.env`; regenerate the config.
