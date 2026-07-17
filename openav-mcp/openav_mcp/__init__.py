"""openav-mcp — an MCP server face over the OpenAV↔Epiphan REST bridges.

Exposes two tool layers to an LLM agent:
  * scene-level — the OpenAV orchestrator state API (run whole rooms/scenes)
  * device-level — the Epiphan Pearl + EC20 OpenAV microservices (precise control)

Device credentials are injected from config so the model never handles passwords.
Part of the "Epiphan hardware running OpenAV" agentic control plane — SilkRoute
orchestrates these tools; this server never replaces OpenAV, it fronts it.
"""

__version__ = "0.1.0"
