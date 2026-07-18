#!/usr/bin/env bash
# One-time (post-clone) setup: the Dartmouth-OpenAV/microservice-framework submodule's own
# .gitignore excludes go.mod/go.sum, so a fresh clone/checkout has neither. Without this,
# `go test`/`go build` outside Docker fail with:
#   "reading source/microservice-framework/go.mod: no such file or directory"
# (The Dockerfile does the equivalent inline for container builds — this script is the same
# step for local dev and CI.) Safe to re-run any time.
set -euo pipefail
service_dir="$(cd "$(dirname "$0")" && pwd)"

cd "$service_dir/source/microservice-framework"
rm -f go.mod go.sum
go mod init github.com/Dartmouth-OpenAV/microservice-framework
go mod tidy

# The framework module just re-resolved its own deps from scratch (no committed go.sum to pin
# against, per its own upstream .gitignore) — reconcile the top-level go.mod/go.sum against
# whatever versions it picked, same as the Dockerfile's post-replace `go mod tidy` step. This
# will likely modify go.mod/go.sum locally; that's expected, not a bug — commit the bump only
# if you intend to, otherwise `git checkout -- go.mod go.sum` after testing.
cd "$service_dir"
go mod tidy
