#!/bin/bash
set -e
echo "Setting up epiphan-openav-bridge..."

# Check for required env
if [ ! -f .env ]; then
  echo "Warning: No .env file found. Required vars: PEARL_HOST, PEARL_USERNAME, PEARL_PASSWORD, EC20_HOST, EC20_USERNAME, EC20_PASSWORD"
fi

# Phase 1: Python RTSP proof dependencies (optional)
if [ -f proof/requirements.txt ]; then
  echo "Installing Python proof-of-concept deps..."
  python3 -m venv .venv
  source .venv/bin/activate
  pip install -r proof/requirements.txt
fi

# Phase 2: Go microservices
echo "Tidying Go modules..."
if [ -d openav-epiphan-pearl ]; then
  cd openav-epiphan-pearl && go mod tidy && cd ..
fi
if [ -d openav-epiphan-ec20 ]; then
  cd openav-epiphan-ec20 && go mod tidy && cd ..
fi

echo "Ready!"
echo "  Build Pearl service: cd openav-epiphan-pearl && go build -o pearl-service ."
echo "  Build EC20 service: cd openav-epiphan-ec20 && go build -o ec20-service ."
echo "  Full stack: cd demo && docker compose up"
