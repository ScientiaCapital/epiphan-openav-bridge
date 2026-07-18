"""MCP server exposing the OpenAVClient's two tool layers to an LLM agent.

Read-only tools are always exported; mutating (device/scene control) tools are
gated by ``mutating_enabled`` — mirroring SilkRoute's read-only-default export
policy so a demo/monitoring deployment can't accidentally drive hardware.
"""

from __future__ import annotations

import json
import sys
from dataclasses import dataclass
from typing import Any

import structlog
from mcp import types
from mcp.server import Server
from mcp.server.stdio import stdio_server

from openav_mcp.client import DeviceRequestError, OpenAVClient
from openav_mcp.config import load_config_from_env

_DEVICE = {"device": {"type": "string", "description": "Device alias from config"}}


def _schema(props: dict[str, Any], required: list[str]) -> dict[str, Any]:
    return {"type": "object", "properties": props, "required": required}


@dataclass
class _Spec:
    name: str
    description: str
    schema: dict[str, Any]
    read_only: bool
    method: str  # OpenAVClient attribute name


_SPECS: list[_Spec] = [
    # -- scene layer --
    _Spec("set_room_state", "Set one control on a room via the OpenAV orchestrator.",
          _schema({"system": {"type": "string"}, "control_set": {"type": "string"},
                   "control": {"type": "string"}, "value": {"type": "boolean"}},
                  ["system", "control_set", "control", "value"]), False, "set_room_state"),
    _Spec("run_scene", "Run a named multi-step room scene (e.g. record_session).",
          _schema({"system": {"type": "string"}, "scene": {"type": "string"}},
                  ["system", "scene"]), False, "run_scene"),
    _Spec("list_room_controls", "List available scenes and control sets for a room.",
          _schema({"system": {"type": "string"}}, ["system"]), True, "list_room_controls"),
    # -- device layer: Pearl --
    _Spec("pearl_control_recording", "Start or stop recording on a Pearl encoder.",
          _schema({**_DEVICE, "action": {"type": "string", "enum": ["start", "stop"]}},
                  ["device", "action"]), False, "pearl_control_recording"),
    _Spec("pearl_singletouch", "Start/stop Pearl recording + streaming together.",
          _schema({**_DEVICE, "action": {"type": "string", "enum": ["start", "stop"]}},
                  ["device", "action"]), False, "pearl_singletouch"),
    _Spec("pearl_status", "Get a Pearl encoder's recording status.",
          _schema({**_DEVICE}, ["device"]), True, "pearl_status"),
    # -- device layer: EC20 --
    _Spec("ec20_ptz", "Move an EC20 PTZ camera (pan, tilt, zoom, optional move speed).",
          _schema({**_DEVICE, "pan": {"type": "number"}, "tilt": {"type": "number"},
                   "zoom": {"type": "number"},
                   "speed": {"type": "integer", "default": 50,
                              "description": "PTZ move speed, no documented range — must be positive"}},
                  ["device", "pan", "tilt", "zoom"]), False, "ec20_ptz"),
    _Spec("ec20_tracking", "Enable/disable EC20 AI presenter tracking.",
          _schema({**_DEVICE, "action": {"type": "string", "enum": ["enable", "disable"]},
                   "mode": {"type": "string", "default": "presenter"}},
                  ["device", "action"]), False, "ec20_tracking"),
    _Spec("ec20_preset_recall", "Recall an EC20 PTZ preset (0-255).",
          _schema({**_DEVICE, "preset_id": {"type": "integer"}}, ["device", "preset_id"]),
          False, "ec20_preset_recall"),
    _Spec("ec20_status", "Get an EC20 camera's status (incl. tracking).",
          _schema({**_DEVICE}, ["device"]), True, "ec20_status"),
]


def list_tool_specs(*, mutating_enabled: bool) -> list[types.Tool]:
    """The MCP Tool catalog. Mutating tools are omitted when disabled."""
    tools: list[types.Tool] = []
    for s in _SPECS:
        if not s.read_only and not mutating_enabled:
            continue
        tools.append(
            types.Tool(
                name=s.name,
                description=s.description,
                inputSchema=s.schema,
                annotations=types.ToolAnnotations(
                    readOnlyHint=s.read_only,
                    destructiveHint=not s.read_only,
                ),
            )
        )
    return tools


async def _dispatch(client: OpenAVClient, name: str, args: dict[str, Any]) -> Any:
    spec = next((s for s in _SPECS if s.name == name), None)
    if spec is None:
        return {"error": f"unknown tool '{name}'"}
    method = getattr(client, spec.method)
    result = method(**args)
    if hasattr(result, "__await__"):  # async client methods
        result = await result
    return result


def build_server(
    client: OpenAVClient | None = None, *, mutating_enabled: bool = True, name: str = "openav"
) -> Server:
    """Build the MCP server. Defaults to env-configured client."""
    active = client or OpenAVClient(load_config_from_env())
    server: Server = Server(name)

    @server.list_tools()
    async def _list() -> list[types.Tool]:
        return list_tool_specs(mutating_enabled=mutating_enabled)

    @server.call_tool()
    async def _call(name: str, arguments: dict[str, Any]) -> list[types.TextContent]:
        if not mutating_enabled and not any(
            s.name == name and s.read_only for s in _SPECS
        ):
            text = json.dumps({"error": f"tool '{name}' disabled (mutating tools off)"})
            return [types.TextContent(type="text", text=text)]
        try:
            result = await _dispatch(active, name, arguments or {})
            return [types.TextContent(type="text", text=json.dumps(result))]
        except (KeyError, ValueError, DeviceRequestError) as exc:
            return [types.TextContent(type="text", text=json.dumps({"error": str(exc)}))]

    return server


async def serve_stdio(*, mutating_enabled: bool = True) -> None:
    server = build_server(mutating_enabled=mutating_enabled)
    stderr_log = structlog.wrap_logger(structlog.PrintLogger(file=sys.stderr))
    stderr_log.info("openav_mcp_starting", mutating_enabled=mutating_enabled)
    async with stdio_server() as (read_stream, write_stream):
        await server.run(read_stream, write_stream, server.create_initialization_options())
