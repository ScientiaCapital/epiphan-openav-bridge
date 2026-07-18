# Contributing

Thanks for your interest in `epiphan-openav-bridge`. This project follows
[Dartmouth-OpenAV](https://github.com/Dartmouth-OpenAV)'s microservice conventions — see
[CLAUDE.md](CLAUDE.md) for the full set of coding rules before opening a PR.

## Building and testing

**First-time setup for each Go microservice** — the vendored `Dartmouth-OpenAV/microservice-framework`
submodule deliberately excludes `go.mod`/`go.sum` from its own git history (see its `.gitignore`), so a
fresh clone has neither. Run this once per service before `go test`/`go build` will work outside Docker:

```bash
cd openav-epiphan-pearl && bash ./init-framework-mod.sh
cd openav-epiphan-ec20 && bash ./init-framework-mod.sh
```

This regenerates the framework's `go.mod` and reconciles the top-level `go.mod`/`go.sum` against it —
which may show up as a local diff to `go.mod`/`go.sum` (the framework has no committed lockfile to pin
against, so this can pick up newer transitive dependency versions than what's currently committed). That's
expected; `git checkout -- go.mod go.sum` afterward if you don't want to keep the bump. CI runs this same
script on every fresh checkout — see `.github/workflows/ci.yml`.

```bash
# Go microservices (each is a standalone module)
cd openav-epiphan-pearl && go test ./source/ -v
cd openav-epiphan-ec20 && go test ./source/ -v

# Docker builds
docker build -t openav-epiphan-pearl ./openav-epiphan-pearl
docker build -t openav-epiphan-ec20 ./openav-epiphan-ec20

# openav-mcp (Python)
cd openav-mcp && pip install -e ".[dev]" && ruff check . && pytest -q
```

All four checks above also run in CI (`.github/workflows/ci.yml`) on every push and PR.

## Before opening a PR

- Both Go test suites and the `openav-mcp` test suite must pass.
- `ruff check .` must be clean for any `openav-mcp` changes.
- Follow the existing driver patterns in `openav-epiphan-pearl/source/driver.go` /
  `openav-epiphan-ec20/source/driver.go` — each microservice is a self-contained single binary
  per OpenAV convention (small helpers like `parseSocketKey` are intentionally duplicated rather
  than shared across services; see `.claude/observers/ARCH.md` for the reasoning).
- No hardcoded credentials, IPs, or device-specific config — the Go services take credentials
  per-request via the URL path; `openav-mcp` injects them from `OPENAV_DEVICES`.
- If you're changing EC20 PTZ/preset bounds or validation, update both the Go driver
  (`openav-epiphan-ec20/source/driver.go`) and the Python MCP client
  (`openav-mcp/openav_mcp/client.py`) together — these two layers must stay in sync (see the
  `DOC-CONFIRMED` comments in both files).

## Reporting bugs / requesting features

Open an issue. If you're running OpenAV at your institution and want to test with Epiphan
hardware, say so — that context is genuinely useful.

## Security issues

Please don't open a public issue for a security vulnerability — see [SECURITY.md](SECURITY.md).
