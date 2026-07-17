"""Configuration for openav-mcp — device aliases + service URLs, from env.

The LLM references devices by friendly ALIAS (e.g. "room-320b-pearl"); this
module resolves aliases to host + credentials so the model never sees passwords.
"""

from __future__ import annotations

import json
import os
from dataclasses import dataclass, field
from typing import Literal

DeviceKind = Literal["pearl", "ec20"]


@dataclass
class DeviceConfig:
    """One Epiphan device reachable through an OpenAV microservice."""

    alias: str
    host: str
    username: str
    password: str
    kind: DeviceKind

    @property
    def address(self) -> str:
        """The OpenAV ``:address`` segment: ``user:pass@host`` (INTERNAL — never
        return this to a tool caller; it embeds the password)."""
        return f"{self.username}:{self.password}@{self.host}"


@dataclass
class OpenAVConfig:
    """Where the OpenAV orchestrator and per-device microservices live."""

    orchestrator_url: str = "http://localhost:8080"
    pearl_service_url: str = "http://localhost:8081"
    ec20_service_url: str = "http://localhost:8082"
    devices: dict[str, DeviceConfig] = field(default_factory=dict)
    # Mock mode: no HTTP — record/echo calls in memory. For hardware-free demos+tests.
    mock: bool = False

    def device(self, alias: str) -> DeviceConfig:
        try:
            return self.devices[alias]
        except KeyError as exc:
            known = ", ".join(sorted(self.devices)) or "(none configured)"
            raise KeyError(f"Unknown device '{alias}'. Known devices: {known}") from exc


def load_config_from_env() -> OpenAVConfig:
    """Build config from env vars.

    OPENAV_ORCHESTRATOR_URL, OPENAV_PEARL_URL, OPENAV_EC20_URL, OPENAV_MOCK,
    and OPENAV_DEVICES (JSON list of {alias, host, username, password, kind}).
    """
    devices: dict[str, DeviceConfig] = {}
    raw = os.environ.get("OPENAV_DEVICES", "").strip()
    if raw:
        for d in json.loads(raw):
            devices[d["alias"]] = DeviceConfig(
                alias=d["alias"],
                host=d["host"],
                username=d.get("username", "admin"),
                password=d.get("password", ""),
                kind=d.get("kind", "pearl"),
            )
    return OpenAVConfig(
        orchestrator_url=os.environ.get("OPENAV_ORCHESTRATOR_URL", "http://localhost:8080"),
        pearl_service_url=os.environ.get("OPENAV_PEARL_URL", "http://localhost:8081"),
        ec20_service_url=os.environ.get("OPENAV_EC20_URL", "http://localhost:8082"),
        devices=devices,
        mock=os.environ.get("OPENAV_MOCK", "").lower() in {"1", "true", "yes"},
    )
