"""OpenAVClient — thin async client over the OpenAV orchestrator + device services.

Two layers:
  * scene-level → PUT {orchestrator}/api/systems/{system}/state (partial state tree)
  * device-level → PUT/GET the Pearl + EC20 OpenAV microservices

In ``mock`` mode nothing hits the network: calls are recorded/echoed in memory so
the whole control plane runs hardware-free (for demos + tests). Credentials are
resolved from config internally and never returned to callers.
"""

from __future__ import annotations

from typing import Any

from openav_mcp.config import DeviceKind, OpenAVConfig

# Named multi-step scenes for run_scene(). Each step = (control_set, control, value).
SCENE_RECIPES: dict[str, list[tuple[str, str, Any]]] = {
    "record_session": [("camera", "tracking", True), ("recording", "record", True)],
    "stop_session": [("recording", "record", False), ("camera", "tracking", False)],
}


class DeviceRequestError(RuntimeError):
    """A device/orchestrator HTTP call failed. Message is always credential-safe.

    Never built from the raw httpx exception's ``str()`` — that embeds the request
    URL, which embeds ``user:pass@host`` for device calls (see ``_resolve_address``).
    """


class OpenAVClient:
    def __init__(self, config: OpenAVConfig) -> None:
        self.config = config
        # Mock state: last state tree PUT, and per-device action log.
        self._mock_state: dict[str, Any] = {}
        self._mock_devices: dict[str, dict[str, Any]] = {}
        self._transport: Any = None  # test-only seam for httpx.MockTransport

    # -- internal ---------------------------------------------------------
    def _resolve_address(self, alias: str) -> str:
        """alias → ``user:pass@host`` (INTERNAL; embeds password, never returned)."""
        return self.config.device(alias).address

    def _require_kind(self, device: str, expected: DeviceKind) -> None:
        """Validate the alias exists AND is configured as the expected device kind —
        nothing else stops a caller from e.g. sending ec20_ptz to a device configured
        kind="pearl"."""
        cfg = self.config.device(device)
        if cfg.kind != expected:
            raise ValueError(f"device '{device}' is kind={cfg.kind!r}, expected {expected!r}")

    @staticmethod
    def _state_tree(control_set: str, control: str, value: Any) -> dict[str, Any]:
        return {"control_sets": {control_set: {"controls": {control: {"value": value}}}}}

    async def _request(self, method: str, url: str, *, json_body: Any = None) -> str:
        """Issue an HTTP call, translating any failure into a credential-safe error.

        ``httpx``'s exception messages embed the full request URL, which for device
        calls embeds ``user:pass@host`` — never let that string reach a caller
        (server.py forwards uncaught exception text straight to the model).
        """
        import httpx

        try:
            async with httpx.AsyncClient(timeout=10, transport=self._transport) as client:
                resp = await client.request(method, url, json=json_body)
                resp.raise_for_status()
                return resp.text
        except httpx.HTTPStatusError as exc:
            raise DeviceRequestError(
                f"device request failed: HTTP {exc.response.status_code}"
            ) from None
        except httpx.HTTPError as exc:
            raise DeviceRequestError(f"device request failed: {type(exc).__name__}") from None

    async def _put(self, url: str, json_body: Any) -> Any:
        return await self._request("PUT", url, json_body=json_body)

    async def _device_put(self, service_url: str, alias: str, path: str, body: Any) -> Any:
        addr = self._resolve_address(alias)
        return await self._put(f"{service_url}/{addr}/{path}", body)

    # -- scene layer ------------------------------------------------------
    async def set_room_state(
        self, system: str, control_set: str, control: str, value: Any
    ) -> dict[str, Any]:
        """Set one control on a room via the OpenAV orchestrator state API."""
        tree = self._state_tree(control_set, control, value)
        if self.config.mock:
            self._mock_state = tree
            return tree
        url = f"{self.config.orchestrator_url}/api/systems/{system}/state"
        await self._put(url, tree)
        return tree

    async def run_scene(self, system: str, scene: str) -> dict[str, Any]:
        """Run a named multi-step scene (sequence of set_room_state calls)."""
        if scene not in SCENE_RECIPES:
            known = ", ".join(sorted(SCENE_RECIPES))
            raise KeyError(f"Unknown scene '{scene}'. Known scenes: {known}")
        steps = []
        for control_set, control, value in SCENE_RECIPES[scene]:
            await self.set_room_state(system, control_set, control, value)
            steps.append({"control_set": control_set, "control": control, "value": value})
        return {"ok": True, "system": system, "scene": scene, "steps": steps}

    def list_room_controls(self, system: str) -> dict[str, Any]:
        """List the scene recipes + control-set vocabulary agents can drive."""
        return {
            "system": system,
            "scenes": sorted(SCENE_RECIPES),
            "control_sets": {
                "recording": ["record", "streaming"],
                "camera": ["tracking", "ptz_home"],
            },
        }

    # -- device layer: Pearl ---------------------------------------------
    async def pearl_control_recording(self, device: str, action: str) -> dict[str, Any]:
        if action not in {"start", "stop"}:
            raise ValueError("action must be 'start' or 'stop'")
        self._require_kind(device, "pearl")
        if self.config.mock:
            self._mock_devices.setdefault(device, {})["recording"] = action
        else:
            await self._device_put(self.config.pearl_service_url, device, "recording", action)
        return {"device": device, "action": action, "ok": True}

    async def pearl_singletouch(self, device: str, action: str) -> dict[str, Any]:
        if action not in {"start", "stop"}:
            raise ValueError("action must be 'start' or 'stop'")
        self._require_kind(device, "pearl")
        if not self.config.mock:
            await self._device_put(self.config.pearl_service_url, device, "singletouch", action)
        else:
            # Singletouch starts/stops recording+streaming together — reflect that in the
            # tracked mock state so a subsequent pearl_status() shows it, same as
            # pearl_control_recording.
            self._mock_devices.setdefault(device, {})["recording"] = action
        return {"device": device, "action": action, "ok": True}

    async def pearl_status(self, device: str) -> dict[str, Any]:
        self._require_kind(device, "pearl")
        if self.config.mock:
            rec = self._mock_devices.get(device, {}).get("recording", "stopped")
            return {"device": device, "ok": True, "status": {"recording": rec, "state": "online"}}
        addr = self._resolve_address(device)
        raw = await self._request("GET", f"{self.config.pearl_service_url}/{addr}/status")
        return {"device": device, "ok": True, "status": raw}

    # -- device layer: EC20 ----------------------------------------------
    async def ec20_ptz(
        self, device: str, pan: float, tilt: float, zoom: float, speed: int = 50
    ) -> dict[str, Any]:
        self._require_kind(device, "ec20")
        # Keep in sync with openav-epiphan-ec20/source/driver.go controlPTZ (DOC-CONFIRMED
        # physical limits) — validated in both mock and live mode so an agent can't learn
        # a range in mock that then fails against real hardware. speed has no documented
        # range, only guard against non-positive values (matches the Go driver).
        if not -162.5 <= pan <= 162.5:
            raise ValueError("pan must be -162.5..162.5")
        if not -30 <= tilt <= 90:
            raise ValueError("tilt must be -30..90")
        if speed <= 0:
            raise ValueError("speed must be positive")
        if not self.config.mock:
            await self._device_put(
                self.config.ec20_service_url,
                device,
                f"ptz/{pan}/{tilt}",
                {"zoom": zoom, "speed": speed},
            )
        else:
            self._mock_devices.setdefault(device, {})["ptz"] = [pan, tilt, zoom, speed]
        return {
            "device": device,
            "pan": pan,
            "tilt": tilt,
            "zoom": zoom,
            "speed": speed,
            "ok": True,
        }

    async def ec20_tracking(self, device: str, action: str, mode: str = "presenter") -> dict[str, Any]:
        if action not in {"enable", "disable"}:
            raise ValueError("action must be 'enable' or 'disable'")
        self._require_kind(device, "ec20")
        if not self.config.mock:
            await self._device_put(self.config.ec20_service_url, device, f"tracking/{action}", mode)
        else:
            self._mock_devices.setdefault(device, {})["tracking"] = action
        return {"device": device, "action": action, "mode": mode, "ok": True}

    async def ec20_preset_recall(self, device: str, preset_id: int) -> dict[str, Any]:
        if not 0 <= preset_id <= 255:
            raise ValueError("preset_id must be 0-255")
        self._require_kind(device, "ec20")
        if not self.config.mock:
            await self._device_put(self.config.ec20_service_url, device, f"preset/{preset_id}", "")
        return {"device": device, "preset_id": preset_id, "ok": True}

    async def ec20_status(self, device: str) -> dict[str, Any]:
        self._require_kind(device, "ec20")
        if self.config.mock:
            d = self._mock_devices.get(device, {})
            status = {"tracking": d.get("tracking", "disabled"), "state": "online"}
            return {"device": device, "ok": True, "status": status}
        addr = self._resolve_address(device)
        raw = await self._request("GET", f"{self.config.ec20_service_url}/{addr}/status")
        return {"device": device, "ok": True, "status": raw}
