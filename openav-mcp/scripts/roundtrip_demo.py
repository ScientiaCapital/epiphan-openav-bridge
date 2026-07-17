"""Self-contained proof that openav-mcp works over real MCP — no hardware, no SilkRoute.

Spawns `python -m openav_mcp` in mock mode, lists its tools, and drives a scene +
a device tool through the stdio MCP protocol. This is the plug-and-play smoke test
an engineer runs first.

    pip install -e ".[dev]"        # from openav-mcp/
    python scripts/roundtrip_demo.py

Expected: "ROUND-TRIP OK" and a JSON result for each call.
"""

from __future__ import annotations

import asyncio
import json
import sys

from mcp import ClientSession
from mcp.client.stdio import StdioServerParameters, stdio_client

DEVICES = json.dumps(
    [
        {"alias": "room-320b-pearl", "host": "pearl-host", "username": "admin",
         "password": "s3cret", "kind": "pearl"},
        {"alias": "room-320b-cam", "host": "ec20-host", "username": "admin",
         "password": "s3cret", "kind": "ec20"},
    ]
)


async def main() -> int:
    params = StdioServerParameters(
        command=sys.executable,
        args=["-m", "openav_mcp"],
        env={"OPENAV_MOCK": "true", "OPENAV_DEVICES": DEVICES},
    )
    async with stdio_client(params) as (read, write), ClientSession(read, write) as session:
        await session.initialize()

        tools = await session.list_tools()
        names = sorted(t.name for t in tools.tools)
        print(f"discovered {len(names)} tools:", names)
        assert {"run_scene", "set_room_state", "pearl_control_recording", "ec20_tracking"} <= set(names)

        scene = await session.call_tool(
            "run_scene", {"system": "smart-room-demo", "scene": "record_session"}
        )
        scene_text = scene.content[0].text
        print("run_scene ->", scene_text)

        cam = await session.call_tool(
            "ec20_tracking", {"device": "room-320b-cam", "action": "enable", "mode": "presenter"}
        )
        cam_text = cam.content[0].text
        print("ec20_tracking ->", cam_text)

        assert "s3cret" not in scene_text and "s3cret" not in cam_text, "credential leak!"
        print("ROUND-TRIP OK")
        return 0


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))
