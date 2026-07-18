# Security Policy

## Reporting a vulnerability

Please **do not** open a public GitHub issue for a security vulnerability. Instead, email the
maintainer directly at **tkipper@epiphan.com** with:

- A description of the issue and its potential impact
- Steps to reproduce (a minimal example is ideal)
- Any relevant logs or configuration (redact real device credentials/IPs first)

We'll acknowledge your report and follow up with next steps. There is no bug bounty program.

## Scope notes specific to this project

- The Go microservices (`openav-epiphan-pearl`, `openav-epiphan-ec20`) are stateless and take
  device credentials per-request via the URL path — they should only ever be run on a trusted
  local network, never exposed directly to the internet.
- `openav-mcp` resolves device credentials from the `OPENAV_DEVICES` env var and is designed so
  the calling LLM agent never sees a raw credential — if you find a path where a credential
  leaks into a tool response or log line, that's a high-priority report.
- The EC20 REST endpoint paths in `openav-epiphan-ec20/source/driver.go` are still placeholders
  pending hardware verification (see `.claude/programs/ec20-api-discovery.md`) — that's a known,
  tracked gap, not something you need to report separately.
