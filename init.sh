#!/usr/bin/env bash
set -euo pipefail

echo "Initializing epiphan-openav-bridge (Go microservices)..."

git submodule update --init --recursive

for svc in openav-epiphan-pearl openav-epiphan-ec20; do
  if [ -d "$svc" ]; then
    echo "Building $svc..."
    pushd "$svc" >/dev/null
    go mod tidy
    go build -o /dev/null . 2>/dev/null || true
    popd >/dev/null
  fi
done

if [ -d "proof" ] && [ -f "proof/requirements.txt" ]; then
  if [ ! -d ".venv" ]; then
    python3 -m venv .venv
  fi
  source .venv/bin/activate
  pip install -q -r proof/requirements.txt
fi

echo ""
echo "Done."
echo "Run Go tests:    cd openav-epiphan-pearl && go test ./..."
echo "Docker build:    docker build -t openav-epiphan-pearl ./openav-epiphan-pearl"
echo "Full stack demo: cd demo && docker compose up"
