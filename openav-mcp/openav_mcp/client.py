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

from openav_mcp.config import OpenAVConfig

# Named multi-step scenes for run_scene(). Each step = (control_set, control, value).
SCENE_RECIPES: dict[str, list[tuple[str, str, Any]]] = {
    "record_session": [("camera", "tracking", True), ("recording", "record", True)],
    "stop_session": [("recording", "record", False), ("camera", "tracking", False)],
}


class OpenAVClient:
    def __init__(self, config: OpenAVConfig) -> None:
        self.config = config
        # Mock state: last state tree PUT, and per-device action log.
        self._mock_state: dict[str, Any] = {}
        self._mock_devices: dict[str, dict[str, Any]] = {}

    # -- internal ---------------------------------------------------------
    def _resolve_address(self, alias: str) -> str:
        """alias → ``user:pass@host`` (INTERNAL; embeds password, never returned)."""
        return self.config.device(alias).address

    @staticmethod
    def _state_tree(control_set: str, control: str, value: Any) -> dict[str, Any]:
        return {"control_sets": {control_set: {"controls": {control: {"value": value}}}}}

    async def _put(self, url: str, json_body: Any) -> Any:
        import httpx

        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.put(url, json=json_body)
            resp.raise_for_status()
            return resp.text

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
        if self.config.mock:
            self.config.device(device)  # validate alias
            self._mock_devices.setdefault(device, {})["recording"] = action
        else:
            await self._device_put(self.config.pearl_service_url, device, "recording", action)
        return {"device": device, "action": action, "ok": True}

    async def pearl_singletouch(self, device: str, action: str) -> dict[str, Any]:
        if action not in {"start", "stop"}:
            raise ValueError("action must be 'start' or 'stop'")
        if not self.config.mock:
            await self._device_put(self.config.pearl_service_url, device, "singletouch", action)
        else:
            self.config.device(device)
        return {"device": device, "action": action, "ok": True}

    async def pearl_status(self, device: str) -> dict[str, Any]:
        self.config.device(device)  # validate alias
        if self.config.mock:
            rec = self._mock_devices.get(device, {}).get("recording", "stopped")
            return {"device": device, "recording": rec, "state": "online"}
        import httpx

        addr = self._resolve_address(device)
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.get(f"{self.config.pearl_service_url}/{addr}/status")
            resp.raise_for_status()
            return {"device": device, "raw": resp.text}

    # -- device layer: EC20 ----------------------------------------------
    async def ec20_ptz(self, device: str, pan: float, tilt: float, zoom: float) -> dict[str, Any]:
        if not self.config.mock:
            await self._device_put(
                self.config.ec20_service_url, device, f"ptz/{pan}/{tilt}", zoom
            )
        else:
            self.config.device(device)
            self._mock_devices.setdefault(device, {})["ptz"] = [pan, tilt, zoom]
        return {"device": device, "pan": pan, "tilt": tilt, "zoom": zoom, "ok": True}

    async def ec20_tracking(self, device: str, action: str, mode: str = "presenter") -> dict[str, Any]:
        if action not in {"enable", "disable"}:
            raise ValueError("action must be 'enable' or 'disable'")
        if not self.config.mock:
            await self._device_put(self.config.ec20_service_url, device, f"tracking/{action}", mode)
        else:
            self.config.device(device)
            self._mock_devices.setdefault(device, {})["tracking"] = action
        return {"device": device, "action": action, "mode": mode, "ok": True}

    async def ec20_preset_recall(self, device: str, preset_id: int) -> dict[str, Any]:
        if not 1 <= preset_id <= 255:
            raise ValueError("preset_id must be 1-255")
        if not self.config.mock:
            await self._device_put(self.config.ec20_service_url, device, f"preset/{preset_id}", "")
        else:
            self.config.device(device)
        return {"device": device, "preset_id": preset_id, "ok": True}

    async def ec20_status(self, device: str) -> dict[str, Any]:
        self.config.device(device)
        if self.config.mock:
            d = self._mock_devices.get(device, {})
            return {"device": device, "tracking": d.get("tracking", "disabled"), "state": "online"}
        import httpx

        addr = self._resolve_address(device)
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.get(f"{self.config.ec20_service_url}/{addr}/status")
            resp.raise_for_status()
            return {"device": device, "raw": resp.text}
