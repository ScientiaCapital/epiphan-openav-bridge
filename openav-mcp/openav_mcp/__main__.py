"""Run the openav-mcp server over stdio.

    python -m openav_mcp                 # mutating tools enabled (default)
    python -m openav_mcp --read-only     # read-only tools only

Config comes from env (see config.load_config_from_env): OPENAV_ORCHESTRATOR_URL,
OPENAV_PEARL_URL, OPENAV_EC20_URL, OPENAV_DEVICES (JSON), OPENAV_MOCK.
"""

from __future__ import annotations

import asyncio
import contextlib
import sys

from openav_mcp.server import serve_stdio


def main() -> None:
    mutating = "--read-only" not in sys.argv
    with contextlib.suppress(KeyboardInterrupt):
        asyncio.run(serve_stdio(mutating_enabled=mutating))


if __name__ == "__main__":
    main()
