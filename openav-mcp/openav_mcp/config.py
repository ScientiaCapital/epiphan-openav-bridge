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
        try:
            parsed = json.loads(raw)
        except json.JSONDecodeError as exc:
            raise ValueError(f"OPENAV_DEVICES is not valid JSON: {exc}") from exc
        if not isinstance(parsed, list):
            raise ValueError(
                f"OPENAV_DEVICES must be a JSON list, got {type(parsed).__name__}"
            )
        for i, d in enumerate(parsed):
            if not isinstance(d, dict):
                raise ValueError(f"OPENAV_DEVICES[{i}] must be an object, got {type(d).__name__}")
            for required in ("alias", "host"):
                if required not in d:
                    # Report the entry's keys only, never its values — one of them may be
                    # a password.
                    raise ValueError(
                        f"OPENAV_DEVICES[{i}] is missing required field '{required}' "
                        f"(present fields: {sorted(d.keys())})"
                    )
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
