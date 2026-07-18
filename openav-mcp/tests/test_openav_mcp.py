"""Tests for openav-mcp — the MCP face over the OpenAV↔Epiphan REST bridges.

Runs entirely in mock mode: no hardware, no live OpenAV orchestrator. Verifies
the two tool layers (scene + device), credential injection (LLM never sees
passwords), and the MCP server tool catalog + annotations.
"""

from __future__ import annotations

import pytest

from openav_mcp.client import OpenAVClient
from openav_mcp.config import DeviceConfig, OpenAVConfig
from openav_mcp.server import build_server, list_tool_specs


def _config() -> OpenAVConfig:
    return OpenAVConfig(
        orchestrator_url="http://orchestrator:8080",
        pearl_service_url="http://pearl-svc:80",
        ec20_service_url="http://ec20-svc:80",
        devices={
            "room-320b-pearl": DeviceConfig(
                alias="room-320b-pearl", host="pearl-host", username="admin",
                password="s3cret", kind="pearl",
            ),
            "room-320b-cam": DeviceConfig(
                alias="room-320b-cam", host="ec20-host", username="admin",
                password="s3cret", kind="ec20",
            ),
        },
        mock=True,
    )


def _client() -> OpenAVClient:
    return OpenAVClient(_config())


class TestSceneLayer:
    @pytest.mark.asyncio
    async def test_set_room_state_records_partial_tree(self) -> None:
        c = _client()
        out = await c.set_room_state("smart-room-demo", "recording", "record", True)
        # Mock echoes the exact partial-state tree it would PUT to the orchestrator.
        assert out["control_sets"]["recording"]["controls"]["record"]["value"] is True

    @pytest.mark.asyncio
    async def test_run_scene_executes_sequence(self) -> None:
        c = _client()
        out = await c.run_scene("smart-room-demo", "record_session")
        # record_session = enable tracking + start recording (>=2 steps)
        assert out["ok"] is True
        assert len(out["steps"]) >= 2


class TestDeviceLayer:
    @pytest.mark.asyncio
    async def test_pearl_recording_start(self) -> None:
        c = _client()
        out = await c.pearl_control_recording("room-320b-pearl", "start")
        assert out["action"] == "start"
        assert out["device"] == "room-320b-pearl"

    @pytest.mark.asyncio
    async def test_ec20_tracking_enable(self) -> None:
        c = _client()
        out = await c.ec20_tracking("room-320b-cam", "enable", mode="presenter")
        assert out["action"] == "enable"
        assert out["mode"] == "presenter"

    @pytest.mark.asyncio
    async def test_ec20_preset_zero_is_valid(self) -> None:
        # Preset 0 is valid per EC20 docs (home/podium framing). Mirrors the Go
        # driver's validatePresetID 0-255 fix — keeps the two layers in sync.
        c = _client()
        out = await c.ec20_preset_recall("room-320b-cam", 0)
        assert out["preset_id"] == 0
        assert out["ok"] is True

    @pytest.mark.asyncio
    async def test_ec20_preset_out_of_range_errors(self) -> None:
        c = _client()
        with pytest.raises(ValueError):
            await c.ec20_preset_recall("room-320b-cam", 256)

    @pytest.mark.asyncio
    async def test_unknown_device_errors(self) -> None:
        c = _client()
        with pytest.raises(KeyError):
            await c.pearl_control_recording("does-not-exist", "start")


class TestCredentialSafety:
    @pytest.mark.asyncio
    async def test_password_never_in_output(self) -> None:
        c = _client()
        results = [
            await c.pearl_control_recording("room-320b-pearl", "start"),
            await c.pearl_status("room-320b-pearl"),
            await c.set_room_state("smart-room-demo", "recording", "record", True),
        ]
        assert all("s3cret" not in str(r) for r in results)

    def test_resolved_address_has_creds_but_is_internal(self) -> None:
        c = _client()
        addr = c._resolve_address("room-320b-pearl")  # internal only
        assert addr == "admin:s3cret@pearl-host"


class TestMCPServer:
    def test_tool_catalog_has_both_layers(self) -> None:
        names = {t.name for t in list_tool_specs(mutating_enabled=True)}
        # scene layer
        assert {"set_room_state", "run_scene", "list_room_controls"} <= names
        # device layer
        assert {"pearl_control_recording", "pearl_status", "ec20_ptz", "ec20_tracking"} <= names

    def test_readonly_tools_annotated(self) -> None:
        specs = {t.name: t for t in list_tool_specs(mutating_enabled=True)}
        assert specs["pearl_status"].annotations.readOnlyHint is True
        assert specs["pearl_control_recording"].annotations.readOnlyHint is False

    def test_mutating_tools_hidden_when_disabled(self) -> None:
        names = {t.name for t in list_tool_specs(mutating_enabled=False)}
        assert "pearl_status" in names  # read-only stays
        assert "pearl_control_recording" not in names  # mutating hidden

    def test_build_server_named(self) -> None:
        server = build_server(_client(), mutating_enabled=True)
        assert server.name == "openav"
