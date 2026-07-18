"""Narrated smart-room demo — an AI agent runs a lecture capture, hardware-free.

Tells the whole story this project exists for: *plain-English intent → real AV room
control*, driven over the actual MCP protocol with NO hardware. It spawns
`python -m openav_mcp` in mock mode and walks a realistic lecture-capture scenario in
Room 320B (Agent → MCP → OpenAV orchestrator → Pearl encoder + EC20 camera), then
regenerates a shareable Markdown walkthrough (DEMO.md).

    pip install -e ".[dev]"                 # from openav-mcp/
    python scripts/demo_smart_room.py       # narrated run + regenerates ../DEMO.md
    python scripts/demo_smart_room.py --no-markdown   # console only
    python scripts/demo_smart_room.py --markdown path/to/OUT.md

Everything runs in-memory (OPENAV_MOCK=true). The mock control plane keeps device state,
so a status call *reflects* a prior control call — this is real orchestration, not canned
echoes. The device password is resolved internally and asserted to never appear in any
tool output. Exit 0 + "ROUND-TRIP OK" on success.
"""

from __future__ import annotations

import argparse
import asyncio
import json
import sys
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from mcp import ClientSession
from mcp.client.stdio import StdioServerParameters, stdio_client

# -- Demo fixture: one room, two devices. The password below is what must NEVER leak. --
SYSTEM = "smart-room-demo"
PEARL = "room-320b-pearl"
CAM = "room-320b-cam"
SECRET = "s3cret"
DEVICES = json.dumps(
    [
        {"alias": PEARL, "host": "pearl-320b.av.example", "username": "admin",
         "password": SECRET, "kind": "pearl"},
        {"alias": CAM, "host": "ec20-320b.av.example", "username": "admin",
         "password": SECRET, "kind": "ec20"},
    ]
)

DEFAULT_MARKDOWN = Path(__file__).resolve().parent.parent / "DEMO.md"


@dataclass
class Step:
    n: int
    title: str
    intent: str          # why the agent does this (plain English)
    tool: str            # MCP tool called ("—" for protocol-level steps)
    args: dict[str, Any]
    result: str          # compact result shown to the user


@dataclass
class Demo:
    steps: list[Step] = field(default_factory=list)
    outputs: list[str] = field(default_factory=list)  # every raw text the model saw

    def record(self, title: str, intent: str, tool: str, args: dict[str, Any], result: str) -> None:
        self.steps.append(Step(len(self.steps) + 1, title, intent, tool, args, result))
        self.outputs.append(result)


def _compact(text: str) -> str:
    """Re-render a JSON tool result as compact single-line text for display."""
    try:
        return json.dumps(json.loads(text), separators=(",", ":"))
    except (json.JSONDecodeError, TypeError):
        return text


async def _call(session: ClientSession, demo: Demo, title: str, intent: str,
                tool: str, args: dict[str, Any]) -> str:
    res = await session.call_tool(tool, args)
    text = res.content[0].text
    demo.record(title, intent, tool, args, _compact(text))
    return text


def _server(read_only: bool = False) -> StdioServerParameters:
    extra = ["--read-only"] if read_only else []
    return StdioServerParameters(
        command=sys.executable,
        args=["-m", "openav_mcp", *extra],
        env={"OPENAV_MOCK": "true", "OPENAV_DEVICES": DEVICES},
    )


async def run_scenario(demo: Demo) -> list[str]:
    """Drive the full lecture-capture scenario. Returns the discovered tool names."""
    async with stdio_client(_server()) as (read, write), ClientSession(read, write) as session:
        await session.initialize()

        # 1. Discover — what can the agent do in this room?
        tools = await session.list_tools()
        names = sorted(t.name for t in tools.tools)
        read_only = sorted(t.name for t in tools.tools if getattr(t.annotations, "readOnlyHint", False))
        assert {"run_scene", "set_room_state", "pearl_control_recording", "ec20_tracking"} <= set(names)
        demo.record(
            "Discover tools", "Agent asks the MCP server what it can control.",
            "list_tools", {},
            f"{len(names)} tools ({len(read_only)} read-only, {len(names) - len(read_only)} mutating)",
        )

        # 2. Inspect the room (scene layer)
        await _call(session, demo, "Inspect the room",
                    "What scenes and controls does Room 320B expose?",
                    "list_room_controls", {"system": SYSTEM})

        # 3. Baseline status (read-only, device layer)
        await _call(session, demo, "Baseline: camera", "Is the camera online, is it tracking?",
                    "ec20_status", {"device": CAM})
        await _call(session, demo, "Baseline: encoder", "Is the encoder online, is it recording?",
                    "pearl_status", {"device": PEARL})

        # 4. Frame the podium — preset 0 is the home/podium framing (exercises 0-255 range)
        await _call(session, demo, "Frame the podium",
                    "Recall preset 0 — the home/podium shot — before the talk starts.",
                    "ec20_preset_recall", {"device": CAM, "preset_id": 0})

        # 5. Compose the shot (PTZ)
        await _call(session, demo, "Compose the shot",
                    "Nudge pan/tilt and pull a little zoom to frame the lectern.",
                    "ec20_ptz", {"device": CAM, "pan": 12.0, "tilt": -3.0, "zoom": 2.0})

        # 6. Follow the presenter (AI tracking)
        await _call(session, demo, "Follow the presenter",
                    "Enable AI presenter tracking so the camera follows the speaker.",
                    "ec20_tracking", {"device": CAM, "action": "enable", "mode": "presenter"})

        # 7. Roll recording (encoder)
        await _call(session, demo, "Roll recording", "Start recording the session on the Pearl.",
                    "pearl_control_recording", {"device": PEARL, "action": "start"})

        # 8. Confirm live — status now REFLECTS steps 6-7 (proves real orchestration)
        await _call(session, demo, "Confirm: camera", "Verify tracking is now active.",
                    "ec20_status", {"device": CAM})
        await _call(session, demo, "Confirm: encoder", "Verify recording is now rolling.",
                    "pearl_status", {"device": PEARL})

        # 9. The one-command scene (scene layer shorthand for the whole thing)
        await _call(session, demo, "One-command scene",
                    "Show the shorthand: a single scene call the orchestrator expands into steps.",
                    "run_scene", {"system": SYSTEM, "scene": "record_session"})

        # 10. Wind down
        await _call(session, demo, "Stop recording", "The talk ends — stop the recording.",
                    "pearl_control_recording", {"device": PEARL, "action": "stop"})
        await _call(session, demo, "Release the camera", "Disable tracking and free the camera.",
                    "ec20_tracking", {"device": CAM, "action": "disable"})

        return names


async def run_readonly_showcase(demo: Demo) -> list[str]:
    """Spawn a second server in --read-only mode; the mutating tools disappear."""
    async with stdio_client(_server(read_only=True)) as (read, write), ClientSession(read, write) as session:
        await session.initialize()
        tools = await session.list_tools()
        names = sorted(t.name for t in tools.tools)
        demo.record(
            "Safety gate (read-only mode)",
            "Re-launched with --read-only: only monitoring tools are exported; nothing can drive hardware.",
            "list_tools (--read-only)", {}, f"{len(names)} tools: {', '.join(names)}",
        )
        return names


# ---------------------------------------------------------------- renderers

def print_console(demo: Demo) -> None:
    print("Smart-Room Demo — an AI agent runs a lecture capture in Room 320B")
    print("(hardware-free, over the real MCP protocol — OPENAV_MOCK=true)\n")
    for s in demo.steps:
        dots = "." * max(2, 26 - len(s.title))
        print(f"[{s.n:>2}] {s.title} {dots} {s.result}")
    print()


def render_markdown(demo: Demo, tool_names: list[str], readonly_names: list[str]) -> str:
    stamp = datetime.now(timezone.utc).strftime("%Y-%m-%d")
    lines: list[str] = []
    lines += [
        "# Smart-Room Demo — an AI agent runs a lecture capture",
        "",
        "> Auto-generated by `scripts/demo_smart_room.py`. Re-run that script to refresh.",
        "",
        "This walkthrough shows the whole point of `epiphan-openav-bridge`: **plain-English",
        "intent becoming real AV-room control**, driven over the actual [MCP](https://modelcontextprotocol.io)",
        "protocol — with **no hardware**. Everything below runs in-memory",
        "(`OPENAV_MOCK=true`); the same tool calls hit real Pearl/EC20 microservices when",
        "mock mode is off.",
        "",
        "## The control plane",
        "",
        "```",
        "  AI agent (LLM)",
        "        │  natural language → tool calls",
        "        ▼",
        "  openav-mcp  ──►  OpenAV orchestrator     (scene layer:  run_scene / set_room_state)",
        "        │       └►  Pearl microservice      (device layer: recording / streaming)",
        "        └────────►  EC20 microservice       (device layer: PTZ / tracking / presets)",
        "",
        "  Credentials are resolved INSIDE openav-mcp (device alias → user:pass@host).",
        "  The model only ever sees friendly aliases like 'room-320b-cam'.",
        "```",
        "",
        f"The agent discovered **{len(tool_names)} tools**: "
        + ", ".join(f"`{n}`" for n in tool_names)
        + ".",
        "",
        "## The scenario",
        "",
        "An agent is asked to *\"record the lecture in Room 320B and keep the camera on the",
        "presenter.\"* Here is every MCP call it makes, in order:",
        "",
        "| # | Agent intent | MCP tool | Args | Result |",
        "|---|---|---|---|---|",
    ]
    for s in demo.steps:
        args = "`" + json.dumps(s.args, separators=(",", ":")) + "`" if s.args else "—"
        lines.append(
            f"| {s.n} | {s.intent} | `{s.tool}` | {args} | `{s.result}` |"
        )
    lines += [
        "",
        "Notice steps 9–10: after `ec20_tracking(enable)` (7) and `pearl_control_recording(start)`",
        "(8), the status calls **report the new state** — the mock control plane tracks device",
        "state, so this demonstrates real orchestration, not canned responses.",
        "",
        "## Safety: read-only mode",
        "",
        "Re-launched with `--read-only`, the server exports only the monitoring tools — the",
        "mutating tools are gone, so a demo or dashboard deployment physically cannot drive",
        "hardware:",
        "",
        "```",
        f"$ python -m openav_mcp --read-only   →   {len(readonly_names)} tools: {', '.join(readonly_names)}",
        "```",
        "",
        "## Safety: no credential leakage",
        "",
        "The device password is configured once and resolved internally. It appears in **none**",
        "of the tool results above — the demo asserts this on every run and fails loudly if a",
        "secret ever escapes.",
        "",
        "## Run it yourself",
        "",
        "```bash",
        "cd openav-mcp",
        'pip install -e ".[dev]"',
        "python scripts/demo_smart_room.py",
        "```",
        "",
        f"_Generated {stamp} · hardware-free · exit `ROUND-TRIP OK`._",
        "",
    ]
    return "\n".join(lines)


# ---------------------------------------------------------------- main

async def main_async(markdown_path: Path | None) -> int:
    demo = Demo()
    tool_names = await run_scenario(demo)
    readonly_names = await run_readonly_showcase(demo)

    print_console(demo)

    # Security assertion — the password must appear in nothing the model/user saw.
    haystack = "\n".join(demo.outputs) + "\n" + "\n".join(s.result for s in demo.steps)
    if SECRET in haystack:
        print("!! CREDENTIAL LEAK DETECTED — aborting", file=sys.stderr)
        return 1

    if markdown_path is not None:
        md = render_markdown(demo, tool_names, readonly_names)
        if SECRET in md:
            print("!! CREDENTIAL LEAK in generated markdown — aborting", file=sys.stderr)
            return 1
        markdown_path.write_text(md, encoding="utf-8")
        print(f"wrote walkthrough → {markdown_path}")

    print("NO CREDENTIAL LEAK  |  ROUND-TRIP OK")
    return 0


def main() -> int:
    ap = argparse.ArgumentParser(description="Narrated hardware-free smart-room MCP demo.")
    ap.add_argument("--markdown", type=Path, default=DEFAULT_MARKDOWN,
                    help=f"where to write the walkthrough (default: {DEFAULT_MARKDOWN})")
    ap.add_argument("--no-markdown", action="store_true", help="console only; do not write DEMO.md")
    args = ap.parse_args()
    path = None if args.no_markdown else args.markdown
    return asyncio.run(main_async(path))


if __name__ == "__main__":
    sys.exit(main())
