"""Tests for openav-mcp — the MCP face over the OpenAV↔Epiphan REST bridges.

Runs entirely in mock mode: no hardware, no live OpenAV orchestrator. Verifies
the two tool layers (scene + device), credential injection (LLM never sees
passwords), and the MCP server tool catalog + annotations.
"""

from __future__ import annotations

import json

import httpx
import pytest
from mcp import types

from openav_mcp.client import DeviceRequestError, OpenAVClient
from openav_mcp.config import DeviceConfig, OpenAVConfig, load_config_from_env
from openav_mcp.server import build_server, list_tool_specs


def _config(*, mock: bool = True) -> OpenAVConfig:
    return OpenAVConfig(
        orchestrator_url="http://orchestrator:8080",
        pearl_service_url="http://pearl-svc:80",
        ec20_service_url="http://ec20-svc:80",
        devices={
            "room-320b-pearl": DeviceConfig(
                alias="room-320b-pearl",
                host="pearl-host",
                username="admin",
                password="s3cret",
                kind="pearl",
            ),
            "room-320b-cam": DeviceConfig(
                alias="room-320b-cam",
                host="ec20-host",
                username="admin",
                password="s3cret",
                kind="ec20",
            ),
        },
        mock=mock,
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
    async def test_ec20_jog(self) -> None:
        c = _client()
        out = await c.ec20_jog("room-320b-cam", "up", speed=8)
        assert out["direction"] == "up"
        assert out["speed"] == 8
        assert out["ok"] is True

    @pytest.mark.asyncio
    async def test_ec20_jog_invalid_direction_errors(self) -> None:
        c = _client()
        with pytest.raises(ValueError):
            await c.ec20_jog("room-320b-cam", "sideways")

    @pytest.mark.asyncio
    async def test_ec20_jog_speed_out_of_range_errors(self) -> None:
        c = _client()
        with pytest.raises(ValueError):
            await c.ec20_jog("room-320b-cam", "left", speed=99)

    @pytest.mark.asyncio
    async def test_ec20_preset_save(self) -> None:
        c = _client()
        out = await c.ec20_preset_save("room-320b-cam", 3, name="Podium")
        assert out["preset_id"] == 3
        assert out["name"] == "Podium"
        assert out["ok"] is True

    @pytest.mark.asyncio
    async def test_ec20_preset_save_out_of_range_errors(self) -> None:
        c = _client()
        with pytest.raises(ValueError):
            await c.ec20_preset_save("room-320b-cam", 300)

    @pytest.mark.asyncio
    async def test_ec20_preset_save_name_too_long_errors(self) -> None:
        c = _client()
        with pytest.raises(ValueError):
            await c.ec20_preset_save("room-320b-cam", 1, name="x" * 65)

    @pytest.mark.asyncio
    async def test_unknown_device_errors(self) -> None:
        c = _client()
        with pytest.raises(KeyError):
            await c.pearl_control_recording("does-not-exist", "start")

    @pytest.mark.asyncio
    async def test_ec20_ptz_pan_out_of_range_errors(self) -> None:
        c = _client()
        with pytest.raises(ValueError):
            await c.ec20_ptz("room-320b-cam", pan=999, tilt=0, zoom=0)

    @pytest.mark.asyncio
    async def test_ec20_ptz_tilt_out_of_range_errors(self) -> None:
        c = _client()
        with pytest.raises(ValueError):
            await c.ec20_ptz("room-320b-cam", pan=0, tilt=999, zoom=0)

    @pytest.mark.asyncio
    async def test_ec20_ptz_default_speed(self) -> None:
        c = _client()
        out = await c.ec20_ptz("room-320b-cam", pan=10, tilt=5, zoom=1.0)
        assert out["speed"] == 50

    @pytest.mark.asyncio
    async def test_ec20_ptz_custom_speed(self) -> None:
        c = _client()
        out = await c.ec20_ptz("room-320b-cam", pan=10, tilt=5, zoom=1.0, speed=90)
        assert out["speed"] == 90

    @pytest.mark.asyncio
    async def test_ec20_ptz_non_positive_speed_errors(self) -> None:
        c = _client()
        with pytest.raises(ValueError):
            await c.ec20_ptz("room-320b-cam", pan=0, tilt=0, zoom=0, speed=0)

    @pytest.mark.asyncio
    async def test_ec20_tool_on_pearl_device_errors(self) -> None:
        # room-320b-pearl is configured kind="pearl" — nothing should let an ec20 tool
        # target it silently.
        c = _client()
        with pytest.raises(ValueError):
            await c.ec20_tracking("room-320b-pearl", "enable")

    @pytest.mark.asyncio
    async def test_pearl_tool_on_ec20_device_errors(self) -> None:
        c = _client()
        with pytest.raises(ValueError):
            await c.pearl_control_recording("room-320b-cam", "start")

    @pytest.mark.asyncio
    async def test_pearl_singletouch_updates_mock_recording_state(self) -> None:
        c = _client()
        await c.pearl_singletouch("room-320b-pearl", "start")
        status = await c.pearl_status("room-320b-pearl")
        assert status["status"]["recording"] == "start"

    @pytest.mark.asyncio
    async def test_pearl_status_envelope_shape(self) -> None:
        c = _client()
        out = await c.pearl_status("room-320b-pearl")
        assert set(out) == {"device", "ok", "status"}

    @pytest.mark.asyncio
    async def test_ec20_status_envelope_shape(self) -> None:
        c = _client()
        out = await c.ec20_status("room-320b-cam")
        assert set(out) == {"device", "ok", "status"}


class TestConfigLoading:
    def test_malformed_json_raises_clean_error(self, monkeypatch) -> None:
        monkeypatch.setenv("OPENAV_DEVICES", "not json")
        with pytest.raises(ValueError, match="not valid JSON"):
            load_config_from_env()

    def test_missing_required_field_raises_clean_error(self, monkeypatch) -> None:
        monkeypatch.setenv("OPENAV_DEVICES", '[{"host": "1.2.3.4", "password": "s3cret"}]')
        with pytest.raises(ValueError) as exc_info:
            load_config_from_env()
        message = str(exc_info.value)
        assert "missing required field 'alias'" in message
        assert "s3cret" not in message  # never echo values, only field names

    @pytest.mark.parametrize("raw", ['{"alias": "x"}', "5", '"just a string"', "null"])
    def test_non_list_top_level_raises_clean_error(self, monkeypatch, raw) -> None:
        # Valid JSON, but not a list — used to crash with an uncaught
        # AttributeError/TypeError instead of a clean config error.
        monkeypatch.setenv("OPENAV_DEVICES", raw)
        with pytest.raises(ValueError, match="must be a JSON list"):
            load_config_from_env()

    def test_non_object_list_entry_raises_clean_error(self, monkeypatch) -> None:
        monkeypatch.setenv("OPENAV_DEVICES", '["not-an-object"]')
        with pytest.raises(ValueError, match=r"OPENAV_DEVICES\[0\] must be an object"):
            load_config_from_env()


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
        assert addr == "admin:s3cret@pearl-host"  # no special chars -> unchanged

    @pytest.mark.asyncio
    async def test_device_http_error_never_leaks_credentials(self) -> None:
        # Live mode (mock=False) so the request actually builds a credential-bearing
        # URL; a MockTransport stands in for the real device and returns a plain 500,
        # simulating e.g. a wrong password on real hardware.
        c = OpenAVClient(_config(mock=False))
        c._transport = httpx.MockTransport(lambda request: httpx.Response(500))
        with pytest.raises(DeviceRequestError) as exc_info:
            await c.pearl_status("room-320b-pearl")
        message = str(exc_info.value)
        assert "s3cret" not in message
        assert "pearl-host" not in message

    @pytest.mark.asyncio
    async def test_special_char_credentials_are_percent_encoded(self) -> None:
        # A password with '/', '#', ':' and '@' must stay inside the single :address
        # path segment: unencoded, '/' adds path segments, '#' truncates the URL into a
        # fragment, and an extra ':'/'@' makes the framework drop the credentials.
        password = "p/s#s:w@rd"
        cfg = _config(mock=False)
        cfg.devices["room-320b-cam"].username = "ad/m:n"
        cfg.devices["room-320b-cam"].password = password
        seen: list[httpx.Request] = []

        def handler(request: httpx.Request) -> httpx.Response:
            seen.append(request)
            return httpx.Response(200, text='"ok"')

        c = OpenAVClient(cfg)
        c._transport = httpx.MockTransport(handler)
        await c.ec20_tracking("room-320b-cam", "enable")
        await c.ec20_status("room-320b-cam")

        assert len(seen) == 2
        for req in seen:
            raw_path = req.url.raw_path.decode()
            assert raw_path.startswith("/ad%2Fm%3An:p%2Fs%23s%3Aw%40rd@ec20-host/")
            assert req.url.fragment == ""
            assert password not in str(req.url)
        assert seen[0].url.raw_path.decode().count("/") == 3  # /<address>/tracking/enable

    def test_address_percent_encodes_credentials_only(self) -> None:
        d = DeviceConfig(
            alias="a", host="192.0.2.10:80", username="u@x", password="a/b#c:d@e", kind="pearl"
        )
        assert d.address == "u%40x:a%2Fb%23c%3Ad%40e@192.0.2.10:80"

    @pytest.mark.asyncio
    async def test_invalid_url_maps_to_credential_safe_error(self) -> None:
        # httpx.InvalidURL is NOT an httpx.HTTPError subclass, and its message embeds
        # the URL (and so user:pass). It must surface as DeviceRequestError instead.
        cfg = _config(mock=False)
        cfg.devices["room-320b-pearl"].host = "pearl-host\x7f"  # non-printable -> InvalidURL
        cfg.devices["room-320b-pearl"].password = "p/s#s:w@rd"
        c = OpenAVClient(cfg)
        c._transport = httpx.MockTransport(lambda request: httpx.Response(200))
        with pytest.raises(DeviceRequestError) as exc_info:
            await c.pearl_status("room-320b-pearl")
        message = str(exc_info.value)
        assert "InvalidURL" in message
        for secret in ("p/s#s:w@rd", "p%2Fs%23s%3Aw%40rd", "s3cret", "pearl-host"):
            assert secret not in message
        assert exc_info.value.__cause__ is None and exc_info.value.__suppress_context__


class TestMCPServer:
    def test_tool_catalog_has_both_layers(self) -> None:
        names = {t.name for t in list_tool_specs(mutating_enabled=True)}
        # scene layer
        assert {"set_room_state", "run_scene", "list_room_controls"} <= names
        # device layer
        assert {
            "pearl_control_recording",
            "pearl_status",
            "ec20_ptz",
            "ec20_tracking",
            "ec20_jog",
            "ec20_preset_recall",
            "ec20_preset_save",
        } <= names

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

    @pytest.mark.asyncio
    async def test_mutating_tool_call_actually_blocked_when_disabled(self) -> None:
        # test_mutating_tools_hidden_when_disabled only checks the catalog listing —
        # this drives an actual call through the registered handler to confirm the
        # runtime guard in _call fires too, not just the catalog omission.
        server = build_server(_client(), mutating_enabled=False)
        handler = server.request_handlers[types.CallToolRequest]
        request = types.CallToolRequest(
            method="tools/call",
            params=types.CallToolRequestParams(
                name="pearl_control_recording",
                arguments={"device": "room-320b-pearl", "action": "start"},
            ),
        )
        result = await handler(request)
        payload = json.loads(result.root.content[0].text)
        assert payload == {"error": "tool 'pearl_control_recording' disabled (mutating tools off)"}

    @pytest.mark.asyncio
    async def test_readonly_tool_call_still_works_when_mutating_disabled(self) -> None:
        server = build_server(_client(), mutating_enabled=False)
        handler = server.request_handlers[types.CallToolRequest]
        request = types.CallToolRequest(
            method="tools/call",
            params=types.CallToolRequestParams(
                name="pearl_status", arguments={"device": "room-320b-pearl"}
            ),
        )
        result = await handler(request)
        payload = json.loads(result.root.content[0].text)
        assert payload["ok"] is True
